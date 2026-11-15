package analyze

import (
	"fmt"
	"sort"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// promptRules ports the rule counts of prompt_trends.py (restated preferences, recall requests, memory phrases).
// With typedOnly, Msgs, Sessions, Days, First and Last count only typed prompts; All* and ByOrigin count all.
func promptRules(ps []*Prompt, f Filter, typedOnly bool) []RuleItem {
	var out []RuleItem
	for j, rl := range rules.PromptRules {
		it := RuleItem{Key: rl.Key, Kind: rl.Kind, Label: rl.Label, ByAgent: map[string]int{}, ByOrigin: map[string]int{}}
		sess, days, all := map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, p := range ps {
			if !f.keep(p.Agent, p.Project, p.Ts) || !p.Has(j) {
				continue
			}
			it.AllMsgs++
			all[sk(p.Agent, p.Sid)] = true
			it.ByOrigin[p.Origin]++
			if typedOnly && !p.Typed() {
				continue
			}
			it.Msgs++
			sess[sk(p.Agent, p.Sid)] = true
			days[p.Ts[:10]] = true
			it.ByAgent[p.Agent]++
			if it.First == "" {
				it.First = p.Ts
			}
			it.Last = p.Ts
			if len(it.Examples) < 3 {
				it.Examples = append(it.Examples, Pointer{Agent: p.Agent, Session: p.Sid, Project: p.Project, Ts: p.Ts, RawFile: p.RawFile, RawLine: p.RawLine})
			}
		}
		it.Sessions, it.Days, it.AllSessions = len(sess), len(days), len(all)
		out = append(out, it)
	}
	return out
}

func month(ts string) string {
	if len(ts) >= 7 {
		return ts[:7]
	}
	return ts
}

// sigRepeats ports Q1a of repeat_after_fix.py: an error whose informative signature first appeared in another
// session on another day. It returns, per error id, whether it is such a repeat.
func sigRepeats(errs []*Err) map[int64]bool {
	type first struct{ sid, day string }
	seen := map[string]first{}
	out := map[int64]bool{}
	for _, e := range errs {
		if e.EType == "user_error_paste" || !rules.Informative(e.Sig) {
			continue
		}
		fs, ok := seen[e.Sig]
		if !ok {
			seen[e.Sig] = first{e.Sid, e.Ts[:10]}
		} else if fs.sid != e.Sid && fs.day != e.Ts[:10] {
			out[e.ID] = true
		}
	}
	return out
}

func monthly(d *Data, reps []Repeat, f Filter, typedOnly bool) []MonthRow {
	rows := map[string]*MonthRow{}
	get := func(m string) *MonthRow {
		r := rows[m]
		if r == nil {
			r = &MonthRow{Month: m}
			rows[m] = r
		}
		return r
	}
	rep := sigRepeats(d.Errs)
	for _, e := range d.Errs {
		if e.EType == "user_error_paste" || !f.keep(e.Agent, e.Project, e.Ts) {
			continue
		}
		r := get(month(e.Ts))
		r.Errors++
		if rep[e.ID] {
			r.RepeatSig++
		}
	}
	for _, rp := range reps {
		x := rp.X
		if !f.keep(x.Agent, x.Project, x.Ts) {
			continue
		}
		r := get(month(x.Ts))
		r.Episodes++
		if rp.Matched() {
			r.RAF++
		}
	}
	for _, p := range d.Prompts {
		if !f.keep(p.Agent, p.Project, p.Ts) {
			continue
		}
		r := get(month(p.Ts))
		r.Prompts++
		for j, rl := range rules.PromptRules {
			if rl.Kind == "pref" && p.Has(j) && (!typedOnly || p.Typed()) {
				r.RulesRestated++
			}
		}
	}
	var out []MonthRow
	for _, r := range rows {
		if r.Errors > 0 {
			r.RepeatShare = float64(r.RepeatSig) / float64(r.Errors)
		}
		if r.Episodes > 0 {
			r.RAFShare = float64(r.RAF) / float64(r.Episodes)
		}
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Month < out[j].Month })
	return out
}

// Causes whose repeat can be stopped before the command runs. Template only: the type of fix, not a new rule.
var hookable = map[string]string{
	"zsh_eq_quote":         "PreToolUse hook on Bash: block unquoted words that start with '=' (zsh expands =cmd).",
	"zsh_glob_nomatch":     "PreToolUse hook on Bash: flag unquoted [ ] * ? in paths and URLs (zsh no-match error).",
	"zsh_readonly_status":  "PreToolUse hook on Bash: block assignments to the zsh read-only variable 'status'.",
	"prettier_check":       "Pre-commit or PostToolUse hook: run the formatter before lint/commit.",
	"wrangler_types_stale": "Hook before check/build: run 'wrangler types' (cf-typegen) when wrangler config changed.",
	"classifier_bot_check": "PreToolUse hook on Bash: block curl with a spoofed browser User-Agent; point to the web-access route.",
	"port_in_use":          "PreToolUse hook on dev-server starts: check the port with lsof first.",
	"edit_before_read":     "PreToolUse hook on Edit/Write: require a Read of the file first.",
	"ssh_vps_user":         "PreToolUse hook on Bash: rewrite the ssh target to the working user@host.",
}

