package analyze

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/testfixture"
)

func build(t *testing.T, o Options) *Report {
	t.Helper()
	home := testfixture.Home(t)
	dbPath := filepath.Join(t.TempDir(), "events.db")
	p := ingest.Paths{Home: home, ClaudeDir: filepath.Join(home, ".claude"), CodexDir: filepath.Join(home, ".codex")}
	if _, err := ingest.Run(ingest.Options{DB: dbPath, Paths: p}); err != nil {
		t.Fatal(err)
	}
	db, err := ingest.OpenStoreRO(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	o.Paths.ClaudeDir = p.ClaudeDir
	r, err := Build(db, o)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// zsh_glob_nomatch: Aug (codex, open) -> Sep (codex, other-app, fixed) -> Oct (claude, demo-app, fixed).
// Only the Oct episode repeats a fix, and the fix came from another project and the other agent.
func TestRepeatAfterFix(t *testing.T) {
	r := build(t, Options{})
	a := r.RAF
	if a.Repeats != 1 || a.OtherProject != 1 || a.SameProject != 0 {
		t.Fatalf("raf = %+v", a)
	}
	if a.ByAgent["claude"].OtherAgentOnly != 1 {
		t.Errorf("by agent = %+v", a.ByAgent)
	}
	top := a.Top[0]
	if top.Key != "cause:zsh_glob_nomatch" || top.EarliestFix.Session != testfixture.CodexSep || top.EarliestFix.RawLine != 4 {
		t.Errorf("top = %+v", top)
	}
	if top.FixEvent == nil || top.FixEvent.RawLine != 5 || !strings.HasPrefix(top.FixEvent.RawFile, "~/.codex/sessions/") {
		t.Errorf("fix event = %+v", top.FixEvent)
	}
	// one minute from the Claude error to its fix
	if r.TimeLost.RepeatMinutes < 0.9 || r.TimeLost.RepeatMinutes > 1.0 {
		t.Errorf("time lost = %+v", r.TimeLost)
	}
	var found bool
	for _, it := range r.Repeated {
		if it.Key == "cause:zsh_glob_nomatch" {
			found = it.Sessions == 3 && it.Days == 3 && it.Projects == 2
		}
	}
	if !found {
		t.Errorf("repeated errors = %+v", r.Repeated)
	}
}

func TestFiltersAndRules(t *testing.T) {
	r := build(t, Options{Filter: Filter{Agent: "codex"}})
	if r.RAF.Repeats != 0 {
		t.Errorf("codex-only filter: repeats = %d", r.RAF.Repeats)
	}
	r = build(t, Options{Filter: Filter{Since: "2026-10-01"}})
	if r.RAF.Episodes == 0 || r.RAF.Repeats != 1 {
		t.Errorf("since filter keeps the October repeat and its earlier fix: %+v", r.RAF)
	}
	r = build(t, Options{})
	found := false
	for _, it := range r.Rules {
		if it.Key == "no_ai_attribution" {
			found = true
			if it.Msgs != 2 || it.Sessions != 2 {
				t.Errorf("attribution rule = %+v", it)
			}
		}
	}
	if !found {
		t.Fatal("report is missing the no_ai_attribution rule")
	}
}

func TestValidationSplit(t *testing.T) {
	r := build(t, Options{Split: &Split{"2026-02-01", "2026-08-31", "2026-09-01", "2026-11-14"}})
	q := r.Validation.Q2
	// TEST: Sep zsh (no fixed TRAIN episode: Aug was open), Oct zsh (earlier TEST fix only), plus other TEST errors
	if q["repeat_after_fix"].(int) != 0 || q["earlier_test_only"].(int) != 1 {
		t.Errorf("q2 = %v", q)
	}
}

func TestRender(t *testing.T) {
	r := build(t, Options{})
	for _, f := range []string{"text", "md", "json"} {
		var b bytes.Buffer
		if err := Render(&b, r, f); err != nil {
			t.Fatal(err)
		}
		out := b.String()
		if f == "json" {
			var v Report
			if err := json.Unmarshal(b.Bytes(), &v); err != nil {
				t.Fatalf("json: %v", err)
			}
			if !v.Filters || v.RAF.Repeats != 1 || len(v.RAF.Top) != 1 || len(v.Rules) == 0 {
				t.Fatalf("json report lost counts or rules: %+v", v)
			}
			if p := v.RAF.Top[0].EarliestFix; p.Session != testfixture.CodexSep || p.RawLine != 4 {
				t.Errorf("json evidence pointer = %+v", p)
			}
			continue
		}
		for _, want := range []string{"Coverage", "Repeated errors", "Repeat-after-fix", "time lost", "Restated rules", "Monthly trend", "Suggested fixes"} {
			if !strings.Contains(strings.ToLower(out), strings.ToLower(want)) {
				t.Errorf("%s output misses %q", f, want)
			}
		}
		if strings.Contains(out, "/Users/") {
			t.Errorf("%s output shows raw paths", f)
		}
	}
}
