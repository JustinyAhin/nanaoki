package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/activity"
	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/anon"
	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
	"github.com/JustinyAhin/nanaoki/internal/rules"
	"github.com/JustinyAhin/nanaoki/internal/stats"
	"github.com/JustinyAhin/nanaoki/internal/style"
)

const statsUsage = `Usage: nanaoki stats [--since 7d] [--until DATE] [--anon] [--format text|md|json]

What the plugin did in the period: lessons shown and whether the same failure came back in that session,
guards fired and whether the next command used the fix, noise candidates, repeats that got no lesson, hook
health, notes, and a rough before/after of the repeat-after-fix rate. Run nanaoki ingest first so the
store holds the recent sessions.
`

func runStats(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, statsUsage) }
	db := fs.String("db", ingest.DefaultDB(), "")
	since := fs.String("since", "", "")
	until := fs.String("until", "", "")
	format := fs.String("format", "text", "")
	anonF := fs.Bool("anon", false, "")
	noColor := fs.Bool("no-color", false, "")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	switch *format {
	case "text", "md", "json":
	default:
		return fmt.Errorf("--format must be text, md or json")
	}
	now := time.Now()
	for _, d := range []*string{since, until} {
		v, err := resolveDate(*d, now)
		if err != nil {
			return err
		}
		*d = v
	}
	o := stats.Options{Since: *since, Until: *until, Paths: stats.DefaultPaths(), Now: now}
	if ix, err := lessons.LoadIndex(lessons.IndexPath()); err == nil {
		o.Index = ix
	}
	var r *stats.Report
	var known []string
	if _, err := os.Stat(*db); err == nil {
		store, err := ingest.OpenStoreRO(*db)
		if err != nil {
			return err
		}
		defer store.Close()
		if r, err = stats.Build(store, o); err != nil {
			return err
		}
		if *anonF {
			if known, err = analyze.KnownProjects(store); err != nil {
				return err
			}
		}
	} else {
		var err error
		if r, err = stats.Build(nil, o); err != nil {
			return err
		}
	}
	if *anonF {
		r.Anonymize(anon.New(known))
	}
	return stats.Render(stdout, r, *format, style.Detect(stdout, *noColor))
}

const noteUsage = `Usage: nanaoki note "text" [--useful|--noise|--wrong|--missing] [--lesson ID] [--guard ID]
       nanaoki note --list [--since 7d]

Saves a note about the plugin to ~/.local/share/nanaoki/notes.jsonl with the time, the project of the
current folder and the current session (from the agent's environment, else the newest hook call in this
project). Tags: --useful, --noise (shown but did not help), --wrong (wrong advice), --missing (a lesson
that should have shown up). nanaoki stats counts the notes.
`

func runNote(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("note", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, noteUsage) }
	list := fs.Bool("list", false, "")
	since := fs.String("since", "", "")
	lesson := fs.String("lesson", "", "")
	guard := fs.String("guard", "", "")
	session := fs.String("session", "", "")
	tags := map[string]*bool{}
	for _, t := range activity.Tags {
		tags[t] = fs.Bool(t, false, "")
	}
	var words []string
	for {
		if err := fs.Parse(args); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() == 0 {
			break
		}
		words = append(words, fs.Arg(0))
		args = fs.Args()[1:]
	}
	path := activity.NotesPath()
	now := time.Now()
	if *list {
		s, err := resolveDate(*since, now)
		if err != nil {
			return err
		}
		ns, err := activity.Read[activity.Note](path)
		if err != nil {
			return err
		}
		var rows []stats.NoteRow
		for _, l := range ns {
			if s == "" || l.V.Ts[:min(10, len(l.V.Ts))] >= s {
				rows = append(rows, stats.NoteRow{Line: l.N, Note: l.V})
			}
		}
		stats.RenderNotes(stdout, rows, style.Detect(stdout, false))
		return nil
	}
	text := strings.TrimSpace(strings.Join(words, " "))
	if text == "" {
		return fmt.Errorf("a note needs text\n\n%s", noteUsage)
	}
	tag := ""
	for _, t := range activity.Tags {
		if *tags[t] {
			if tag != "" {
				return fmt.Errorf("use one tag: --%s or --%s", tag, t)
			}
			tag = t
		}
	}
	cwd, _ := os.Getwd()
	n := activity.Note{Ts: now.UTC().Format(time.RFC3339), Project: rules.Project(cwd), Lesson: *lesson, Guard: *guard, Tag: tag,
		Text: rules.Redact(text, 2000)}
	n.Agent, n.Session = currentSession(*session, n.Project, now)
	if lessonRefRx.MatchString(n.Lesson) {
		id, err := resolveLessonRef(n.Lesson, n.Agent, n.Session)
		if err != nil {
			return err
		}
		n.Lesson = id
	}
	n.ID = activity.NoteID(n.Ts, n.Text)
	if err := activity.Append(path, n); err != nil {
		return err
	}
	sess := "no session found"
	if n.Session != "" {
		sess = strings.TrimSpace(n.Agent + " session " + n.Session[:min(8, len(n.Session))])
	}
	fmt.Fprintf(stdout, "Saved note %s (%s, %s).\n", n.ID, n.Project, sess)
	return nil
}

var lessonRefRx = regexp.MustCompile(`^L[1-9]$`)

// resolveLessonRef maps "L2" to the lesson shown as [L2] in the newest hook reply of the session. One reply's
// lines share the time and event, in the order of the reply.
func resolveLessonRef(ref, agent, sid string) (string, error) {
	if sid == "" {
		return "", fmt.Errorf("--lesson %s needs the current session; none found (pass --session ID or the lesson id)", ref)
	}
	shown, err := activity.Read[lessons.ShownEntry](lessons.ShownLogPath())
	if err != nil {
		return "", err
	}
	var last []lessons.ShownEntry
	for _, l := range shown {
		e := l.V
		if e.Session != sid || (agent != "" && e.Agent != agent) {
			continue
		}
		if len(last) == 0 || e.Ts != last[0].Ts || e.Event != last[0].Event {
			last = last[:0]
		}
		last = append(last, e)
	}
	i := int(ref[1] - '1')
	if i >= len(last) {
		return "", fmt.Errorf("no lesson %s in the newest hook reply of this session", ref)
	}
	return last[i].Lesson, nil
}

// currentSession: an explicit id, the agent's environment, else the newest hook call in this project in the
// last 6 hours.
func currentSession(explicit, project string, now time.Time) (string, string) {
	if explicit != "" {
		return "", explicit
	}
	for _, e := range []struct{ env, agent string }{{"NANAOKI_SESSION_ID", ""}, {"CLAUDE_CODE_SESSION_ID", "claude"},
		{"CLAUDE_SESSION_ID", "claude"}, {"CODEX_THREAD_ID", "codex"}} {
		if v := os.Getenv(e.env); v != "" {
			return e.agent, v
		}
	}
	calls, _ := activity.Read[activity.CallEntry](activity.CallLogPath())
	for i := len(calls) - 1; i >= 0; i-- {
		c := calls[i].V
		t, err := time.Parse(time.RFC3339, c.Ts)
		if err != nil || now.Sub(t) > 6*time.Hour {
			break
		}
		if c.Session != "" && c.Project == project {
			return c.Agent, c.Session
		}
	}
	return "", ""
}