func suggest(r *Report, top int) []Suggestion {
	var out []Suggestion
	done := map[string]bool{}
	add := func(s Suggestion) {
		if done[s.For] || len(out) >= top {
			return
		}
		done[s.For] = true
		out = append(out, s)
	}
	type cand struct {
		key, kind, k    string
		count, projects int
	}
	var cs []cand
	for _, t := range r.RAF.Top {
		kind, k := splitKey(t.Key)
		cs = append(cs, cand{t.Key, kind, k, t.Episodes, 0})
	}
	for _, it := range r.Repeated {
		kind, k := splitKey(it.Key)
		cs = append(cs, cand{it.Key, kind, k, it.Sessions, it.Projects})
	}
	prj := map[string]int{}
	for _, it := range r.Repeated {
		prj[it.Key] = it.Projects
	}
	// restated preferences interleave by count
	for _, c := range cs {
		if len(out) >= top-2 {
			break
		}
		switch {
		case c.kind == "cause" && hookable[c.k] != "":
			add(Suggestion{Type: "hook", Target: "agent hooks (Claude Code settings.json, Codex hooks)", Text: hookable[c.k], For: c.key, Count: c.count})
		case c.kind == "cause" && rules.Scope[c.k] == "env":
			add(Suggestion{Type: "environment fact", Target: "global ~/.claude/CLAUDE.md and ~/.codex/AGENTS.md", Text: rules.Lesson[c.k], For: c.key, Count: c.count})
		case c.kind == "cause":
			add(Suggestion{Type: "rule", Target: "AGENTS.md of projects on this stack", Text: rules.Lesson[c.k], For: c.key, Count: c.count})
		default:
			tgt := "AGENTS.md of the project"
			if prj[c.key] >= 3 {
				tgt = fmt.Sprintf("global instructions (seen in %d projects)", prj[c.key])
			}
			add(Suggestion{Type: "rule", Target: tgt, Text: "Record this error and the command that fixed it (see the earliest fix pointer).", For: c.key, Count: c.count})
		}
	}
	prefs := append([]RuleItem(nil), r.Rules...)
	sort.SliceStable(prefs, func(i, j int) bool { return prefs[i].Sessions > prefs[j].Sessions })
	for _, p := range prefs {
		if p.Kind != "pref" || p.Sessions < 2 {
			continue
		}
		add(Suggestion{Type: "rule", Target: "global ~/.claude/CLAUDE.md and ~/.codex/AGENTS.md",
			Text: fmt.Sprintf("State once: %q (restated in %d sessions).", p.Label, p.Sessions), For: "rule:" + p.Key, Count: p.Sessions})
		if len(out) >= top {
			break
		}
	}
	return out
}

func splitKey(k string) (string, string) {
	for i := 0; i < len(k); i++ {
		if k[i] == ':' {
			return k[:i], k[i+1:]
		}
	}
	return "", k
}

// ---------- validation mode (TRAIN/TEST numbers of repeat_after_fix.py and prompt_trends.py) ----------

type Validation struct {
	Split       Split                     `json:"split"`
	Q2          map[string]any            `json:"q2"`
	SigSpread   map[string]int            `json:"sig_spread"`
	Monthly     map[string]map[string]int `json:"monthly_errors"`
	CauseSess   map[string][2]int         `json:"cause_sessions_train_test"`
	Prompts     map[string][2]int         `json:"rules_msgs_sessions"`
	PromptTotal int                       `json:"prompts"`
}

// splitErrs returns the errors in TRAIN or TEST, with Err.Split set.
func splitErrs(d *Data, s Split) []*Err {
	var errs []*Err
	for _, e := range d.Errs {
		if e.Ts < s.TrainStart {
			continue
		}
		e.Split = s.Of(e.Ts)
		if e.Split == "out" {
			continue
		}
		errs = append(errs, e)
	}
	return errs
}

