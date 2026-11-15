package analyze

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Filter limits what the report shows. Earlier fixes and signatures from outside the filter still count as
// prior evidence (a fix in another project is the point of the cross-project numbers).
type Filter struct {
	Since   string `json:"since,omitempty"` // YYYY-MM-DD, inclusive
	Until   string `json:"until,omitempty"` // YYYY-MM-DD, inclusive
	Project string `json:"project,omitempty"`
	Agent   string `json:"agent,omitempty"`
}

func (f Filter) day(ts string) bool {
	d := ts
	if len(d) > 10 {
		d = d[:10]
	}
	return (f.Since == "" || d >= f.Since) && (f.Until == "" || d <= f.Until)
}

func (f Filter) keep(agent, project, ts string) bool {
	return f.day(ts) && (f.Agent == "" || f.Agent == agent) && (f.Project == "" || f.Project == project)
}

// Pointer links a number back to the raw log line.
type Pointer struct {
	Agent   string `json:"agent"`
	Session string `json:"session"`
	Project string `json:"project"`
	Ts      string `json:"ts"`
	RawFile string `json:"raw_file"`
	RawLine int64  `json:"raw_line,omitempty"`
	EventID int64  `json:"event_id"`
}

func errPtr(e *Err) Pointer {
	return Pointer{Agent: e.Agent, Session: e.Sid, Project: e.Project, Ts: e.Ts, RawFile: e.RawFile, RawLine: e.RawLine, EventID: e.ID}
}

// ---------- report types ----------

type SourceCov struct {
	Source     string `json:"source"`
	Agent      string `json:"agent"`
	Events     int    `json:"events"`
	Sessions   int    `json:"sessions"`
	Prompts    int    `json:"user_prompts"`
	ToolErrors int    `json:"tool_errors"`
	First      string `json:"first"`
	Last       string `json:"last"`
}

type FileCov struct {
	Kind  string `json:"kind"`
	Files int    `json:"files"`
	Gone  int    `json:"gone_from_disk"`
	Bytes int64  `json:"bytes"`
}

type Coverage struct {
	Sources       []SourceCov `json:"sources"`
	Files         []FileCov   `json:"files"`
	Projects      int         `json:"projects"`
	LastIngest    string      `json:"last_ingest"`
	IngestSeconds string      `json:"ingest_seconds"`
	Warnings      []string    `json:"warnings"`
}

type RepeatedItem struct {
	Key            string    `json:"key"` // cause:<id> or sig:<signature>
	Kind           string    `json:"kind"`
	Label          string    `json:"label"`
	Scope          string    `json:"scope,omitempty"`
	Events         int       `json:"events"`
	Sessions       int       `json:"sessions"`
	Days           int       `json:"days"`
	Projects       int       `json:"projects"`
	Agents         []string  `json:"agents"`
	FirstSeen      string    `json:"first_seen"`
	LastSeen       string    `json:"last_seen"`
	Fixed          int       `json:"fixed_episodes"`
	RepeatAfterFix int       `json:"repeat_after_fix_episodes"`
	MinutesLost    float64   `json:"minutes_lost_est"`
	Examples       []Pointer `json:"examples"`
}

type RAFItem struct {
	Key          string    `json:"key"`
	Label        string    `json:"label"`
	Episodes     int       `json:"episodes"`
	Events       int       `json:"events"`
	SameProject  int       `json:"same_project"`
	OtherProject int       `json:"other_project_only"`
	EarliestFix  Pointer   `json:"earliest_fix"`
	FixEvent     *Pointer  `json:"fix_event,omitempty"`
	Examples     []Pointer `json:"examples"`
}

type AgentSplit struct {
	Episodes       int `json:"episodes"`
	Repeats        int `json:"repeat_after_fix"`
	SameAgent      int `json:"same_agent_fixed"`
	OtherAgentOnly int `json:"other_agent_only"`
	OtherProject   int `json:"other_project_only"`
}

type RAF struct {
	Method       string                `json:"method"` // rolling or split
	MethodDetail string                `json:"method_detail"`
	Split        *Split                `json:"split,omitempty"` // dates when method = split
	Episodes     int                   `json:"error_episodes"`
	Events       int                   `json:"error_events"`
	Repeats      int                   `json:"repeat_after_fix"`
	RepeatEvents int                   `json:"repeat_events"`
	ByCause      int                   `json:"by_cause"`
	BySig        int                   `json:"by_signature"`
	SameProject  int                   `json:"same_project"`
	OtherProject int                   `json:"other_project_only"`
	ReportedOnly int                   `json:"prior_reported_only"`
	ByAgent      map[string]AgentSplit `json:"by_agent"`
	Top          []RAFItem             `json:"top"`
}

