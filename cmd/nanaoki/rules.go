package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
	"github.com/JustinyAhin/nanaoki/internal/rules"
	"github.com/JustinyAhin/nanaoki/internal/rulesuggest"
)

const rulesUsage = `nanaoki rules: propose rules for CLAUDE.md / AGENTS.md from what you keep typing (print only).

Usage:
  nanaoki rules suggest [--project] [--format text|json] [--db PATH]

  default     global rules (restated in 2+ projects) as a diff for ~/.claude/CLAUDE.md and ~/.codex/AGENTS.md
  --project   rules of the project in the current folder, as a diff for ./AGENTS.md
  --decisions also show "decision" lines (restated instruction sentences). Off by default:
              they can still pick up pasted log text. JSON output always includes them.

Sources: preference rules you typed in 2+ sessions (no AI attribution, commit only when told, ...), and
instruction sentences you restated in 2+ sessions. Rules the file already states are left out.
v0 never writes a file: review the diff and copy the lines you want.
`

func runRules(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "suggest" {
		fmt.Fprint(stdout, rulesUsage)
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("unknown rules command %q", args[0])
	}
	fs := flag.NewFlagSet("rules suggest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, rulesUsage) }
	db := fs.String("db", ingest.DefaultDB(), "")
	project := fs.Bool("project", false, "")
	format := fs.String("format", "text", "")
	write := fs.Bool("write", false, "")
	decisions := fs.Bool("decisions", false, "")
	before := fs.String("before", "", "")
	exclSess := fs.String("exclude-session", "", "")
	cfgPath := fs.String("config", "", "")
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if *write {
		return fmt.Errorf("--write is not available in v0: nanaoki only prints the proposed diff")
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("--format must be text or json")
	}
	var cfg lessonsCfg
	if *cfgPath != "" {
		b, err := os.ReadFile(*cfgPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &cfg); err != nil {
			return err
		}
		if cfg.Names != nil {
			rules.SetNames(cfg.Names)
		}
	}
	store, err := ingest.OpenStoreRO(*db)
	if err != nil {
		return err
	}
	defer store.Close()
	excl := toSet(cfg.ExcludeProjects)
	r, err := lessons.Load(store, lessons.Config{Since: cfg.Since, ExcludeProjects: excl, ExcludePrefProjects: toSet(cfg.ExcludePrefProjects)})
	if err != nil {
		return err
	}
	exPrompts := toSet(append(append([]string{}, cfg.ExcludeProjects...), cfg.ExcludePrefProjects...))
	rs := rulesuggest.Suggest(r.Data, rulesuggest.Options{Before: *before, ExcludeSession: *exclSess, ExcludeProjects: exPrompts})
	cwd, _ := os.Getwd()
	proj := rules.Project(cwd)
	var pick []rulesuggest.Rule
	for _, x := range rs {
		// "decision" lines can still pick up pasted log text; show them only on request
		if x.Kind == "decision" && !*decisions {
			continue
		}
		if (*project && x.Scope == "project" && x.Project == proj) || (!*project && x.Scope == "global") || *format == "json" {
			pick = append(pick, x)
		}
	}
	if *format == "json" {
		if pick == nil {
			pick = []rulesuggest.Rule{}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", " ")
		return enc.Encode(pick)
	}
	if len(pick) == 0 {
		fmt.Fprintln(stdout, "No stable rules to propose.")
		return nil
	}
	fmt.Fprintf(stdout, "Proposed rules (%d). Evidence: sessions, projects, last seen.\n\n", len(pick))
	for _, x := range pick {
		fmt.Fprintf(stdout, "  %-8s %3d sess  %2d proj  %s  %s\n", x.Kind, x.Sessions, len(x.Projects), day10(x.Last),
			rules.Trunc(strings.Join(strings.Fields(x.Text), " "), 110))
	}
	paths := ingest.DefaultPaths()
	var targets [][2]string
	if *project {
		targets = [][2]string{{filepath.Join(cwd, "AGENTS.md"), "AGENTS.md"}}
	} else {
		targets = [][2]string{{filepath.Join(paths.ClaudeDir, "CLAUDE.md"), "~/.claude/CLAUDE.md"},
			{filepath.Join(paths.CodexDir, "AGENTS.md"), "~/.codex/AGENTS.md"}}
	}
	fmt.Fprintln(stdout, "\nProposed diff (not applied; nanaoki never writes these files):")
	for _, t := range targets {
		d, n := rulesuggest.Diff(t[0], t[1], pick)
		if n == 0 {
			fmt.Fprintf(stdout, "\n%s already states all these rules.\n", t[1])
			continue
		}
		fmt.Fprintln(stdout)
		fmt.Fprint(stdout, d)
	}
	return nil
}

func day10(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}