func validate(d *Data, s Split) *Validation {
	errs := splitErrs(d, s)
	rows := TrainTest(Episodes(errs))
	cnt := func(fn func(Repeat) bool) int {
		n := 0
		for _, r := range rows {
			if fn(r) {
				n++
			}
		}
		return n
	}
	m := func(r Repeat) bool { return r.Matched() }
	ev := 0
	rev := 0
	for _, r := range rows {
		ev += r.X.N
		if r.Matched() {
			rev += r.X.N
		}
	}
	byAgent := map[string]any{}
	for _, a := range []string{"claude", "codex"} {
		a := a
		byAgent[a] = map[string]int{
			"test":               cnt(func(r Repeat) bool { return r.X.Agent == a }),
			"repeat":             cnt(func(r Repeat) bool { return r.X.Agent == a && m(r) }),
			"same_agent":         cnt(func(r Repeat) bool { return r.X.Agent == a && m(r) && r.SameAgent }),
			"other_agent_only":   cnt(func(r Repeat) bool { return r.X.Agent == a && m(r) && r.OtherAgentOnly }),
			"other_project_only": cnt(func(r Repeat) bool { return r.X.Agent == a && m(r) && !r.SameProject }),
		}
	}
	q2 := map[string]any{
		"test_episodes": len(rows), "test_error_events": ev, "repeat_after_fix": cnt(m), "repeat_events": rev,
		"by_sig": cnt(func(r Repeat) bool { return r.MatchSig }), "by_cause": cnt(func(r Repeat) bool { return r.MatchCause }),
		"cause_with_exact_sig":    cnt(func(r Repeat) bool { return r.MatchCause && r.ExactSig }),
		"cause_without_exact_sig": cnt(func(r Repeat) bool { return r.MatchCause && !r.ExactSig }),
		"same_project":            cnt(func(r Repeat) bool { return m(r) && r.SameProject }),
		"other_project_only":      cnt(func(r Repeat) bool { return m(r) && !r.SameProject }),
		"same_agent":              cnt(func(r Repeat) bool { return m(r) && r.SameAgent }),
		"other_agent_only":        cnt(func(r Repeat) bool { return m(r) && r.OtherAgentOnly }),
		"train_reported_only":     cnt(func(r Repeat) bool { return r.ReportedOnly }),
		"earlier_test_only":       cnt(func(r Repeat) bool { return r.EarlierOnly }),
		"no_prior":                cnt(func(r Repeat) bool { return !m(r) && !r.ReportedOnly && !r.EarlierOnly }),
		"by_agent":                byAgent,
	}
	v := &Validation{Split: s, Q2: q2, Monthly: map[string]map[string]int{}, CauseSess: map[string][2]int{}, Prompts: map[string][2]int{}}

	// Q1a monthly + sig spread
	rep := sigRepeats(errs)
	type gk struct{ tool, sig string }
	groups := map[gk][]*Err{}
	for _, e := range errs {
		if e.EType == "user_error_paste" {
			continue
		}
		mm := v.Monthly[month(e.Ts)]
		if mm == nil {
			mm = map[string]int{}
			v.Monthly[month(e.Ts)] = mm
		}
		mm["errors"]++
		mm["agent_"+e.Agent]++
		if rep[e.ID] {
			mm["repeat_sig_earlier_session_day"]++
		}
		if e.Cause != "" {
			mm["with_cause"]++
		}
		groups[gk{e.Tool, e.Sig}] = append(groups[gk{e.Tool, e.Sig}], e)
	}
	ss := map[string]int{"signatures": len(groups)}
	for _, g := range groups {
		ss["error_events"] += len(g)
		sess, days := map[string]bool{}, map[string]bool{}
		for _, e := range g {
			sess[e.Sid] = true
			days[e.Ts[:10]] = true
		}
		if len(sess) >= 2 && len(days) >= 2 {
			ss["multi_session_day_sigs"]++
			ss["multi_events"] += len(g)
		}
	}
	v.SigSpread = ss
	cs := map[string][2]map[string]bool{}
	for _, e := range errs {
		if e.Cause == "" {
			continue
		}
		c, ok := cs[e.Cause]
		if !ok {
			c = [2]map[string]bool{{}, {}}
			cs[e.Cause] = c
		}
		i := 0
		if e.Split == "test" {
			i = 1
		}
		c[i][sk(e.Agent, e.Sid)] = true
	}
	for k, c := range cs {
		v.CauseSess[k] = [2]int{len(c[0]), len(c[1])}
	}
	var ps []*Prompt
	for _, p := range d.Prompts {
		if p.Ts >= s.TrainStart {
			ps = append(ps, p)
		}
	}
	v.PromptTotal = len(ps)
	for _, it := range promptRules(ps, Filter{}, false) {
		v.Prompts[it.Key] = [2]int{it.AllMsgs, it.AllSessions}
	}
	return v
}
