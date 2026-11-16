package stats

import (
	"database/sql"
	"sort"
	"strings"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/activity"
	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/anon"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
)

type Report struct {
	Generated string     `json:"generated"`
	Since     string     `json:"since,omitempty"`
	Until     string     `json:"until"`
	FirstUse  string     `json:"first_use,omitempty"`
	Files     Paths      `json:"files"`
	NoSession int        `json:"entries_without_session"` // left out: hook calls with no session id (manual tests)
	Lessons   LessonSec  `json:"lessons"`
	Guards    GuardSec   `json:"guards"`
	Noise     []Noise    `json:"noise_candidates"`
	Missing   MissingSec `json:"missing"`
	Health    Health     `json:"health"`
	Baseline  Baseline   `json:"baseline"`
	Notes     []NoteRow  `json:"notes"`
}

type LessonShow struct {
	Line      int      `json:"line"` // line in files.shown
	Ts        string   `json:"ts"`
	Lesson    string   `json:"lesson"`
	Event     string   `json:"event"`
	Agent     string   `json:"agent"`
	Session   string   `json:"session"`
	Project   string   `json:"project"`
	Outcome   string   `json:"outcome"`
	Again     *Pointer `json:"again_event,omitempty"`
	NextOK    *Pointer `json:"next_success,omitempty"`
	NextOKMin float64  `json:"next_success_min,omitempty"`
}

type LessonRow struct {
	Lesson   string `json:"lesson"`
	Label    string `json:"label"`
	Shown    int    `json:"shown"`
	Again    int    `json:"again"`
	NotAgain int    `json:"not_again"`
	NoData   int    `json:"no_data"`
	Lines    []int  `json:"lines"`
}

type LessonSec struct {
	Shown        int            `json:"shown"`
	Unique       int            `json:"unique"`
	ByEvent      map[string]int `json:"by_event"`
	OnFailure    int            `json:"on_failure"`
	Again        int            `json:"again"`
	NotAgain     int            `json:"not_again"`
	NoData       int            `json:"no_data"`
	MedianNextOK float64        `json:"median_next_success_min"`
	Rows         []LessonRow    `json:"by_lesson"`
	Shows        []LessonShow   `json:"shows"`
}

type GuardEvent struct {
	Line     int      `json:"line"` // line in files.guards
	Ts       string   `json:"ts"`
	Guard    string   `json:"guard"`
	Decision string   `json:"decision"`
	Agent    string   `json:"agent"`
	Session  string   `json:"session"`
	Project  string   `json:"project"`
	Cmd      string   `json:"cmd"`
	Fixed    string   `json:"fixed,omitempty"`
	Outcome  string   `json:"outcome"`
	Evidence *Pointer `json:"evidence,omitempty"`
}

type GuardRow struct {
	Guard     string         `json:"guard"`
	Fired     int            `json:"fired"`
	Block     int            `json:"block"`
	ByOutcome map[string]int `json:"by_outcome"`
	Lines     []int          `json:"lines"`
}

type GuardSec struct {
	Fired     int            `json:"fired"`
	Warn      int            `json:"warn"`
	Block     int            `json:"block"`
	ByOutcome map[string]int `json:"by_outcome"`
	Rows      []GuardRow     `json:"by_guard"`
	Events    []GuardEvent   `json:"events"`
}

type Noise struct {
	Kind  string   `json:"kind"` // lesson or guard
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Shown int      `json:"shown"`
	Why   string   `json:"why"`
	Lines []int    `json:"lines,omitempty"`
	Notes []string `json:"notes,omitempty"`
}

type MissingItem struct {
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Lesson   string          `json:"lesson"`
	InIndex  bool            `json:"in_index"` // false: the lesson store had no lesson for it (rebuild lessons)
	Error    analyze.Pointer `json:"error"`
	FixedRef analyze.Pointer `json:"earlier_fix"`
}

type MissingSec struct {
	ActiveSessions int           `json:"sessions_with_hook"`
	Repeats        int           `json:"repeat_after_fix_in_hook_sessions"`
	Covered        int           `json:"covered"` // a lesson or a guard for the key showed in the session
	Items          []MissingItem `json:"items"`
	Notes          []string      `json:"notes"`
}

