// nanaoki reads local Claude Code and Codex history and reports where agents repeat mistakes.
// No LLM, no network.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/ingest"
)

const usage = `nanaoki: find where your coding agents repeat mistakes (local only, no LLM, no network).

Usage:
  nanaoki [flags]            ingest, then report
  nanaoki ingest [flags]     build or update the event store
  nanaoki report [flags]     print the report from the store
  nanaoki lessons ...        build, export, list the lesson store (nanaoki lessons for help)
  nanaoki hook EVENT         hook handler for Claude Code and Codex (JSON on stdin; see PLUGIN.md)
  nanaoki search QUERY       full-text search over past sessions, with citations
  nanaoki rules suggest      propose rules for CLAUDE.md / AGENTS.md (print only)

Flags:
  --db PATH          event store (default ~/.local/share/nanaoki/events.db)
  --since DATE       report from this day: YYYY-MM-DD, today, or a span (30d, 4w, 3m, 1y = the last
                     30 days, 4 weeks, ...)
  --until DATE       report up to this day, inclusive (same forms)
  --project NAME     report one project (folder name after work/projects/ or work/consulting/)
  --agent NAME       claude or codex
  --format F         text (default), md or json
  --top N            rows per section (default 10)
  --split SPLIT      section 3 uses a fixed split: errors in TEST that repeat a fix from TRAIN.
                     auto        TEST = last third of your history, TRAIN = the rest
                     last:30d    TEST = last 30 days (or 4w, 3m, 1y), TRAIN = everything before
                     T0:T1:S0:S1 explicit days (YYYY-MM-DD)
                     default is rolling (any earlier session)
  --no-filters       count every message and every pasted error (no origin filters, no 3-day
                     spread rule); with --split this gives the numbers of the Python scripts
  --rebuild          ingest: drop the store and read everything again
  --workers N        ingest: parallel parsers (default: CPU count)
  --quiet            ingest: no progress lines

Reads (read-only): $CODEX_HOME or ~/.codex (sessions, archived_sessions, thread_history_1.sqlite,
history.jsonl) and $CLAUDE_CONFIG_DIR or ~/.claude (projects/**, history.jsonl, settings.json).
`

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "hook" {
		os.Exit(runHook(os.Args[2:], os.Stdin, os.Stdout))
	}
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "nanaoki:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	cmd := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "lessons":
		return runLessons(args, os.Stdin, stdout, stderr)
	case "search":
		return runSearch(args, stdout, stderr)
	case "rules":
		return runRules(args, stdout, stderr)
	case "", "ingest", "report":
	case "help":
		fmt.Fprint(stdout, usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
	fs := flag.NewFlagSet("nanaoki", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	db := fs.String("db", ingest.DefaultDB(), "")
	since := fs.String("since", "", "")
	until := fs.String("until", "", "")
	project := fs.String("project", "", "")
	agent := fs.String("agent", "", "")
	format := fs.String("format", "text", "")
	top := fs.Int("top", 10, "")
	rebuild := fs.Bool("rebuild", false, "")
	workers := fs.Int("workers", 0, "")
	quiet := fs.Bool("quiet", false, "")
	split := fs.String("split", "", "")
	noFilters := fs.Bool("no-filters", false, "")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	now := time.Now()
	for _, d := range []*string{since, until} {
		v, err := resolveDate(*d, now)
		if err != nil {
			return err
		}
		*d = v
	}
	if *agent != "" && *agent != "claude" && *agent != "codex" {
		return fmt.Errorf("--agent must be claude or codex")
	}
	switch *format {
	case "text", "md", "json":
	default:
		return fmt.Errorf("--format must be text, md or json")
	}
	paths := ingest.DefaultPaths()

	if cmd == "" || cmd == "ingest" {
		var log io.Writer
		if !*quiet {
			log = stderr
		}
		st, err := ingest.Run(ingest.Options{DB: *db, Paths: paths, Workers: *workers, Rebuild: *rebuild, Log: log})
		if err != nil {
			return err
		}
		if !*quiet {
			fmt.Fprintf(stderr, "ingest: %d files (%d parsed, %d unchanged, %d gone from disk), %d rows added, %d copied lines dropped, %d prompt/gap-fill rows, %.1fs\n",
				st.FilesSeen, st.FilesParsed, st.FilesUnchanged, st.FilesGone, st.RowsInserted, st.RowsDeduped, st.RowsDerived, st.Elapsed.Seconds())
			for _, e := range st.Errors {
				fmt.Fprintln(stderr, "  warning:", e)
			}
		}
		if cmd == "ingest" {
			return nil
		}
	}

	store, err := ingest.OpenStoreRO(*db)
	if err != nil {
		return err
	}
	defer store.Close()
	o := analyze.Options{Filter: analyze.Filter{Since: *since, Until: *until, Project: *project, Agent: *agent}, Top: *top, NoFilters: *noFilters}
	o.Paths.ClaudeDir = paths.ClaudeDir
	if *split != "" {
		p, err := resolveSplit(*split, now, store)
		if err != nil {
			return err
		}
		o.Split = &analyze.Split{TrainStart: p[0], TrainEnd: p[1], TestStart: p[2], TestEnd: p[3]}
	}
	r, err := analyze.Build(store, o)
	if err != nil {
		return err
	}
	return analyze.Render(stdout, r, *format)
}
