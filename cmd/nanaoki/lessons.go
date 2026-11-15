package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/anon"
	"github.com/JustinyAhin/nanaoki/internal/guards"
	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
	"github.com/JustinyAhin/nanaoki/internal/rules"
	"github.com/JustinyAhin/nanaoki/internal/style"
)

const lessonsUsage = `nanaoki lessons: the rolling lesson store (local, no LLM, no network).

Usage:
  nanaoki lessons build  [--db PATH] [--index PATH]
        build the store from the event store and cache the index for the hooks
  nanaoki lessons export [--dir DIR] [--project] [--index PATH]
        write one markdown file per confirmed lesson (default dir ~/.local/share/nanaoki/lessons;
        --project: lessons of the project in the current folder, into ./.lessons)
  nanaoki lessons list   [--index PATH] [--all] [--anon] [--no-color]
        print the confirmed lessons (--all: candidates too; --anon: hide project names and file paths)
  nanaoki lessons query  --config FILE [--db PATH] < triggers.jsonl
        eval adapter: for each trigger line {id, trigger, t, exclude_session, mode, k}, print the lessons
        known before t (mode retrieve or strict)
`

type lessonsCfg struct {
	Since               string   `json:"since"`
	ExcludeProjects     []string `json:"exclude_projects"`
	ExcludePrefProjects []string `json:"exclude_pref_projects"`
	Names               []string `json:"names"`
}

func toSet(v []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range v {
		m[x] = true
	}
	return m
}

func runLessons(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, lessonsUsage)
		return nil
	}
	sub, args := args[0], args[1:]
	fs := flag.NewFlagSet("lessons "+sub, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, lessonsUsage) }
	db := fs.String("db", ingest.DefaultDB(), "")
	index := fs.String("index", lessons.IndexPath(), "")
	cfgPath := fs.String("config", "", "")
	dir := fs.String("dir", "", "")
	project := fs.Bool("project", false, "")
	all := fs.Bool("all", false, "")
	noColor := fs.Bool("no-color", false, "")
	anonF := fs.Bool("anon", false, "")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	switch sub {
	case "build":
		t0 := time.Now()
		store, err := ingest.OpenStoreRO(*db)
		if err != nil {
			return err
		}
		defer store.Close()
		r, err := lessons.Load(store, lessons.Config{})
		if err != nil {
			return err
		}
		ix := r.BuildIndex()
		if err := ix.Save(*index); err != nil {
			return err
		}
		n := map[string]int{}
		conf := 0
		for _, L := range ix.Lessons {
			n[L.Kind]++
			if lessons.StatusOf(L) == "confirmed" {
				conf++
			}
		}
		fmt.Fprintf(stderr, "lessons: %d (cause %d, sig %d, pref %d), %d confirmed; index %s; %.1fs\n",
			len(ix.Lessons), n["cause"], n["sig"], n["pref"], conf, *index, time.Since(t0).Seconds())
		return nil
	case "query":
		return lessonsQuery(*db, *cfgPath, stdin, stdout)
	case "export":
		ix, err := lessons.LoadIndex(*index)
		if err != nil {
			return fmt.Errorf("no lesson index (run `nanaoki lessons build` first): %w", err)
		}
		cwd, _ := os.Getwd()
		res, err := lessons.Export(ix, lessons.ExportOptions{Dir: *dir, Project: *project, Cwd: cwd, ShownLog: lessons.ShownLogPath()})
		if err != nil {
			return err
		}
		fmt.Fprintf(stderr, "export: %d written, %d kept as retired, %d unchanged, into %s\n", res.Written, res.Retired, res.Unchanged, res.Dir)
		return nil
	case "list":
		ix, err := lessons.LoadIndex(*index)
		if err != nil {
			return fmt.Errorf("no lesson index (run `nanaoki lessons build` first): %w", err)
		}
		ls := append([]*lessons.Lesson(nil), ix.Lessons...)
		lessons.SortLessons(ls)
		sty := style.Detect(stdout, *noColor)
		var an *anon.Anon
		if *anonF {
			var known []string
			for _, L := range ls {
				for p := range L.Projects {
					known = append(known, p)
				}
			}
			an = anon.New(known)
		}
		t := style.Table{Head: []string{"ID", "Status", "Sessions", "Lesson"}, Flex: -1}
		for _, L := range ls {
			st := lessons.StatusOf(L)
			if st != "confirmed" && !*all {
				continue
			}
			stc := sty.Dim(st)
			if st == "confirmed" {
				stc = sty.Green(st)
			}
			t.Rows = append(t.Rows, []string{sty.Dim(L.ID), stc, fmt.Sprint(L.Sessions), style.OneLine(an.Lesson(L.Text))})
		}
		sty.Render(stdout, t)
		return nil
	}
	return fmt.Errorf("unknown lessons command %q\n\n%s", sub, lessonsUsage)
}

