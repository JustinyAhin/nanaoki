package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/rules"
	"github.com/JustinyAhin/nanaoki/internal/search"
)

const searchUsage = `nanaoki search: full-text search over your past Claude Code and Codex sessions (local).

Usage:
  nanaoki search [flags] "QUERY"

Flags:
  --project NAME   only this project ("." = the project of the current folder)
  --agent NAME     claude or codex
  --since DATE     from this day (YYYY-MM-DD, today, or a span: 30d, 4w, 3m, 1y)
  --until DATE     up to this day, inclusive
  --limit N        results (default 8; at most 2 per session)
  --snippet N      snippet length in tokens (default and max 64)
  --format F       text (default) or json
  --raw            QUERY is FTS5 syntax (phrases "a b", AND, NOT, prefix*); default: any of the words
  --no-sync        do not update the index from the event store first
  --db PATH        event store (default ~/.local/share/nanaoki/events.db); the index is search.db next to it

Each result cites the agent, session id, time, project and the source file line.
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
	res, err := search.Search(sdb, search.Query{Text: q, Raw: *raw, Project: *project, Agent: *agent, Since: *since, Until: *until,
		Before: *before, ExcludeSession: *exclSess, ExcludeProjects: ex, Limit: *limit, SnippetTokens: *snip})
	if err != nil {
		return err
	}
	if *format == "json" {
		if res == nil {
			res = []search.Result{}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", " ")
		return enc.Encode(res)
	}
	if len(res) == 0 {
		fmt.Fprintln(stdout, "no results")
		return nil
	}
	for i, r := range res {
		ts := r.Ts
		if len(ts) >= 16 {
			ts = ts[:10] + " " + ts[11:16]
		}
		sid := r.Session
		if len(sid) > 8 {
			sid = sid[:8]
		}
		fmt.Fprintf(stdout, "%d. %s  %s UTC  %s  session %s  %s\n   %s\n", i+1, r.Agent, ts, r.Project, sid, r.EventType,
			strings.Join(strings.Fields(r.Snippet), " "))
		if r.Pointer != "" {
			fmt.Fprintf(stdout, "   %s\n", r.Pointer)
		}
	}
	return nil
}