type Health struct {
	Calls        int            `json:"calls"`
	ByEvent      map[string]int `json:"by_event"`
	WithOutput   int            `json:"with_output"`
	Errors       int            `json:"errors"`
	HookLogLines []string       `json:"hook_log"`
	Failures     []HookFailure  `json:"hook_failures"` // any hook (not only nanaoki) that timed out or failed, from the transcripts
	P50          float64        `json:"p50_ms"`
	P95          float64        `json:"p95_ms"`
	Max          float64        `json:"max_ms"`
	MsNote       string         `json:"ms_note"`
}

type HookFailure struct {
	Hook    string   `json:"hook"`
	Command string   `json:"command"`
	Reason  string   `json:"last_reason"`
	Count   int      `json:"count"`
	Last    *Pointer `json:"last_event"`
}

type NoteRow struct {
	Line int `json:"line"` // line in files.notes
	activity.Note
}

// Build computes the stats. db may be nil (no store yet): outcomes are then no_data and the baseline is empty.
func Build(db *sql.DB, o Options) (*Report, error) {
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	if o.Until == "" {
		o.Until = o.Now.Format("2006-01-02")
	}
	r := &Report{Generated: o.Now.UTC().Format(time.RFC3339), Since: o.Since, Until: o.Until, Files: o.Paths,
		Lessons: LessonSec{ByEvent: map[string]int{}}, Guards: GuardSec{ByOutcome: map[string]int{}},
		Health: Health{ByEvent: map[string]int{}, MsNote: "time inside the hook process; process start is not included"}}

	shown, err := activity.Read[lessons.ShownEntry](o.Paths.Shown)
	if err != nil {
		return nil, err
	}
	gs, err := activity.Read[activity.GuardEntry](o.Paths.Guards)
	if err != nil {
		return nil, err
	}
	calls, err := activity.Read[activity.CallEntry](o.Paths.Calls)
	if err != nil {
		return nil, err
	}
	notes, err := activity.Read[activity.Note](o.Paths.Notes)
	if err != nil {
		return nil, err
	}

	first := func(ts, sid string) {
		if sid != "" && ts != "" && (r.FirstUse == "" || ts < r.FirstUse) {
			r.FirstUse = ts
		}
	}
	// Only real hook calls start the live test: shown.jsonl can hold entries from manual tests before calls.jsonl existed.
	for _, l := range calls {
		first(l.V.Ts, l.V.Session)
	}
	in := func(ts string) bool { return inWindow(ts, o.Since, o.Until) }

	byID := map[string]*lessons.Lesson{}
	if o.Index != nil {
		for _, L := range o.Index.Lessons {
			byID[L.ID] = L
		}
	}

	var data *analyze.Data
	var reps []analyze.Repeat
	errsBy := map[string][]*analyze.Err{}
	if db != nil {
		data, reps, err = analyze.History(db)
		if err != nil {
			return nil, err
		}
		for _, e := range data.Errs {
			k := sk(e.Agent, e.Sid)
			errsBy[k] = append(errsBy[k], e)
		}
	}
	evCache := map[string][]Event{}
	events := func(agent, sid string) []Event {
		k := sk(agent, sid)
		if v, ok := evCache[k]; ok || db == nil {
			return v
		}
		v, _ := SessionEvents(db, agent, sid)
		evCache[k] = v
		return v
	}

	shownIn := map[string]map[string]bool{} // session -> lesson ids shown (any time: the hook shows a lesson once)
	rows := map[string]*LessonRow{}
	var nextMins []float64
	for _, l := range shown {
		e := l.V
		if e.Session == "" {
			if in(e.Ts) {
				r.NoSession++
			}
			continue
		}
		k := sk(e.Agent, e.Session)
		if shownIn[k] == nil {
			shownIn[k] = map[string]bool{}
		}
		shownIn[k][e.Lesson] = true
		if !in(e.Ts) {
			continue
		}
		s := LessonShow{Line: l.N, Ts: e.Ts, Lesson: e.Lesson, Event: e.Event, Agent: e.Agent, Session: e.Session, Project: e.Project}
		evs := events(e.Agent, e.Session)
		if ev, m := nextOK(evs, e.Ts); ev != nil {
			s.NextOK, s.NextOKMin = ev.ptr(), m
			nextMins = append(nextMins, m)
		}
		row := rows[e.Lesson]
		if row == nil {
			row = &LessonRow{Lesson: e.Lesson, Label: labelOf(byID[e.Lesson], e.Lesson)}
			rows[e.Lesson] = row
		}
		row.Shown++
		row.Lines = append(row.Lines, l.N)
		ls := &r.Lessons
		ls.Shown++
		ls.ByEvent[eventKind(e.Event)]++
		if e.Event == "UserPromptSubmit" || e.Event == "SessionStart" {
			s.Outcome = NA
		} else {
			L := byID[e.Lesson]
			if L == nil {
				L = lessonFromID(e.Lesson)
			}
			out, ae := LessonOutcome(L, e.Ts, errsBy[k], hasAfter(evs, e.Ts))
			s.Outcome = out
			if ae != nil {
				s.Again = &Pointer{EventID: ae.ID, Ts: ae.Ts, RawFile: ae.RawFile, RawLine: ae.RawLine}
			}
			ls.OnFailure++
			switch out {
			case Again:
				ls.Again++
				row.Again++
			case NotAgain:
				ls.NotAgain++
				row.NotAgain++
			default:
				ls.NoData++
				row.NoData++
			}
		}
		ls.Shows = append(ls.Shows, s)
	}
	r.Lessons.Unique = len(rows)
	r.Lessons.MedianNextOK = percentile(nextMins, 0.5)
	for _, row := range rows {
		r.Lessons.Rows = append(r.Lessons.Rows, *row)
	}
	sort.SliceStable(r.Lessons.Rows, func(i, j int) bool {
		a, b := r.Lessons.Rows[i], r.Lessons.Rows[j]
		if a.Shown != b.Shown {
			return a.Shown > b.Shown
		}
		return a.Lesson < b.Lesson
	})

	guardIn := map[string]map[string]bool{} // session -> causes a guard warned about
	grows := map[string]*GuardRow{}
	for _, l := range gs {
		g := l.V
		if g.Session == "" {
			if in(g.Ts) {
				r.NoSession++
			}
			continue
		}
		k := sk(g.Agent, g.Session)
		if guardIn[k] == nil {
			guardIn[k] = map[string]bool{}
		}
		guardIn[k][g.Cause] = true
		if !in(g.Ts) {
			continue
		}
		out, ev := GuardOutcome(g, events(g.Agent, g.Session))
		ge := GuardEvent{Line: l.N, Ts: g.Ts, Guard: g.Guard, Decision: g.Decision, Agent: g.Agent, Session: g.Session,
			Project: g.Project, Cmd: g.Cmd, Fixed: g.Fixed, Outcome: out}
		if ev != nil {
			ge.Evidence = ev.ptr()
		}
		r.Guards.Events = append(r.Guards.Events, ge)
		r.Guards.Fired++
		if g.Decision == "block" {
			r.Guards.Block++
		} else {
			r.Guards.Warn++
		}
		r.Guards.ByOutcome[out]++
		row := grows[g.Guard]
		if row == nil {
			row = &GuardRow{Guard: g.Guard, ByOutcome: map[string]int{}}
			grows[g.Guard] = row
		}
		row.Fired++
		if g.Decision == "block" {
			row.Block++
		}
		row.ByOutcome[out]++
		row.Lines = append(row.Lines, l.N)
	}
	for _, row := range grows {
		r.Guards.Rows = append(r.Guards.Rows, *row)
	}
	sort.SliceStable(r.Guards.Rows, func(i, j int) bool { return r.Guards.Rows[i].Fired > r.Guards.Rows[j].Fired })

	tagged := map[string][]string{} // lesson or guard id -> note ids tagged noise or wrong
	for _, l := range notes {
		n := l.V
		if !in(n.Ts) {
			continue
		}
		r.Notes = append(r.Notes, NoteRow{l.N, n})
		if n.Tag == "noise" || n.Tag == "wrong" {
			for _, id := range []string{n.Lesson, n.Guard} {
				if id != "" {
					tagged[id] = append(tagged[id], n.ID)
				}
			}
		}
		if n.Tag == "missing" {
			r.Missing.Notes = append(r.Missing.Notes, n.ID)
		}
	}

	seenNoise := map[string]bool{}
	for _, row := range r.Lessons.Rows {
		traced := row.Again + row.NotAgain
		var why []string
		if row.Shown >= NoiseMinShown && traced > 0 && row.NotAgain == 0 {
			why = append(why, "the same failure came back after every traced show")
		}
		if ids := tagged[row.Lesson]; len(ids) > 0 {
			why = append(why, "tagged in notes")
		}
		if len(why) > 0 {
			seenNoise[row.Lesson] = true
			r.Noise = append(r.Noise, Noise{Kind: "lesson", ID: row.Lesson, Label: row.Label, Shown: row.Shown,
				Why: strings.Join(why, "; "), Lines: row.Lines, Notes: tagged[row.Lesson]})
		}
	}
	for _, row := range r.Guards.Rows {
		traced := row.Fired - row.ByOutcome[NoData]
		var why []string
		if row.Fired >= NoiseMinShown && traced > 0 && row.ByOutcome[Followed] == 0 {
			why = append(why, "no traced event used the fix")
		}
		if ids := tagged[row.Guard]; len(ids) > 0 {
			why = append(why, "tagged in notes")
		}
		if len(why) > 0 {
			seenNoise[row.Guard] = true
			r.Noise = append(r.Noise, Noise{Kind: "guard", ID: row.Guard, Label: row.Guard, Shown: row.Fired,
				Why: strings.Join(why, "; "), Lines: row.Lines, Notes: tagged[row.Guard]})
		}
	}
	for id, ids := range tagged {
		if !seenNoise[id] {
			kind := "guard"
			if strings.Contains(id, ":") {
				kind = "lesson"
			}
			r.Noise = append(r.Noise, Noise{Kind: kind, ID: id, Label: labelOf(byID[id], id), Why: "tagged in notes", Notes: ids})
		}
	}
	sort.SliceStable(r.Noise, func(i, j int) bool { return r.Noise[i].Shown > r.Noise[j].Shown })

	active := map[string]bool{}
	var ms []float64
	for _, l := range calls {
		c := l.V
		if c.Session != "" {
			active[sk(c.Agent, c.Session)] = true
		}
		if !in(c.Ts) {
			continue
		}
		r.Health.Calls++
		r.Health.ByEvent[c.Event]++
		if c.Out {
			r.Health.WithOutput++
		}
		if c.Err {
			r.Health.Errors++
		}
		ms = append(ms, c.Ms)
	}
	r.Health.HookLogLines = hookLogErrors(o.Paths.HookLog, o.Since, o.Until)
	if db != nil {
		if r.Health.Failures, err = hookFailures(db, o.Since, o.Until); err != nil {
			return nil, err
		}
	}
	r.Health.P50, r.Health.P95, r.Health.Max = percentile(ms, 0.5), percentile(ms, 0.95), percentile(ms, 1)

	// missing: repeat-after-fix episodes in sessions where the hook ran, with no lesson and no guard for the key
	for k := range shownIn {
		active[k] = true
	}
	for k := range guardIn {
		active[k] = true
	}
	r.Missing.ActiveSessions = len(active)
	for _, rp := range reps {
		x := rp.X
		k := sk(x.Agent, x.Sid)
		if !rp.Matched() || !active[k] || !in(x.Ts) || x.Errs[0].EType == "user_error_paste" {
			continue
		}
		r.Missing.Repeats++
		id := x.Kind + ":" + lessons.ShortHash(x.Kind+x.Key)
		if shownIn[k][id] || guardIn[k][x.Cause] {
			r.Missing.Covered++
			continue
		}
		_, inIx := byID[id]
		fe := rp.Earliest.FixedErr()
		if fe == nil {
			fe = rp.Earliest.Errs[0]
		}
		r.Missing.Items = append(r.Missing.Items, MissingItem{Key: x.KeyID(), Label: keyLabel(x.Kind, x.Key), Lesson: id,
			InIndex: inIx, Error: ptrOf(x.Errs[0]), FixedRef: ptrOf(fe)})
	}

	if r.FirstUse != "" {
		r.Baseline = BaselineOf(reps, r.FirstUse, o.Since, o.Until)
	} else {
		r.Baseline = Baseline{Note: baselineNote}
	}
	return r, nil
}