type queryIn struct {
	ID             string          `json:"id"`
	Trigger        lessons.Trigger `json:"trigger"`
	T              string          `json:"t"`
	ExcludeSession string          `json:"exclude_session"`
	Mode           string          `json:"mode"` // retrieve | strict | guards
	K              int             `json:"k"`
	Cmd            string          `json:"cmd"`        // guards: the command
	Cwd            string          `json:"cwd"`        // guards: its folder
	FormatRan      bool            `json:"format_ran"` // guards: a formatter ran earlier in the session
}

type queryOut struct {
	ID     string           `json:"id"`
	Hits   []lessons.Hit    `json:"hits"`
	Guards []guards.Finding `json:"guards,omitempty"`
}

func lessonsQuery(dbPath, cfgPath string, stdin io.Reader, stdout io.Writer) error {
	var cfg lessonsCfg
	if cfgPath != "" {
		b, err := os.ReadFile(cfgPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &cfg); err != nil {
			return err
		}
	}
	if cfg.Names != nil {
		rules.SetNames(cfg.Names)
	}
	store, err := ingest.OpenStoreRO(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	r, err := lessons.Load(store, lessons.Config{Since: cfg.Since, ExcludeProjects: toSet(cfg.ExcludeProjects),
		ExcludePrefProjects: toSet(cfg.ExcludePrefProjects)})
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	enc := json.NewEncoder(stdout)
	for sc.Scan() {
		var q queryIn
		if err := json.Unmarshal(sc.Bytes(), &q); err != nil {
			return err
		}
		if q.K == 0 {
			q.K = 3
		}
		if q.Mode == "guards" {
			// Replay: no file system or PATH checks for the zsh guards (the error already happened); the node
			// guard assumes the node on PATH differed from the pin (the pin is read from the repo on disk today).
			e := &guards.Env{Cwd: q.Cwd, Shell: "/bin/zsh", Replay: true, FormatRan: q.FormatRan,
				SSHUsers:  r.SSHUsersAt(q.T, q.ExcludeSession),
				LookPath:  func(string) (string, error) { return "", os.ErrNotExist },
				NodeMajor: func() (int, bool) { return -1, true }}
			if err := enc.Encode(queryOut{ID: q.ID, Hits: []lessons.Hit{}, Guards: guards.Check(q.Cmd, e)}); err != nil {
				return err
			}
			continue
		}
		// trigger stacks = session stacks + project stacks known before t
		snap := r.Snapshot(q.Trigger, q.T, q.ExcludeSession)
		var hits []lessons.Hit
		if q.Mode == "strict" {
			hits = snap.RetrieveStartStrict(q.Trigger, func(p string) map[string]bool { return r.RepoStacksAt(p, q.T) }, q.K)
		} else {
			hits = snap.Retrieve(q.Trigger, q.K)
		}
		if hits == nil {
			hits = []lessons.Hit{}
		}
		if err := enc.Encode(queryOut{ID: q.ID, Hits: hits}); err != nil {
			return err
		}
	}
	return sc.Err()
}
