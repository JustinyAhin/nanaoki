package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"unicode/utf8"
)

// Render writes the report as text, md or json.
func Render(w io.Writer, r *Report, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		enc.SetEscapeHTML(false)
		return enc.Encode(r)
	case "md":
		return renderDoc(w, r, true)
	default:
		return renderDoc(w, r, false)
	}
}

func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	if s == "" {
		return "-"
	}
	return s
}

func day(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	if ts == "" {
		return "-"
	}
	return ts
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func num(n int) string {
	s := fmt.Sprint(n)
	if n < 0 {
		return "-" + num(-n)
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func pct(a, b int) string {
	if b == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(a)/float64(b))
}

func proj(p string) string {
	if p == "" {
		return "?"
	}
	return p
}

func keyLabel(k string) string {
	kind, v := splitKey(k)
	if kind == "cause" {
		return v
	}
	return "\"" + clip(v, 70) + "\""
}

var sourceName = map[string]string{
	"claude_transcript": "transcripts", "claude_history": "history.jsonl (prompts)",
	"codex_rollout": "rollouts", "codex_sqlite": "thread_history sqlite (gap fill)", "codex_history": "history.jsonl (prompts)",
}

// table writes rows as a Markdown table or as aligned text.
type table struct {
	md   bool
	w    io.Writer
	rows [][]string
}

func (t *table) row(cells ...string) { t.rows = append(t.rows, cells) }

func (t *table) flush() {
	if len(t.rows) == 0 {
		return
	}
	if t.md {
		for i, r := range t.rows {
			for j := range r {
				r[j] = strings.ReplaceAll(r[j], "|", `\|`)
			}
			fmt.Fprintf(t.w, "| %s |\n", strings.Join(r, " | "))
			if i == 0 {
				seps := make([]string, len(r))
				for j := range seps {
					seps[j] = "---"
				}
				fmt.Fprintf(t.w, "| %s |\n", strings.Join(seps, " | "))
			}
		}
		fmt.Fprintln(t.w)
	} else {
		tw := tabwriter.NewWriter(t.w, 0, 0, 2, ' ', 0)
		for _, r := range t.rows {
			fmt.Fprintf(tw, "  %s\n", strings.Join(r, "\t"))
		}
		tw.Flush()
	}
	t.rows = nil
}

func renderDoc(w io.Writer, r *Report, md bool) error {
	h := func(n int, title string) {
		if md {
			fmt.Fprintf(w, "\n## %d. %s\n\n", n, title)
		} else {
			fmt.Fprintf(w, "\n%d. %s\n", n, strings.ToUpper(title))
		}
	}
	p := func(format string, a ...any) {
		if md {
			fmt.Fprintf(w, format+"\n\n", a...)
		} else {
			fmt.Fprintf(w, "  "+format+"\n", a...)
		}
	}
	t := &table{md: md, w: w}
	c := r.Coverage
	if md {
		fmt.Fprintf(w, "# nanaoki report\n\n")
	} else {
		fmt.Fprintf(w, "nanaoki report\n")
	}
	filt := []string{}
	if r.Filter.Since != "" {
		filt = append(filt, "since "+r.Filter.Since)
	}
	if r.Filter.Until != "" {
		filt = append(filt, "until "+r.Filter.Until)
	}
	if r.Filter.Project != "" {
		filt = append(filt, "project "+r.Filter.Project)
	}
	if r.Filter.Agent != "" {
		filt = append(filt, "agent "+r.Filter.Agent)
	}
	meta := fmt.Sprintf("Store ingested %s (%s s). %d projects.", c.LastIngest, c.IngestSeconds, c.Projects)
	if len(filt) > 0 {
		meta += " Filter: " + strings.Join(filt, ", ") + "."
	}
	p("%s", meta)
	if a := r.RAF; a.Split != nil {
		p("Split: TRAIN %s to %s, TEST %s to %s (section 3 and section 4 use it).", a.Split.TrainStart, a.Split.TrainEnd, a.Split.TestStart, a.Split.TestEnd)
	}
	if r.Filters {
		var d []string
		for _, k := range []string{"scripted", "injected", "template", "probe"} {
			if r.Dropped[k] > 0 {
				d = append(d, fmt.Sprintf("%s %s", num(r.Dropped[k]), k))
			}
		}
		if len(d) > 0 {
			p("Origin filter: pasted errors that the human did not paste are left out (%s). --no-filters counts them.", strings.Join(d, ", "))
		}
	} else {
		p("No filters (--no-filters): every pasted error and every message counts, and the repeated-errors list uses 2+ days.")
	}
	p("All counts come from regex rules over local logs (no LLM). Treat them as estimates.")

	// 1
	h(1, "Coverage")
	t.row("Agent", "Source", "Sessions", "Events", "Prompts", "Tool errors", "From", "To")
	for _, s := range c.Sources {
		t.row(s.Agent, sourceName[s.Source], num(s.Sessions), num(s.Events), num(s.Prompts), num(s.ToolErrors), day(s.First), day(s.Last))
	}
	t.flush()
	for _, f := range c.Files {
		if f.Gone > 0 {
			p("%d %s files are gone from disk; their events are kept in the store.", f.Gone, f.Kind)
		}
	}
	for _, wn := range c.Warnings {
		p("WARNING: %s", wn)
	}

	// 2
	h(2, "Repeated errors")
	p("%s error keys (cause, or exact signature) occur in 2+ sessions on %d+ days: %s events. Top %d by sessions:",
		num(r.RepeatedAll.Keys), r.RepeatedAll.MinDays, num(r.RepeatedAll.Events), min(10, len(r.Repeated)))
	t.row("#", "Sessions", "Days", "Projects", "Events", "First", "Last", "Error")
	for i, it := range r.Repeated {
		if i >= 10 {
			break
		}
		t.row(fmt.Sprint(i+1), num(it.Sessions), num(it.Days), num(it.Projects), num(it.Events), day(it.FirstSeen), day(it.LastSeen), keyLabel(it.Key))
	}
	t.flush()

	// 3
	a := r.RAF
	h(3, "Repeat-after-fix")
	if a.Split != nil {
		p("Method: fixed split. Errors in TEST (%s to %s) whose cause or signature was fixed in TRAIN (%s to %s).",
			a.Split.TestStart, a.Split.TestEnd, a.Split.TrainStart, a.Split.TrainEnd)
		p("%s of %s TEST error episodes (%s) hit a cause or signature that was already fixed in TRAIN. Same project: %s. Other project only: %s.",
			num(a.Repeats), num(a.Episodes), pct(a.Repeats, a.Episodes), num(a.SameProject), num(a.OtherProject))
	} else {
		p("Method: rolling. Each error episode is compared with fixes in any earlier session (all dates). --split gives a fixed TRAIN/TEST split.")
		p("%s of %s error episodes (%s) hit a cause or signature that was already fixed in an earlier session. Same project: %s. Other project only: %s.",
			num(a.Repeats), num(a.Episodes), pct(a.Repeats, a.Episodes), num(a.SameProject), num(a.OtherProject))
	}
	var parts []string
	for _, ag := range []string{"claude", "codex"} {
		s, ok := a.ByAgent[ag]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s of %s (fixed only by the other agent: %s)", ag, num(s.Repeats), num(s.Episodes), num(s.OtherAgentOnly)))
	}
	if len(parts) > 0 {
		p("By agent: %s.", strings.Join(parts, "; "))
	}
	t.row("Repeats", "Same proj", "Other proj", "Error", "Earliest fix (session, date, project)")
	for i, it := range a.Top {
		if i >= 10 {
			break
		}
		t.row(num(it.Episodes), num(it.SameProject), num(it.OtherProject), keyLabel(it.Key),
			fmt.Sprintf("%s %s %s", short(it.EarliestFix.Session), day(it.EarliestFix.Ts), proj(it.EarliestFix.Project)))
	}
	t.flush()

	// 4
	tl := r.TimeLost
	h(4, "Estimated time lost")
	p("About %.1f h in repeat-after-fix episodes (%s of them have a measurable gap), %.1f h in all error episodes. This is an estimate.",
		tl.RepeatMinutes/60, num(tl.RepeatEpisodesTimed), tl.AllErrorMinutes/60)
	p("Method: wall time from each error to the next successful call in the same session, only when that success comes within 30 min. User wait time and token cost are not counted.")
	shown := 0
	t.row("Hours", "Repeats", "Error")
	for _, it := range topByMinutes(r.Repeated) {
		if shown >= 5 || it.MinutesLost <= 0 {
			break
		}
		t.row(fmt.Sprintf("%.1f", it.MinutesLost/60), num(it.RepeatAfterFix), keyLabel(it.Key))
		shown++
	}
	if shown > 0 {
		t.flush()
	} else {
		t.rows = nil
	}

	// 5
	h(5, "Restated rules")
	if r.Filters {
		p("Preferences you typed again in new sessions (2+ sessions). Counts are typed messages only. \"Not typed\" = command or skill calls (/name, $name) and templates; --no-filters counts them.")
	} else {
		p("Preferences you state again in new sessions (2+ sessions), all origins:")
	}
	t.row("Sessions", "Msgs", "Days", "First", "Last", "Not typed", "Rule")
	n := 0
	var other []string
	for _, it := range sortRules(r.Rules) {
		if it.Kind != "pref" {
			if it.AllMsgs > 0 {
				s := fmt.Sprintf("%s %s msgs in %s sessions", it.Label, num(it.Msgs), num(it.Sessions))
				if nt := it.AllMsgs - it.Msgs; nt > 0 {
					s += fmt.Sprintf(" (+%s not typed)", num(nt))
				}
				other = append(other, s)
			}
			continue
		}
		if it.Sessions < 2 {
			continue
		}
		nt := "-"
		if n := it.AllMsgs - it.Msgs; n > 0 {
			nt = num(n)
		}
		t.row(num(it.Sessions), num(it.Msgs), num(it.Days), day(it.First), day(it.Last), nt, it.Label)
		n++
	}
	if n > 0 {
		t.flush()
	} else {
		t.rows = nil
		p("None found.")
	}
	if len(other) > 0 {
		p("Memory signals: %s.", strings.Join(other, "; "))
	}

	// 6
	h(6, "Monthly trend")
	p("Repeat sig = tool errors whose signature first appeared in another session on another day. RAF = repeat-after-fix episodes (rolling method). Rules restated = typed messages only, unless --no-filters.")
	t.row("Month", "Tool errors", "Repeat sig", "Episodes", "RAF", "Prompts", "Rules restated")
	for _, m := range r.Monthly {
		t.row(m.Month, num(m.Errors), fmt.Sprintf("%s (%s)", num(m.RepeatSig), pct(m.RepeatSig, m.Errors)),
			num(m.Episodes), fmt.Sprintf("%s (%s)", num(m.RAF), pct(m.RAF, m.Episodes)), num(m.Prompts), num(m.RulesRestated))
	}
	t.flush()

	// 7
	h(7, "Suggested fixes")
	for i, s := range r.Suggestions {
		lbl := keyLabel(s.For)
		if strings.HasPrefix(s.For, "rule:") {
			lbl = "restated rule"
		}
		if md {
			fmt.Fprintf(w, "%d. **%s** for %s (%s): %s Where: %s.\n", i+1, s.Type, lbl, num(s.Count), s.Text, s.Target)
		} else {
			fmt.Fprintf(w, "  %d. [%s] %s (%s): %s\n     Where: %s.\n", i+1, s.Type, lbl, num(s.Count), s.Text, s.Target)
		}
	}
	if md {
		fmt.Fprintln(w)
	}
	p("Evidence: `nanaoki report --format json` lists the raw file and line for each example and fix.")
	return nil
}

func topByMinutes(items []RepeatedItem) []RepeatedItem {
	out := append([]RepeatedItem(nil), items...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].MinutesLost > out[j-1].MinutesLost; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func sortRules(rs []RuleItem) []RuleItem {
	out := append([]RuleItem(nil), rs...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Sessions > out[j-1].Sessions; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
