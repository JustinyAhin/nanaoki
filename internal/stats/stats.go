// Package stats measures the live use of the plugin: lessons shown, guards fired, what the agent did next,
// notes, hook health, and a rough before/after of the repeat-after-fix rate.
package stats

import (
	"bufio"
	"database/sql"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/activity"
	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/guards"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
	"github.com/JustinyAhin/nanaoki/internal/rules"
)

const (
	NoiseMinShown = 3
	followWindow  = 10 * time.Minute
	nextOKWindow  = 30 * time.Minute
	// The transcript stamps a tool call when the model writes it, a little before the PreToolUse hook runs.
	hookSkew = 10 * time.Second
	maxNext  = 5
)

const (
	Again    = "again"     // the same failure came back in the session after the lesson
	NotAgain = "not_again" // it did not
	NoData   = "no_data"   // the session is not in the store (run nanaoki ingest) or has no later call
	NA       = "n/a"       // session-start lesson: no failure to compare

	Followed = "followed" // a later command used the fix
	Ignored  = "ignored"  // the warned command ran and did not fail the same way
	Repeated = "repeated" // the same failure happened and no fix followed
	Other    = "other"
)

type Paths struct {
	Shown   string `json:"shown"`
	Guards  string `json:"guards"`
	Calls   string `json:"calls"`
	Notes   string `json:"notes"`
	HookLog string `json:"hook_log"`
}

func DefaultPaths() Paths {
	return Paths{Shown: lessons.ShownLogPath(), Guards: activity.GuardLogPath(), Calls: activity.CallLogPath(),
		Notes: activity.NotesPath(), HookLog: activity.HookLogPath()}
}

type Options struct {
	Since, Until string // YYYY-MM-DD, inclusive; "" = open
	Paths        Paths
	Index        *lessons.Index // nil: labels from the log only
	Now          time.Time
}

type Event struct {
	ID      int64
	Ts      string
	Tool    string
	Cmd     string
	Out     string
	Err     bool
	RawFile string
	RawLine int64
}

func (e Event) shell() bool {
	switch e.Tool {
	case "Bash", "bash", "shell", "Shell", "exec_command", "local_shell":
		return true
	}
	return false
}

func (e Event) ptr() *Pointer {
	return &Pointer{EventID: e.ID, Ts: e.Ts, RawFile: e.RawFile, RawLine: e.RawLine}
}

type Pointer struct {
	EventID int64  `json:"event_id"`
	Ts      string `json:"ts"`
	RawFile string `json:"raw_file"`
	RawLine int64  `json:"raw_line,omitempty"`
}

func parse(ts string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, ts)
	return t, err == nil
}

func dayOf(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

func inWindow(ts, since, until string) bool {
	d := dayOf(ts)
	return (since == "" || d >= since) && (until == "" || d <= until)
}

func sk(agent, sid string) string { return agent + "\x00" + sid }

// LessonOutcome: after a lesson was shown for a failure, did an error with the lesson's key (cause or exact
// signature) come back in the same session? errs are the session's errors in time order. hasData is false
// when the store holds no event of the session after the show.
func LessonOutcome(L *lessons.Lesson, shownTs string, errs []*analyze.Err, hasData bool) (string, *analyze.Err) {
	t0, ok := parse(shownTs)
	if !ok || L == nil {
		return NoData, nil
	}
	for _, e := range errs {
		t, ok := parse(e.Ts)
		// the hook stamps whole seconds after the error it answers
		if !ok || !t.Truncate(time.Second).After(t0) {
			continue
		}
		if (L.Kind == "cause" && e.Cause == L.Key) || (L.Kind == "sig" && e.Sig == L.Key) {
			return Again, e
		}
	}
	if !hasData {
		return NoData, nil
	}
	return NotAgain, nil
}

// GuardOutcome classifies what happened after a guard fired. evs are the session's events in time order.
//   - The guarded command itself is the first event near the hook time with the same stored command. A
//     blocked command did not run; its event (if any) is skipped.
//   - followed: a later shell command is the suggested command, or has the same head, succeeds and no longer
//     trips the guard.
//   - repeated: the command failed with the guard's cause and no fix followed.
//   - ignored: a warned command ran without that failure.
func GuardOutcome(g activity.GuardEntry, evs []Event) (string, *Event) {
	t0, ok := parse(g.Ts)
	if !ok {
		return NoData, nil
	}
	var win []Event
	for _, e := range evs {
		t, ok := parse(e.Ts)
		if !ok || !e.shell() || t.Before(t0.Add(-hookSkew)) || t.After(t0.Add(followWindow)) {
			continue
		}
		win = append(win, e)
	}
	anchor := -1
	for i, e := range win {
		if activity.Cmd(e.Cmd) == g.Cmd {
			anchor = i
			break
		}
	}
	var rest []Event
	failed := false
	var failEv *Event
	if anchor >= 0 {
		rest = win[anchor+1:]
		a := win[anchor]
		if g.Decision != guards.Block {
			if !(a.Err && rules.CauseOf(a.Cmd, a.Out) == g.Cause) {
				return Ignored, &a
			}
			failed, failEv = true, &a
		}
	} else {
		for i, e := range win {
			if t, _ := parse(e.Ts); !t.Before(t0) {
				rest = win[i:]
				break
			}
		}
	}
	if len(rest) > maxNext {
		rest = rest[:maxNext]
	}
	head := rules.Head("Bash", g.Cmd)
	// a zsh guard fired, so the shell was zsh; Replay keeps the check off the file system and PATH
	env := &guards.Env{Shell: "zsh", Replay: true}
	for i := range rest {
		e := rest[i]
		c := activity.Cmd(e.Cmd)
		sameFail := e.Err && rules.CauseOf(e.Cmd, e.Out) == g.Cause
		if g.Fixed != "" && c == g.Fixed && !sameFail {
			return Followed, &e
		}
		if !e.Err && head != "" && rules.Head("Bash", e.Cmd) == head && c != g.Cmd && !trips(e.Cmd, g.Guard, env) {
			return Followed, &e
		}
		if sameFail && !failed {
			failed, failEv = true, &e
		}
	}
	switch {
	case failed:
		return Repeated, failEv
	case len(rest) == 0 && anchor < 0:
		return NoData, nil
	case len(rest) == 0:
		return Other, nil
	}
	return Other, &rest[0]
}

func trips(cmd, guard string, env *guards.Env) bool {
	for _, f := range guards.Check(cmd, env) {
		if f.Guard == guard {
			return true
		}
	}
	return false
}

type Window struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Episodes int     `json:"error_episodes"`
	Repeats  int     `json:"repeat_after_fix"`
	Rate     float64 `json:"rate"`
}