type TimeLost struct {
	Method              string  `json:"method"`
	RepeatMinutes       float64 `json:"repeat_after_fix_minutes"`
	AllErrorMinutes     float64 `json:"all_error_episode_minutes"`
	RepeatEpisodesTimed int     `json:"repeat_episodes_with_time"`
}

type RuleItem struct {
	Key         string         `json:"key"`
	Kind        string         `json:"kind"` // pref, recall, mem
	Label       string         `json:"label"`
	Msgs        int            `json:"msgs"`     // typed only (all origins with --no-filters)
	Sessions    int            `json:"sessions"` // typed only (all origins with --no-filters)
	AllMsgs     int            `json:"msgs_all_origins"`
	AllSessions int            `json:"sessions_all_origins"`
	ByOrigin    map[string]int `json:"msgs_by_origin"`
	Days        int            `json:"days"`
	First       string         `json:"first"`
	Last        string         `json:"last"`
	ByAgent     map[string]int `json:"by_agent"`
	Examples    []Pointer      `json:"examples"`
}

type MonthRow struct {
	Month         string  `json:"month"`
	Errors        int     `json:"tool_errors"`
	RepeatSig     int     `json:"repeat_sig_earlier_session_day"`
	RepeatShare   float64 `json:"repeat_sig_share"`
	Episodes      int     `json:"error_episodes"`
	RAF           int     `json:"repeat_after_fix_episodes"`
	RAFShare      float64 `json:"repeat_after_fix_share"`
	Prompts       int     `json:"prompts"`
	RulesRestated int     `json:"rule_restatements"`
}

type Suggestion struct {
	Type   string `json:"type"` // rule, hook, environment fact
	Target string `json:"target"`
	Text   string `json:"text"`
	For    string `json:"for"` // key of the item
	Count  int    `json:"count"`
}

// Report is the full output (also the JSON format).
type Report struct {
	Generated   string         `json:"generated"`
	Filter      Filter         `json:"filter"`
	Filters     bool           `json:"origin_and_spread_filters"` // false with --no-filters
	Dropped     map[string]int `json:"pasted_errors_dropped_by_origin,omitempty"`
	Coverage    Coverage       `json:"coverage"`
	Repeated    []RepeatedItem `json:"repeated_errors"`
	RepeatedAll struct {
		Keys    int `json:"keys"`
		Events  int `json:"events"`
		MinDays int `json:"min_days"` // spread rule: a key needs 2+ sessions on MinDays+ days
	} `json:"repeated_errors_total"`
	RAF         RAF          `json:"repeat_after_fix"`
	TimeLost    TimeLost     `json:"time_lost"`
	Rules       []RuleItem   `json:"restated_rules"`
	Monthly     []MonthRow   `json:"monthly"`
	Suggestions []Suggestion `json:"suggested_fixes"`
	Validation  *Validation  `json:"validation,omitempty"`
}

// Options for Build.
type Options struct {
	Filter Filter
	Top    int
	Paths  struct{ ClaudeDir string }
	Split  *Split // fixed TRAIN/TEST split: section 3 uses repeat_after_fix.py's method; JSON adds "validation"
	// NoFilters turns off the origin filters (pasted errors and restated rules count only what the human
	// typed) and the spread rule of the repeated-errors list. With it, the numbers equal the Python scripts.
	NoFilters bool
}

