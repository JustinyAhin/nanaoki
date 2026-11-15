// Package analyze computes the report from the event store, with the same rules as the research prototype.
package analyze

import (
	"database/sql"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Err is one error event (is_error=1).
type Err struct {
	ID                                                     int64
	Source, Agent, Sid, Project, Ts, EType, Tool, Cmd, Out string
	Sig, RawFile                                           string
	RawLine                                                int64
	Side                                                   int
	Cause, Head, Outcome, Split                            string
	Origin                                                 string // user_error_paste only: origin of the message (rules.Origin*)
	FixID                                                  int64
	causeOK                                                bool
}

// OK is one successful call (is_error=0, command-like event type).
type OK struct {
	Ts, Head string
	ID       int64
}

// Prompt is one user prompt after the prompt_trends filters.
type Prompt struct {
	Source, Agent, Sid, Project, Ts, Text, RawFile string
	RawLine                                        int64
	Rules                                          uint64 // bit j = rules.PromptRules[j] matches
	Origin                                         string // rules.Origin*
}

// Typed reports whether the human wrote the prompt (not a command call, template or injected text).
func (p *Prompt) Typed() bool { return p.Origin == rules.OriginTyped }

func (p *Prompt) Has(j int) bool { return p.Rules&(1<<uint(j)) != 0 }

type Data struct {
	Errs    []*Err
	OK      map[string][]OK // session key -> sorted by (ts, head)
	Prompts []*Prompt
}

var cmdTypes = []string{"tool_call", "tool_error", "test_result", "build_result", "deploy", "commit", "push", "pr", "file_change"}

func sk(agent, sid string) string { return agent + "\x00" + sid }

func nz(s sql.NullString) string { return s.String }

// Load reads errors, successful calls and prompts. errSince limits errors to ts >= errSince ("" = all).
func Load(db *sql.DB, errSince string) (*Data, error) {
	d := &Data{OK: map[string][]OK{}}
	var ver string
	_ = db.QueryRow(`select v from meta where k='rules_version'`).Scan(&ver)
	rulesCurrent := ver == rules.Version
	rs, err := db.Query(`select id, source, agent, session_id, project, ts, event_type, tool, text, output, sig, side, raw_file, raw_line, cause
		from events where is_error=1 and ts is not null and ts >= ?`, errSince)
	if err != nil {
		return nil, err
	}
	for rs.Next() {
		var e Err
		var sid, proj, tool, text, out, sig, rf, cause sql.NullString
		var rl sql.NullInt64
		if err := rs.Scan(&e.ID, &e.Source, &e.Agent, &sid, &proj, &e.Ts, &e.EType, &tool, &text, &out, &sig, &e.Side, &rf, &rl, &cause); err != nil {
			rs.Close()
			return nil, err
		}
		e.Sid, e.Project, e.Tool, e.Cmd, e.Out, e.Sig, e.RawFile, e.RawLine = nz(sid), nz(proj), nz(tool), nz(text), nz(out), nz(sig), nz(rf), rl.Int64
		e.Cause, e.causeOK = cause.String, cause.Valid && rulesCurrent
		d.Errs = append(d.Errs, &e)
	}
	rs.Close()
	sort.Slice(d.Errs, func(i, j int) bool {
		if d.Errs[i].Ts != d.Errs[j].Ts {
			return d.Errs[i].Ts < d.Errs[j].Ts
		}
		return d.Errs[i].ID < d.Errs[j].ID
	})
	parFor(len(d.Errs), func(i int) {
		e := d.Errs[i]
		if !e.causeOK {
			e.Cause = rules.CauseOfEvent(e.EType, e.Cmd, e.Out)
		}
		e.Head = rules.Head(e.Tool, e.Cmd)
	})

	q := `select id, agent, session_id, ts, tool, text from events where is_error=0 and event_type in ('` + strings.Join(cmdTypes, "','") + `')`
	rs, err = db.Query(q)
	if err != nil {
		return nil, err
	}
	for rs.Next() {
		var id int64
		var agent string
		var sid, ts, tool, text sql.NullString
		if err := rs.Scan(&id, &agent, &sid, &ts, &tool, &text); err != nil {
			rs.Close()
			return nil, err
		}
		k := sk(agent, sid.String)
		d.OK[k] = append(d.OK[k], OK{Ts: ts.String, Head: rules.Head(tool.String, text.String), ID: id})
	}
	rs.Close()
	for _, v := range d.OK {
		sort.Slice(v, func(i, j int) bool {
			if v[i].Ts != v[j].Ts {
				return v[i].Ts < v[j].Ts
			}
			return v[i].Head < v[j].Head
		})
	}

	// User messages: all sides, for origins (templates, pastes); prompts: side 0 after the prompt_trends filters.
	rs, err = db.Query(`select source, agent, session_id, cwd, ts, text, raw_file, raw_line, side from events
		where event_type='user_msg' and ts is not null order by ts, id`) // uses ix_ts; prompts are few
	if err != nil {
		return nil, err
	}
	var msgs []Msg
	seen := map[string]bool{}
	for rs.Next() {
		var p Prompt
		var sid, cwd, text, rf sql.NullString
		var rl sql.NullInt64
		var side int
		if err := rs.Scan(&p.Source, &p.Agent, &sid, &cwd, &p.Ts, &text, &rf, &rl, &side); err != nil {
			rs.Close()
			return nil, err
		}
		msgs = append(msgs, Msg{Agent: p.Agent, Sid: sid.String, Text: text.String, RawFile: rf.String, RawLine: rl.Int64, Side: side})
		if side != 0 {
			continue
		}
		t, k, ok := rules.PromptKeep(text.String)
		if !ok {
			continue
		}
		key := p.Agent + "\x00" + sid.String + "\x00" + k
		if seen[key] {
			continue
		}
		seen[key] = true
		p.Sid, p.Project, p.Text, p.RawFile, p.RawLine = sid.String, rules.Project(cwd.String), t, rf.String, rl.Int64
		d.Prompts = append(d.Prompts, &p)
	}
	rs.Close()
	parFor(len(d.Prompts), func(i int) {
		p := d.Prompts[i]
		body := rules.PromptBody(p.Text)
		for j, rl := range rules.PromptRules {
			if rl.Rx.MatchString(body) {
				p.Rules |= 1 << uint(j)
			}
		}
	})
	ResolveOrigins(d, msgs)
	return d, nil
}

// Msg is one stored user message (any side), used to find the origin of prompts and pasted errors.
type Msg struct {
	Agent, Sid, Text, RawFile string
	RawLine                   int64
	Side                      int
}

// ResolveOrigins sets Prompt.Origin and, for pasted errors, Err.Origin (rules.Origin*).
//   - scripted: the message is in a side session (subagent, codex exec, guardian review);
//   - injected: a tool wrote it (rules.MessageOrigin);
//   - command: a prompt that starts with a slash command or skill call (prompts only; a paste after a
//     command is still a typed paste);
//   - template: the same long body (rules.TemplateKey) in rules.TemplateMinSessions+ sessions;
//   - typed: everything else.
func ResolveOrigins(d *Data, msgs []Msg) {
	tmpl := map[string]map[string]bool{}
	byRaw := make(map[string]*Msg, len(msgs))
	tkeys := make([]string, len(msgs))
	parFor(len(msgs), func(i int) { tkeys[i] = rules.TemplateKey(rules.Strip(msgs[i].Text)) })
	for i := range msgs {
		m := &msgs[i]
		byRaw[rawKey(m.RawFile, m.RawLine)] = m
		if k := tkeys[i]; k != "" {
			if tmpl[k] == nil {
				tmpl[k] = map[string]bool{}
			}
			tmpl[k][sk(m.Agent, m.Sid)] = true
		}
	}
	isTemplate := func(text string) bool {
		k := rules.TemplateKey(rules.Strip(text))
		return k != "" && len(tmpl[k]) >= rules.TemplateMinSessions
	}
	parFor(len(d.Prompts), func(i int) {
		p := d.Prompts[i]
		p.Origin = rules.MessageOrigin(p.Text)
		if p.Origin == rules.OriginTyped && isTemplate(p.Text) {
			p.Origin = rules.OriginTemplate
		}
	})
	for _, e := range d.Errs {
		if e.EType != "user_error_paste" {
			continue
		}
		e.Origin = rules.OriginTyped
		m := byRaw[rawKey(e.RawFile, e.RawLine)]
		switch {
		case e.Side != 0 || (m != nil && m.Side != 0):
			e.Origin = rules.OriginScripted
		case m == nil:
		case rules.MessageOrigin(m.Text) == rules.OriginInjected:
			e.Origin = rules.OriginInjected
		case isTemplate(m.Text):
			e.Origin = rules.OriginTemplate
		}
	}
}

func rawKey(file string, line int64) string { return file + ":" + strconv.FormatInt(line, 10) }

func parFor(n int, fn func(i int)) {
	w := runtime.NumCPU()
	var wg sync.WaitGroup
	next := int64(-1)
	for k := 0; k < w; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(atomic.AddInt64(&next, 1))
				if i >= n {
					return
				}
				fn(i)
			}
		}()
	}
	wg.Wait()
}

