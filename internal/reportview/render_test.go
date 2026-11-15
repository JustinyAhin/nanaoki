package reportview

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/style"
	"github.com/JustinyAhin/nanaoki/internal/testfixture"
)

func build(t *testing.T) *analyze.Report {
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
	o := analyze.Options{}
	o.Paths.ClaudeDir = p.ClaudeDir
	r, err := analyze.Build(db, o)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRender(t *testing.T) {
	r := build(t)
	for _, f := range []string{"text", "md", "json"} {
		var b bytes.Buffer
		if err := Render(&b, r, f, style.Plain); err != nil {
			t.Fatal(err)
		}
		out := b.String()
		if f == "json" {
			var v analyze.Report
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
		if strings.Contains(out, "/Users/") {
			t.Errorf("%s output shows raw paths", f)
		}
	}
}