// Build computes the report.
func Build(db *sql.DB, o Options) (*Report, error) {
	if o.Top <= 0 {
		o.Top = 10
	}
	d, err := Load(db, "")
	if err != nil {
		return nil, err
	}
	d.MarkFixed()
	r := &Report{Filters: !o.NoFilters, Generated: time.Now().UTC().Format(time.RFC3339), Filter: o.Filter}
	if err := coverage(db, r, o); err != nil {
		return nil, err
	}
	r.Dropped = dropPastes(d, o.NoFilters)
	eps := Episodes(d.Errs)
	reps := Chrono(eps)
	f := o.Filter

	// time lost per episode (estimate)
	mins := map[*Episode]float64{}
	for _, x := range eps {
		mins[x] = episodeMinutes(d, x)
	}

	// ---------- repeated errors ----------
	// Spread rule: one burst (a batch run, a fan-out of subagents) puts the same error in many sessions on one
	// or two days. A key must span 3+ days to be listed. --no-filters keeps the Python rule (2+ days).
	r.RepeatedAll.MinDays = 3
	if o.NoFilters {
		r.RepeatedAll.MinDays = 2
	}
	r.Repeated, r.RepeatedAll.Keys, r.RepeatedAll.Events = repeatedErrors(reps, f, r.RepeatedAll.MinDays, mins)
	if len(r.Repeated) > o.Top*3 {
		r.Repeated = r.Repeated[:o.Top*3]
	}

	// ---------- repeat-after-fix ----------
	raf := RAF{Method: "rolling", MethodDetail: "rolling: an error episode (session x cause, or session x exact signature) whose key was fixed in any earlier session, all dates in time order; fixed = a later success with the same command head within 30 min",
		ByAgent: map[string]AgentSplit{}}
	rafRows := reps
	if o.Split != nil {
		s := *o.Split
		raf.Method, raf.Split = "split", &s
		raf.MethodDetail = "split (repeat_after_fix.py): an error episode in TEST " + s.TestStart + " to " + s.TestEnd +
			" whose key was fixed in TRAIN " + s.TrainStart + " to " + s.TrainEnd + "; fixed = a later success with the same command head within 30 min"
		rafRows = TrainTest(Episodes(splitErrs(d, s)))
	}
	minutes := func(x *Episode) float64 {
		v, ok := mins[x]
		if !ok {
			v = episodeMinutes(d, x)
			mins[x] = v
		}
		return v
	}
	type ragg struct {
		it   RAFItem
		best *Episode
	}
	rtop := map[string]*ragg{}
	var rorder []string
	var timeRep, timeAll float64
	timed := 0
	for _, rp := range rafRows {
		x := rp.X
		if !f.keep(x.Agent, x.Project, x.Ts) {
			continue
		}
		raf.Episodes++
		raf.Events += x.N
		timeAll += minutes(x)
		a := raf.ByAgent[x.Agent]
		a.Episodes++
		if rp.ReportedOnly {
			raf.ReportedOnly++
		}
		if rp.Matched() {
			raf.Repeats++
			raf.RepeatEvents += x.N
			if rp.MatchCause {
				raf.ByCause++
			} else {
				raf.BySig++
			}
			a.Repeats++
			if rp.SameAgent {
				a.SameAgent++
			} else {
				a.OtherAgentOnly++
			}
			if rp.SameProject {
				raf.SameProject++
			} else {
				raf.OtherProject++
				a.OtherProject++
			}
			timeRep += minutes(x)
			if minutes(x) > 0 {
				timed++
			}
			k := x.KeyID()
			g := rtop[k]
			if g == nil {
				g = &ragg{it: RAFItem{Key: k, Label: label(x.Kind, x.Key)}}
				rtop[k] = g
				rorder = append(rorder, k)
			}
			g.it.Episodes++
			g.it.Events += x.N
			if rp.SameProject {
				g.it.SameProject++
			} else {
				g.it.OtherProject++
			}
			if g.best == nil || rp.Earliest.Ts < g.best.Ts {
				g.best = rp.Earliest
			}
			if len(g.it.Examples) < 3 {
				g.it.Examples = append(g.it.Examples, errPtr(x.Errs[0]))
			}
		}
		raf.ByAgent[x.Agent] = a
	}
	var fixIDs []int64
	for _, k := range rorder {
		g := rtop[k]
		fe := g.best.FixedErr()
		if fe == nil {
			fe = g.best.Errs[0]
		}
		g.it.EarliestFix = errPtr(fe)
		if fe.FixID != 0 {
			p := Pointer{EventID: fe.FixID}
			g.it.FixEvent = &p
			fixIDs = append(fixIDs, fe.FixID)
		}
		raf.Top = append(raf.Top, g.it)
	}
	sort.SliceStable(raf.Top, func(i, j int) bool { return raf.Top[i].Episodes > raf.Top[j].Episodes })
	if len(raf.Top) > o.Top*3 {
		raf.Top = raf.Top[:o.Top*3]
	}
	if err := resolvePointers(db, raf.Top); err != nil {
		return nil, err
	}
	r.RAF = raf
	r.TimeLost = TimeLost{
		Method:              "estimate: for each error event, wall time to the next successful call in the same session, counted only when that success comes within 30 min; overlapping intervals in one episode are merged",
		RepeatMinutes:       round1(timeRep),
		AllErrorMinutes:     round1(timeAll),
		RepeatEpisodesTimed: timed,
	}
	for i := range r.Repeated {
		r.Repeated[i].MinutesLost = round1(r.Repeated[i].MinutesLost)
	}

	// ---------- restated rules ----------
	r.Rules = promptRules(d.Prompts, f, !o.NoFilters)

	// ---------- monthly ----------
	r.Monthly = monthly(d, reps, f, !o.NoFilters)

	// ---------- suggestions ----------
	r.Suggestions = suggest(r, o.Top)

	if o.Split != nil {
		r.Validation = validate(d, *o.Split)
	}
	return r, nil
}

