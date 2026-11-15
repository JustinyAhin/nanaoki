package rulesuggest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/rules"
	"github.com/JustinyAhin/nanaoki/internal/testfixture"
)

func TestSuggestAndDiff(t *testing.T) {
	rules.SetNames(nil)
	evs := []testfixture.Ev{
		{Agent: "codex", Sid: "a", Project: "alpha", Ts: "2026-09-01T10:00:00Z", Type: "user_msg", Text: "Fix the bug but don't push yet."},
		{Agent: "codex", Sid: "b", Project: "beta", Ts: "2026-09-02T10:00:00Z", Type: "user_msg", Text: "Commit it, do not push."},
		{Agent: "codex", Sid: "c", Project: "alpha", Ts: "2026-09-03T10:00:00Z", Type: "user_msg", Text: "Always run the e2e suite before you open a PR here."},
		{Agent: "codex", Sid: "d", Project: "alpha", Ts: "2026-09-04T10:00:00Z", Type: "user_msg", Text: "Always run the e2e suite before you open a PR here."},
		{Agent: "codex", Sid: "e", Project: "alpha", Ts: "2026-09-05T10:00:00Z", Type: "user_msg", Text: "<system-reminder>Always run the e2e suite before you open a PR here.</system-reminder>"},
	}
	db, err := ingest.OpenStoreRO(testfixture.EventsDB(t, ingest.OpenStore, evs))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	d, err := analyze.Load(db, "")
	if err != nil {
		t.Fatal(err)
	}
	rs := Suggest(d, Options{})
	if len(rs) != 2 || rs[0].Key != "commit_push_only_when_told" || rs[0].Scope != "global" {
		t.Fatalf("rules = %+v", rs)
	}
	if rs[1].Kind != "decision" || rs[1].Scope != "project" || rs[1].Project != "alpha" || rs[1].Sessions != 2 {
		t.Errorf("decision = %+v (the injected copy must not count)", rs[1])
	}
	if got := Suggest(d, Options{Before: "2026-09-04"}); len(got) != 1 {
		t.Errorf("before: %+v", got)
	}
	f := filepath.Join(t.TempDir(), "AGENTS.md")
	os.WriteFile(f, []byte("# Rules\n\n- Do not push before I tell you.\n"), 0o644)
	diff, n := Diff(f, "AGENTS.md", rs)
	if n != 1 || !strings.Contains(diff, "+- Always run the e2e suite") || strings.Contains(diff, "Commit or push") {
		t.Errorf("diff (%d):\n%s", n, diff)
	}
	b, _ := os.ReadFile(f)
	if string(b) != "# Rules\n\n- Do not push before I tell you.\n" {
		t.Error("Diff must not write the file")
	}
}