type Baseline struct {
	FirstUse string `json:"first_use"`
	Before   Window `json:"before"`
	After    Window `json:"after"`
	Note     string `json:"note"`
}

const baselineNote = "Rough before/after on one user's own data, not a controlled test: the work, projects and agents change between the two windows."

// BaselineOf compares the repeat-after-fix rate (the report's rolling rule) in the period after the first
// plugin use (from max(since, firstUse) to until) with the same number of days just before firstUse.
func BaselineOf(reps []analyze.Repeat, firstUse, since, until string) Baseline {
	b := Baseline{FirstUse: firstUse, Note: baselineNote}
	fu, err := time.Parse("2006-01-02", dayOf(firstUse))
	if err != nil {
		return b
	}
	from := fu
	if s, err := time.Parse("2006-01-02", since); err == nil && s.After(from) {
		from = s
	}
	to, err := time.Parse("2006-01-02", until)
	if err != nil || to.Before(from) {
		return b
	}
	days := int(to.Sub(from).Hours()/24) + 1
	bTo := fu.AddDate(0, 0, -1)
	bFrom := bTo.AddDate(0, 0, -(days - 1))
	const df = "2006-01-02"
	b.After = Window{From: from.Format(df), To: to.Format(df)}
	b.Before = Window{From: bFrom.Format(df), To: bTo.Format(df)}
	for _, r := range reps {
		for _, w := range []*Window{&b.Before, &b.After} {
			if inWindow(r.X.Ts, w.From, w.To) {
				w.Episodes++
				if r.Matched() {
					w.Repeats++
				}
			}
		}
	}
	for _, w := range []*Window{&b.Before, &b.After} {
		if w.Episodes > 0 {
			w.Rate = float64(w.Repeats) / float64(w.Episodes)
		}
	}
	return b
}

func SessionEvents(db *sql.DB, agent, sid string) ([]Event, error) {
	rs, err := db.Query(`select id, ts, coalesce(tool,''), coalesce(text,''), coalesce(substr(output,1,4000),''), is_error,
		coalesce(raw_file,''), coalesce(raw_line,0) from events where agent=? and session_id=? and ts is not null
		and event_type in ('tool_call','tool_error','test_result','build_result','deploy','commit','push','pr','file_change')
		order by ts, id`, agent, sid)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	var out []Event
	for rs.Next() {
		var e Event
		var isErr int
		if err := rs.Scan(&e.ID, &e.Ts, &e.Tool, &e.Cmd, &e.Out, &isErr, &e.RawFile, &e.RawLine); err != nil {
			return nil, err
		}
		e.Err = isErr != 0
		out = append(out, e)
	}
	return out, rs.Err()
}

func nextOK(evs []Event, ts string) (*Event, float64) {
	t0, ok := parse(ts)
	if !ok {
		return nil, 0
	}
	for i := range evs {
		t, ok := parse(evs[i].Ts)
		if !ok || !t.Truncate(time.Second).After(t0) || evs[i].Err {
			continue
		}
		if t.Sub(t0) > nextOKWindow {
			return nil, 0
		}
		return &evs[i], t.Sub(t0).Minutes()
	}
	return nil, 0
}

func hasAfter(evs []Event, ts string) bool {
	t0, ok := parse(ts)
	if !ok {
		return false
	}
	for _, e := range evs {
		if t, ok := parse(e.Ts); ok && t.Truncate(time.Second).After(t0) {
			return true
		}
	}
	return false
}

// hookLogErrors returns the hook.log lines in the period (each line starts with an RFC 3339 time).
func hookLogErrors(path, since, until string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		ts, _, _ := strings.Cut(l, " ")
		if _, ok := parse(ts); ok && inWindow(ts, since, until) {
			out = append(out, l)
		}
	}
	return out
}

func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	i := int(p*float64(len(s))+0.999999) - 1
	return s[max(0, min(i, len(s)-1))]
}