// repeatedErrors groups episodes by key. It keeps keys with 2+ sessions on minDays+ days, sorted by sessions,
// then events, and returns them with the number of keys and events kept.
func repeatedErrors(reps []Repeat, f Filter, minDays int, mins map[*Episode]float64) ([]RepeatedItem, int, int) {
	type agg struct {
		it              RepeatedItem
		sess, days, prj map[string]bool
		agents          map[string]bool
	}
	groups := map[string]*agg{}
	var gorder []string
	for _, rp := range reps {
		x := rp.X
		if !f.keep(x.Agent, x.Project, x.Ts) {
			continue
		}
		k := x.KeyID()
		g := groups[k]
		if g == nil {
			g = &agg{it: RepeatedItem{Key: k, Kind: x.Kind, Label: label(x.Kind, x.Key), FirstSeen: x.Ts},
				sess: map[string]bool{}, days: map[string]bool{}, prj: map[string]bool{}, agents: map[string]bool{}}
			if x.Kind == "cause" {
				g.it.Scope = rules.Scope[x.Key]
			}
			groups[k] = g
			gorder = append(gorder, k)
		}
		g.it.Events += x.N
		g.sess[sk(x.Agent, x.Sid)] = true
		for _, e := range x.Errs {
			g.days[e.Ts[:10]] = true
			if e.Ts > g.it.LastSeen {
				g.it.LastSeen = e.Ts
			}
		}
		g.prj[x.Project] = true
		g.agents[x.Agent] = true
		if x.Outcome == "fixed" {
			g.it.Fixed++
		}
		if rp.Matched() {
			g.it.RepeatAfterFix++
			g.it.MinutesLost += mins[x]
		}
		if len(g.it.Examples) < 3 {
			g.it.Examples = append(g.it.Examples, errPtr(x.Errs[0]))
		}
	}
	var items []RepeatedItem
	nk, ne := 0, 0
	for _, k := range gorder {
		g := groups[k]
		g.it.Sessions, g.it.Days, g.it.Projects = len(g.sess), len(g.days), len(g.prj)
		g.it.Agents = keys(g.agents)
		if g.it.Sessions >= 2 && g.it.Days >= minDays {
			items = append(items, g.it)
			nk++
			ne += g.it.Events
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Sessions != items[j].Sessions {
			return items[i].Sessions > items[j].Sessions
		}
		return items[i].Events > items[j].Events
	})
	return items, nk, ne
}

// dropPastes removes pasted errors that the human did not paste: text in a side session (subagent, codex
// exec), text that a tool injected, and long templates sent to many sessions. It returns the counts by origin.
func dropPastes(d *Data, off bool) map[string]int {
	if off {
		return nil
	}
	n := map[string]int{}
	keep := d.Errs[:0]
	for _, e := range d.Errs {
		if e.EType == "user_error_paste" && e.Origin != rules.OriginTyped {
			n[e.Origin]++
			continue
		}
		// A failed probe (cat x 2>/dev/null, test -f, grep with no match) is an answer, not a mistake.
		if e.EType == "tool_error" && rules.IsProbe(e.Cmd, e.Out) {
			n["probe"]++
			continue
		}
		keep = append(keep, e)
	}
	d.Errs = keep
	return n
}

func round1(f float64) float64 { return float64(int64(f*10+0.5)) / 10 }

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func label(kind, key string) string {
	if kind == "cause" {
		return key
	}
	return key
}

func parseTs(ts string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		if len(ts) >= 19 {
			t, err = time.Parse("2006-01-02T15:04:05", ts[:19])
		}
		if err != nil {
			return time.Time{}, false
		}
	}
	return t, true
}