func ptrOf(e *analyze.Err) analyze.Pointer {
	return analyze.Pointer{Agent: e.Agent, Session: e.Sid, Project: e.Project, Ts: e.Ts, RawFile: e.RawFile,
		RawLine: e.RawLine, EventID: e.ID}
}

func eventKind(ev string) string {
	switch ev {
	case "UserPromptSubmit", "SessionStart":
		return "session_start"
	}
	return "failure"
}

func lessonFromID(id string) *lessons.Lesson {
	kind, _, _ := strings.Cut(id, ":")
	// the key is a hash in the id; without the index the outcome cannot be traced
	return &lessons.Lesson{ID: id, Kind: kind, Key: "\x00"}
}

func labelOf(L *lessons.Lesson, id string) string {
	if L == nil {
		return id + " (not in the lesson store)"
	}
	return strings.Join(strings.Fields(L.Text), " ")
}

func keyLabel(kind, key string) string {
	if kind == "cause" {
		return key
	}
	return `"` + key + `"`
}

// Anonymize hides projects, session ids, home paths and private text, as report --anon does.
func (r *Report) Anonymize(a *anon.Anon) {
	for i := range r.Lessons.Shows {
		s := &r.Lessons.Shows[i]
		s.Project, s.Session = a.Project(s.Project), a.Session(s.Session)
		anonPtr(a, s.Again)
		anonPtr(a, s.NextOK)
	}
	for i := range r.Lessons.Rows {
		r.Lessons.Rows[i].Label = a.Lesson(r.Lessons.Rows[i].Label)
	}
	for i := range r.Guards.Events {
		g := &r.Guards.Events[i]
		g.Project, g.Session = a.Project(g.Project), a.Session(g.Session)
		g.Cmd, g.Fixed = a.Lesson(g.Cmd), a.Lesson(g.Fixed)
		anonPtr(a, g.Evidence)
	}
	for i := range r.Noise {
		r.Noise[i].Label = a.Lesson(r.Noise[i].Label)
	}
	for i := range r.Missing.Items {
		m := &r.Missing.Items[i]
		m.Label = a.Lesson(m.Label)
		for _, p := range []*analyze.Pointer{&m.Error, &m.FixedRef} {
			p.Project, p.Session, p.RawFile = a.Project(p.Project), a.Session(p.Session), a.Text(p.RawFile)
		}
	}
	for i := range r.Notes {
		n := &r.Notes[i]
		n.Project, n.Session, n.Text = a.Project(n.Project), a.Session(n.Session), a.Text(n.Text)
	}
	for i := range r.Health.Failures {
		f := &r.Health.Failures[i]
		f.Command, f.Reason = a.Text(f.Command), a.Text(f.Reason)
		anonPtr(a, f.Last)
	}
	for i, l := range r.Health.HookLogLines {
		r.Health.HookLogLines[i] = a.Text(l)
	}
	r.Files = Paths{Shown: a.Text(r.Files.Shown), Guards: a.Text(r.Files.Guards), Calls: a.Text(r.Files.Calls),
		Notes: a.Text(r.Files.Notes), HookLog: a.Text(r.Files.HookLog)}
}

func anonPtr(a *anon.Anon, p *Pointer) {
	if p != nil {
		p.RawFile = a.Text(p.RawFile)
	}
}
