package lessons

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/rules"
	"github.com/JustinyAhin/nanaoki/internal/testfixture"
)

// Synthetic history: the same zsh glob error fixed in two sessions of two projects, and one rule typed in
// two sessions.
var evs = []testfixture.Ev{
	{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:00:00.000Z", Type: "tool_call", Text: "cat package.json", Out: `{"devDependencies":{"@sveltejs/kit":"2"}}`},
	{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:00:01.000Z", Type: "tool_call", Text: "cat svelte.config.js"},
	{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:00:02.000Z", Type: "tool_call", Text: "npx svelte-check"},
	{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:00:03.000Z", Type: "tool_call", Text: "cat svelte.config.ts"},
	{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:01:00.000Z", Type: "tool_error", Text: "ls src/routes/[slug]", Out: "(eval):1: no matches found: src/routes/[slug]", Err: true},
	{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:02:00.000Z", Type: "tool_call", Text: "ls 'src/routes/[slug]'"},
	{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:03:00.000Z", Type: "user_msg", Text: "Looks good. Never add co-author trailers to the commits in this repo."},
	{Agent: "claude", Sid: "s2", Project: "beta", Ts: "2026-09-05T09:00:00.000Z", Type: "tool_error", Text: "rg foo --include=*.md", Out: "zsh:1: no matches found: --include=*.md", Err: true},
	{Agent: "claude", Sid: "s2", Project: "beta", Ts: "2026-09-05T09:00:30.000Z", Type: "tool_call", Text: "rg foo -g '*.md'"},
	{Agent: "claude", Sid: "s2", Project: "beta", Ts: "2026-09-05T09:05:00.000Z", Type: "user_msg", Text: "Never add co-author trailers to the commits in this repo."},
	{Agent: "codex", Sid: "s3", Project: "alpha", Ts: "2026-09-10T08:00:00.000Z", Type: "tool_error", Text: "cat src/routes/[id]/+page.ts", Out: "(eval):1: no matches found: src/routes/[id]/+page.ts", Err: true},
}

func load(t *testing.T) *Rolling {
	t.Helper()
	SetTestNames()
	p := testfixture.EventsDB(t, ingest.OpenStore, evs)
	db, err := ingest.OpenStoreRO(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	r, err := Load(db, Config{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRollingSnapshot(t *testing.T) {
	r := load(t)
	tr := Trigger{Kind: "error", Project: "alpha", Cause: "zsh_glob_nomatch", Sig: "(eval):N: no matches found: <path>", Head: "cat"}
	// at s3's error: both earlier fixes are known
	st := r.Snapshot(tr, "2026-09-10T08:00:00.000Z", "s3")
	hits := st.Retrieve(tr, 3)
	if len(hits) != 1 || hits[0].Lesson.Kind != "cause" || hits[0].Lesson.Sessions != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if !strings.Contains(hits[0].Lesson.Text, "Fix seen (alpha): `ls 'src/routes/[slug]'`") || hits[0].Why != "cause+project" {
		t.Errorf("text/why = %q %q", hits[0].Lesson.Text, hits[0].Why)
	}
	// before s2's fix, leaving out s1: nothing
	if h := r.Snapshot(tr, "2026-09-05T09:00:10.000Z", "s1").Retrieve(tr, 3); len(h) != 0 {
		t.Errorf("leakage: %+v", h)
	}
	if got := r.PstackOf("alpha", "2026-09-01T10:00:03.000Z"); len(got) != 0 {
		t.Errorf("only two signals before the third timestamp: %v", got)
	}
	if got := r.PstackOf("alpha", "2026-09-02"); !reflect.DeepEqual(got, map[string]bool{"sveltekit": true}) {
		t.Errorf("pstack = %v", got)
	}
	if got := r.RepoStacksAt("alpha", "2026-09-02"); !got["sveltekit"] || got["node"] { // node: 1 signal < 2
		t.Errorf("repo stacks = %v", got)
	}
}

func TestPrefsAndStrictStart(t *testing.T) {
	r := load(t)
	ps := r.PrefsAt("2026-09-10", "")
	if len(ps) != 1 || ps[0].Sessions != 2 || ps[0].Text != "Never add co-author trailers to the commits in this repo." {
		t.Fatalf("prefs = %+v", ps)
	}
	st := NewStore(ps, nil)
	tr := Trigger{Kind: "session_start", Project: "alpha", Text: "Commit this. Remember: never add co-author trailers to commits."}
	hits := st.RetrieveStartStrict(tr, func(p string) map[string]bool { return r.RepoStacksAt(p, "2026-09-10") }, StartK)
	if len(hits) != 1 || !strings.HasSuffix(hits[0].Why, "+project") {
		t.Errorf("strict = %+v", hits)
	}
	tr.Project = "gamma" // no repo match, no shared stack
	if h := st.RetrieveStartStrict(tr, func(string) map[string]bool { return nil }, StartK); len(h) != 0 {
		t.Errorf("other repo: %+v", h)
	}
}

func TestSentences(t *testing.T) {
	got := SplitSentences("One.  Two!\n\nThree? four\nfive")
	want := []string{"One.", "Two!", "Three?", "four", "five"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("split = %q", got)
	}
	if n := NormSentence("Use https://x.y/z and /a/b/c 42 times!"); n != "use and N times" {
		t.Errorf("norm = %q", n)
	}
	if ps := PrefSentences("/humanizer"); ps != nil {
		t.Errorf("bare skill call: %v", ps)
	}
	if ps := PrefSentences("What do you think, should we never do this again?"); ps != nil {
		t.Errorf("question: %v", ps)
	}
}

func TestIndexAndExport(t *testing.T) {
	r := load(t)
	ix := r.BuildIndex()
	p := filepath.Join(t.TempDir(), "lessons.gob")
	if err := ix.Save(p); err != nil {
		t.Fatal(err)
	}
	ix2, err := LoadIndex(p)
	if err != nil || len(ix2.Lessons) != len(ix.Lessons) {
		t.Fatalf("load: %v", err)
	}
	dir := t.TempDir()
	res, err := Export(ix2, ExportOptions{Dir: dir, ShownLog: filepath.Join(dir, "none.jsonl")})
	if err != nil || res.Written != 2 { // the cause lesson and the rule (2 projects each)
		t.Fatalf("export: %+v %v", res, err)
	}
	var cause string
	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	for _, f := range files {
		if strings.Contains(f, "cause-") {
			b, _ := os.ReadFile(f)
			cause = string(b)
			for _, want := range []string{"status: confirmed", "enforce: hook", "guard: zsh_glob", "cause: zsh_glob_nomatch",
				`session: "codex:s1"`, "repeats_after: 0", "\nzsh fails on unmatched globs."} {
				if !strings.Contains(cause, want) {
					t.Errorf("cause file lacks %q:\n%s", want, cause)
				}
			}
			os.WriteFile(f, []byte(strings.Replace(cause, "status: confirmed", "status: retired", 1)), 0o644)
		}
	}
	res, _ = Export(ix2, ExportOptions{Dir: dir, ShownLog: filepath.Join(dir, "none.jsonl")})
	if res.Retired != 1 || res.Unchanged != 1 {
		t.Errorf("re-export: %+v", res)
	}
	// project export: no lesson belongs to one project only
	res, _ = Export(ix2, ExportOptions{Project: true, Cwd: "/Users/x/work/projects/alpha", Dir: t.TempDir()})
	if res.Written != 0 {
		t.Errorf("project export: %+v", res)
	}
}

func TestScore(t *testing.T) {
	L := &Lesson{LastTs: "2026-11-02T00:00:00Z", EpisodeTs: []string{"2026-10-01", "2026-11-01", "2026-11-02"}}
	s := scoreOf(L, shownStat{n: 4, first: "2026-10-15T00:00:00Z"})
	if s.Shown != 4 || s.RepeatsAfter != 2 || s.LastSeen != "2026-11-02" {
		t.Errorf("score = %+v", s)
	}
}

// SetTestNames turns off the private-name list (the defaults read the real git config).
func SetTestNames() { rules.SetNames(nil) }
