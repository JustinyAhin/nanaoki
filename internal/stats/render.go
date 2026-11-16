package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/JustinyAhin/nanaoki/internal/style"
)

func Render(w io.Writer, r *Report, format string, st style.Style) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		enc.SetEscapeHTML(false)
		return enc.Encode(r)
	case "md":
		st = style.Style{}
	}
	md := format == "md"
	h := func(n int, title string) {
		if md {
			fmt.Fprintf(w, "\n## %d. %s\n\n", n, title)
		} else {
			fmt.Fprintf(w, "\n%s\n", st.Title(fmt.Sprintf("%d. %s", n, title)))
		}
	}
	p := func(format string, a ...any) {
		if md {
			fmt.Fprintf(w, format+"\n\n", a...)
			return
		}
		for _, l := range st.Wrap(fmt.Sprintf(format, a...), "  ", st.Width) {
			fmt.Fprintln(w, l)
		}
	}
	tbl := func(head []string, rows [][]string) {
		if len(rows) == 0 {
			return
		}
		if !md {
			st.Render(w, style.Table{Head: head, Rows: rows, Indent: 2, Flex: -1})
			return
		}
		for i, r := range append([][]string{head}, rows...) {
			for j := range r {
				r[j] = strings.ReplaceAll(r[j], "|", `\|`)
			}
			fmt.Fprintf(w, "| %s |\n", strings.Join(r, " | "))
			if i == 0 {
				fmt.Fprintf(w, "|%s\n", strings.Repeat(" --- |", len(r)))
			}
		}
		fmt.Fprintln(w)
	}
	b := st.Bold

	if md {
		fmt.Fprintf(w, "# nanaoki stats\n\n")
	} else {
		fmt.Fprintf(w, "%s\n", st.Bold("nanaoki stats"))
	}
	period := "all dates"
	if r.Since != "" {
		period = r.Since + " to " + r.Until
	}
	fu := "none yet"
	if r.FirstUse != "" {
		fu = dayOf(r.FirstUse)
	}
	p("Period: %s. First plugin use: %s. Each number has its log line or event id in --format json.", period, fu)
	if r.NoSession > 0 {
		p("%d log entries have no session id (manual hook tests) and are left out.", r.NoSession)
	}

	ls := r.Lessons
	h(1, "Lessons shown")
	p("%s shows of %s lessons: %d after a failure, %d at session start.", b(fmt.Sprint(ls.Shown)), fmt.Sprint(ls.Unique),
		ls.ByEvent["failure"], ls.ByEvent["session_start"])
	if ls.OnFailure > 0 {
		traced := ls.Again + ls.NotAgain
		p("Key metric: after a lesson for a failure, the same failure came back in that session %s of %d traced times (%s). %d not traced (session not in the store: run nanaoki ingest).",
			b(fmt.Sprint(ls.Again)), traced, pct(ls.Again, traced), ls.NoData)
		if ls.MedianNextOK > 0 {
			p("Median time from a lesson to the next successful call: %.1f min (cap 30 min).", ls.MedianNextOK)
		}
	}
	var rows [][]string
	for i, row := range ls.Rows {
		if i >= 10 {
			break
		}
		rows = append(rows, []string{fmt.Sprint(row.Shown), fmt.Sprint(row.Again), fmt.Sprint(row.NotAgain), fmt.Sprint(row.NoData),
			row.Lesson, row.Label})
	}
	tbl([]string{"Shown", "Again", "Not again", "No data", "Lesson", "Text"}, rows)

	g := r.Guards
	h(2, "Guards")
	p("%s fired: %d warn, %d block. Followed %d, ignored %d, repeated failure %d, other %d, no data %d.",
		b(fmt.Sprint(g.Fired)), g.Warn, g.Block, g.ByOutcome[Followed], g.ByOutcome[Ignored], g.ByOutcome[Repeated],
		g.ByOutcome[Other], g.ByOutcome[NoData])
	if !md && g.Fired > 0 {
		p("%s", st.Dim("followed = a later command used the fix; ignored = the warned command ran and did not fail that way; repeated = it failed the same way and no fix followed."))
	}
	rows = nil
	for _, row := range g.Rows {
		rows = append(rows, []string{row.Guard, fmt.Sprint(row.Fired), fmt.Sprint(row.Block), fmt.Sprint(row.ByOutcome[Followed]),
			fmt.Sprint(row.ByOutcome[Ignored]), fmt.Sprint(row.ByOutcome[Repeated]), fmt.Sprint(row.ByOutcome[Other] + row.ByOutcome[NoData])})
	}
	tbl([]string{"Guard", "Fired", "Block", "Followed", "Ignored", "Repeated", "Other or no data"}, rows)

	h(3, "Noise candidates")
	if len(r.Noise) == 0 {
		p("None: no lesson or guard shown %d+ times with no effect, and no note tagged --noise or --wrong.", NoiseMinShown)
	}
	rows = nil
	for _, n := range r.Noise {
		why := n.Why
		if len(n.Notes) > 0 {
			why += " (" + strings.Join(n.Notes, ", ") + ")"
		}
		rows = append(rows, []string{n.Kind, n.ID, fmt.Sprint(n.Shown), why, n.Label})
	}
	tbl([]string{"Kind", "ID", "Shown", "Why", "Text"}, rows)

	m := r.Missing
	h(4, "Missing")
	p("%d sessions ran the hook. In them, %d errors repeated a fix from an earlier session; a lesson or guard for the same key showed in %d. %s got none. Notes tagged --missing: %d.",
		m.ActiveSessions, m.Repeats, m.Covered, b(fmt.Sprint(len(m.Items))), len(m.Notes))
	rows = nil
	cnt := map[string]int{}
	byKey := map[string]MissingItem{}
	for _, it := range m.Items {
		cnt[it.Key]++
		byKey[it.Key] = it
	}
	var keys []string
	for k := range cnt {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if cnt[keys[i]] != cnt[keys[j]] {
			return cnt[keys[i]] > cnt[keys[j]]
		}
		return keys[i] < keys[j]
	})
	for i, k := range keys {
		if i >= 10 {
			break
		}
		it := byKey[k]
		in := "yes"
		if !it.InIndex {
			in = "no"
		}
		rows = append(rows, []string{fmt.Sprint(cnt[k]), in, it.Label})
	}
	tbl([]string{"Episodes", "In store", "Error"}, rows)

	hl := r.Health
	h(5, "Health")
	p("%s hook calls, %d with output, %d errors; hook.log lines in the period: %d. Time in the hook process: p50 %.1f ms, p95 %s ms, max %.1f ms (process start not included).",
		b(fmt.Sprint(hl.Calls)), hl.WithOutput, hl.Errors, len(hl.HookLogLines), hl.P50, b(fmt.Sprintf("%.1f", hl.P95)), hl.Max)
	for i, l := range hl.HookLogLines {
		if i >= 5 {
			p("... %d more in hook.log", len(hl.HookLogLines)-5)
			break
		}
		p("%s", clip(l, 160))
	}
	if len(hl.Failures) > 0 {
		var rows [][]string
		for _, f := range hl.Failures {
			rows = append(rows, []string{fmt.Sprint(f.Count), f.Hook, clip(f.Command, 60), clip(f.Reason, 50)})
		}
		p("Hooks that failed in the transcripts (any hook, not only nanaoki):")
		tbl([]string{"Times", "Hook", "Command", "Last reason"}, rows)
	}

	bl := r.Baseline
	h(6, "Baseline (rough before/after)")
	if bl.FirstUse == "" || bl.After.From == "" {
		p("No baseline: no plugin use in the activity logs yet.")
	} else {
		tbl([]string{"Window", "From", "To", "Error episodes", "Repeat-after-fix", "Rate"}, [][]string{
			{"before plugin", bl.Before.From, bl.Before.To, fmt.Sprint(bl.Before.Episodes), fmt.Sprint(bl.Before.Repeats), pct(bl.Before.Repeats, bl.Before.Episodes)},
			{"with plugin", bl.After.From, bl.After.To, fmt.Sprint(bl.After.Episodes), fmt.Sprint(bl.After.Repeats), pct(bl.After.Repeats, bl.After.Episodes)},
		})
	}
	p("%s", bl.Note)

	h(7, "Notes")
	if len(r.Notes) == 0 {
		p("No notes in the period. Add one with: nanaoki note \"text\" [--noise|--useful|--wrong|--missing] [--lesson ID|--guard ID]")
	}
	head, rows := noteRows(r.Notes)
	tbl(head, rows)
	return nil
}

func noteRows(ns []NoteRow) ([]string, [][]string) {
	var rows [][]string
	for _, n := range ns {
		ref := strings.TrimSpace(n.Lesson + " " + n.Guard)
		rows = append(rows, []string{n.ID, dayOf(n.Ts), dash(n.Tag), dash(n.Project), dash(ref), strings.Join(strings.Fields(n.Text), " ")})
	}
	return []string{"ID", "Date", "Tag", "Project", "About", "Text"}, rows
}

func RenderNotes(w io.Writer, ns []NoteRow, st style.Style) {
	if len(ns) == 0 {
		fmt.Fprintln(w, "No notes.")
		return
	}
	head, rows := noteRows(ns)
	st.Render(w, style.Table{Head: head, Rows: rows, Flex: -1})
}

func pct(a, b int) string {
	if b == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(a)/float64(b))
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