// episodeMinutes: union of [error, next success] intervals, each only when the success is within 30 min.
func episodeMinutes(d *Data, x *Episode) float64 {
	if x.Errs[0].EType == "user_error_paste" {
		return 0
	}
	type iv struct{ a, b time.Time }
	var ivs []iv
	for _, e := range x.Errs {
		nx, ok := d.NextOK(e.Agent, e.Sid, e.Ts, 30)
		if !ok {
			continue
		}
		a, ok1 := parseTs(e.Ts)
		b, ok2 := parseTs(nx.Ts)
		if ok1 && ok2 && b.After(a) {
			ivs = append(ivs, iv{a, b})
		}
	}
	sort.Slice(ivs, func(i, j int) bool { return ivs[i].a.Before(ivs[j].a) })
	var total time.Duration
	var cur *iv
	for i := range ivs {
		v := ivs[i]
		if cur == nil || v.a.After(cur.b) {
			if cur != nil {
				total += cur.b.Sub(cur.a)
			}
			c := v
			cur = &c
		} else if v.b.After(cur.b) {
			cur.b = v.b
		}
	}
	if cur != nil {
		total += cur.b.Sub(cur.a)
	}
	return total.Minutes()
}

func resolvePointers(db *sql.DB, items []RAFItem) error {
	st, err := db.Prepare(`select agent, coalesce(session_id,''), coalesce(project,''), coalesce(ts,''), coalesce(raw_file,''), coalesce(raw_line,0) from events where id = ?`)
	if err != nil {
		return err
	}
	defer st.Close()
	for i := range items {
		p := items[i].FixEvent
		if p == nil {
			continue
		}
		if err := st.QueryRow(p.EventID).Scan(&p.Agent, &p.Session, &p.Project, &p.Ts, &p.RawFile, &p.RawLine); err != nil && err != sql.ErrNoRows {
			return err
		}
	}
	return nil
}

// ---------- coverage ----------

func coverage(db *sql.DB, r *Report, o Options) error {
	c := &r.Coverage
	rs, err := db.Query(`select source, agent, count(*), count(distinct session_id), coalesce(sum(event_type='user_msg'),0),
		coalesce(sum(is_error=1 and event_type!='user_error_paste'),0), coalesce(min(ts),''), coalesce(max(ts),'') from events group by source, agent order by agent, source`)
	if err != nil {
		return err
	}
	for rs.Next() {
		var s SourceCov
		if err := rs.Scan(&s.Source, &s.Agent, &s.Events, &s.Sessions, &s.Prompts, &s.ToolErrors, &s.First, &s.Last); err != nil {
			rs.Close()
			return err
		}
		c.Sources = append(c.Sources, s)
	}
	rs.Close()
	rs, err = db.Query(`select kind, count(*), coalesce(sum(present=0),0), coalesce(sum(size),0) from files group by kind order by kind`)
	if err != nil {
		return err
	}
	for rs.Next() {
		var fc FileCov
		if err := rs.Scan(&fc.Kind, &fc.Files, &fc.Gone, &fc.Bytes); err != nil {
			rs.Close()
			return err
		}
		c.Files = append(c.Files, fc)
	}
	rs.Close()
	_ = db.QueryRow(`select count(distinct project) from events where project is not null`).Scan(&c.Projects)
	_ = db.QueryRow(`select coalesce(max(case when k='last_ingest' then v end),''), coalesce(max(case when k='ingest_seconds' then v end),'') from meta`).Scan(&c.LastIngest, &c.IngestSeconds)

	// Claude Code deletes transcripts after cleanupPeriodDays (default 30).
	var tr, hist SourceCov
	for _, s := range c.Sources {
		switch s.Source {
		case "claude_transcript":
			tr = s
		case "claude_history":
			hist = s
		}
	}
	cleanup := claudeCleanupDays(o.Paths.ClaudeDir)
	if tr.First != "" && hist.First != "" {
		t1, ok1 := parseTs(tr.First)
		h1, ok2 := parseTs(hist.First)
		if ok1 && ok2 && t1.Sub(h1) > 35*24*time.Hour {
			msg := "Claude transcripts start on " + tr.First[:10] + ", but Claude prompts go back to " + hist.First[:10] +
				". Older transcripts were deleted by Claude Code's cleanup, so before " + tr.First[:10] +
				" the report sees Claude prompts only, no tool errors."
			days, _ := strconv.Atoi(cleanup)
			if days >= 365 {
				msg += " cleanupPeriodDays is now " + cleanup + ", so new transcripts are kept."
			} else {
				cur := "default 30"
				if cleanup != "" {
					cur = "now " + cleanup
				}
				msg += ` To keep transcripts, set "cleanupPeriodDays": 3650 in ~/.claude/settings.json (` + cur + ")."
			}
			msg += " nanaoki keeps events of deleted files once they are ingested."
			c.Warnings = append(c.Warnings, msg)
		}
	}
	return nil
}

func claudeCleanupDays(dir string) string {
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".claude")
	}
	b, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	if v, ok := m["cleanupPeriodDays"].(float64); ok {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}
