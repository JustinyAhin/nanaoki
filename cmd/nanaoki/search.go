package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/anon"
	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/rules"
	"github.com/JustinyAhin/nanaoki/internal/search"
	"github.com/JustinyAhin/nanaoki/internal/style"
)

const searchUsage = `nanaoki search: full-text search over your past Claude Code and Codex sessions (local).

Usage:
  nanaoki search [flags] "QUERY"

Flags:
  --project NAME   only this project ("." = the project of the current folder)
  --agent NAME     claude or codex
  --since DATE     from this day (YYYY-MM-DD, today, or a span: 30d, 4w, 3m, 1y)
  --until DATE     up to this day, inclusive
  --limit N        results (default 8; one per session, with the count of its other hits)
  --snippet N      snippet length in tokens (default and max 64)
  --format F       text (default) or json
  --raw            QUERY is FTS5 syntax (phrases "a b", AND, NOT, prefix*); default: any of the words
  --no-sync        do not update the index from the event store first
  --include-current  keep the current session (default: left out, see below)
  --paths          show the source file and line of each result (always in json)
  --anon           hide project names, home paths and private data (for screenshots and sharing)
  --no-color       no colors (also: NO_COLOR=1; pipes get plain text)
  --db PATH        event store (default ~/.local/share/nanaoki/events.db); the index is search.db next to it

Each result cites the agent, time, project and session id. The current session is the newest session
of the current project with an event in the last 10 minutes.
Run ` + "`nanaoki ingest`" + ` first to read new sessions.
`

func runSearch(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, searchUsage) }
	db := fs.String("db", ingest.DefaultDB(), "")
	project := fs.String("project", "", "")
	agent := fs.String("agent", "", "")
	since := fs.String("since", "", "")
	until := fs.String("until", "", "")
	limit := fs.Int("limit", 8, "")
	snip := fs.Int("snippet", 64, "")
	format := fs.String("format", "text", "")
	raw := fs.Bool("raw", false, "")
	noSync := fs.Bool("no-sync", false, "")
	inclCur := fs.Bool("include-current", false, "")
	paths := fs.Bool("paths", false, "")
	anonF := fs.Bool("anon", false, "")
	noColor := fs.Bool("no-color", false, "")
	// eval only: leakage rules
	before := fs.String("before", "", "")
	exclSess := fs.String("exclude-session", "", "")
	exclProj := fs.String("exclude-projects", "", "")
	names := fs.String("names-file", "", "")
	// flags may come after the query
	var words []string
	rest := args
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() == 0 {
			break
		}
		words = append(words, fs.Arg(0))
		rest = fs.Args()[1:]
	}
	q := strings.Join(words, " ")
	if strings.TrimSpace(q) == "" {
		fmt.Fprint(stderr, searchUsage)
		return fmt.Errorf("no query")
	}
	now := time.Now()
	for _, d := range []*string{since, until} {
		v, err := resolveDate(*d, now)
		if err != nil {
			return err
		}
		*d = v
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("--format must be text or json")
	}
	if *names != "" {
		b, err := os.ReadFile(*names)
		if err != nil {
			return err
		}
		rules.SetNames(strings.Split(string(b), "\n"))
	}
	if *project == "." {
		cwd, _ := os.Getwd()
		*project = rules.Project(cwd)
	}
	sdb, err := search.Open(search.DefaultPath(*db))
	if err != nil {
		return err
	}
	defer sdb.Close()
	if !*noSync {
		if _, err := os.Stat(*db); err != nil {
			return fmt.Errorf("no event store at %s (run `nanaoki ingest` first)", *db)
		}
		st, err := search.Sync(sdb, *db)
		if err != nil {
			return err
		}
		if st.Added+st.Removed > 1000 {
			fmt.Fprintf(stderr, "search index: %d added, %d removed, %.1fs\n", st.Added, st.Removed, st.Elapsed.Seconds())
		}
	}
	ex := map[string]bool{}
	for _, p := range strings.Split(*exclProj, ",") {
		if p != "" {
			ex[p] = true
		}
	}
	sq := search.Query{Text: q, Raw: *raw, Project: *project, Agent: *agent, Since: *since, Until: *until,
		Before: *before, ExcludeSession: *exclSess, ExcludeProjects: ex, Limit: *limit, SnippetTokens: *snip}
	if !*inclCur && *before == "" {
		cwd, _ := os.Getwd()
		cur, err := search.CurrentSession(sdb, rules.Project(cwd), now)
		if err != nil {
			return err
		}
		if cur != "" {
			sq.Exclude = []string{cur}
		}
	}
	res, err := search.Search(sdb, sq)
	if err != nil {
		return err
	}
	var an *anon.Anon
	if *anonF {
		known, err := search.Projects(sdb)
		if err != nil {
			return err
		}
		an = anon.New(known)
		for i := range res {
			r := &res[i]
			r.Project, r.Session = an.Project(r.Project), an.Session(r.Session)
			r.Snippet, r.Marked, r.Pointer = an.Text(r.Snippet), an.Text(r.Marked), an.Text(r.Pointer)
		}
	}
	if *format == "json" {
		if res == nil {
			res = []search.Result{}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", " ")
		return enc.Encode(res)
	}
	st := style.Detect(stdout, *noColor)
	if len(res) == 0 {
		fmt.Fprintln(stdout, "no results")
		return nil
	}
	for i, r := range res {
		writeHit(stdout, st, i+1, r, *paths)
	}
	return nil
}

var kindName = map[string]string{"assistant_msg": "answer", "user_msg": "prompt", "tool_call": "tool call",
	"tool_error": "tool error", "user_error_paste": "pasted error"}

func writeHit(w io.Writer, st style.Style, n int, r search.Result, paths bool) {
	ts := r.Ts
	if t, err := time.Parse(time.RFC3339Nano, r.Ts); err == nil {
		ts = t.Local().Format("2006-01-02 15:04")
	} else if len(ts) >= 16 {
		ts = ts[:10] + " " + ts[11:16]
	}
	sid := r.Session
	if len(sid) > 8 {
		sid = sid[:8]
	}
	kind := kindName[r.EventType]
	if kind == "" {
		kind = strings.ReplaceAll(r.EventType, "_", " ")
	}
	switch r.EventType {
	case "assistant_msg":
		kind = st.Green(kind)
	case "user_msg":
		kind = st.Cyan(kind)
	case "tool_error", "user_error_paste":
		kind = st.Red(kind)
	default:
		kind = st.Yellow(kind)
	}
	agent := r.Agent
	if agent == "claude" {
		agent = st.Magenta(agent)
	} else {
		agent = st.Blue(agent)
	}
	dot := st.Dim(" · ")
	head := fmt.Sprintf("%s  %s%s%s%s%s%s%s%s%s", st.Bold(fmt.Sprintf("%2d", n)), kind, dot, agent, dot, st.Dim(ts), dot,
		st.Bold(proj(r.Project)), dot, st.Dim(sid))
	if r.Hits > 1 {
		head += st.Dim(fmt.Sprintf("  +%d more in this session", r.Hits-1))
	}
	fmt.Fprintln(w, head)
	width := st.Width - 4
	if st.Width == 0 {
		width = style.DefaultWidth - 4
	}
	for _, l := range st.Snippet(r.Marked, search.MarkOpen, search.MarkClose, width, 2) {
		fmt.Fprintln(w, "    "+l)
	}
	if paths && r.Pointer != "" {
		fmt.Fprintln(w, "    "+st.Dim(r.Pointer))
	}
}

func proj(p string) string {
	if p == "" {
		return "?"
	}
	return p
}