func addMin(ts string, m int) string {
	if len(ts) < 19 {
		return ts
	}
	t, err := time.Parse("2006-01-02T15:04:05", ts[:19])
	if err != nil {
		return ts
	}
	return t.Add(time.Duration(m) * time.Minute).Format("2006-01-02T15:04:05")
}

func cut19(s string) string {
	if len(s) > 19 {
		return s[:19]
	}
	return s
}

// MarkFixed: fixed = a later success with the same head in the same session within 30 min.
func (d *Data) MarkFixed() {
	for _, e := range d.Errs {
		if e.EType == "user_error_paste" {
			e.Outcome = "reported"
			continue
		}
		lst := d.OK[sk(e.Agent, e.Sid)]
		i := sort.Search(len(lst), func(i int) bool { return lst[i].Ts > e.Ts })
		lim := addMin(e.Ts, 30)
		e.Outcome = "open"
		for ; i < len(lst) && cut19(lst[i].Ts) <= lim; i++ {
			if lst[i].Head == e.Head && e.Head != "" {
				e.Outcome = "fixed"
				e.FixID = lst[i].ID
				break
			}
		}
	}
}

// NextOK returns the first success in the session after ts, within capMin minutes.
func (d *Data) NextOK(agent, sid, ts string, capMin int) (OK, bool) {
	lst := d.OK[sk(agent, sid)]
	i := sort.Search(len(lst), func(i int) bool { return lst[i].Ts > ts })
	if i < len(lst) && cut19(lst[i].Ts) <= addMin(ts, capMin) {
		return lst[i], true
	}
	return OK{}, false
}
