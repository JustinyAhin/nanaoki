package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setup codex edits the user's own ~/.codex/hooks.json: it must keep their hooks, not add ours twice,
// and remove only ours on uninstall.
func TestSetupCodexKeepsUserHooks(t *testing.T) {
	dir := t.TempDir()
	user := `{"other": 1, "hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "my-guard"}]}]}}`
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}
	var sink strings.Builder
	for i := 0; i < 2; i++ {
		if err := setupCodex(dir, false, &sink); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "hooks.json"))
	s := string(b)
	if !strings.Contains(s, "my-guard") || !strings.Contains(s, `"other"`) {
		t.Fatalf("user hooks lost:\n%s", s)
	}
	if n := strings.Count(s, "nanaoki hook --agent codex pre-tool-use"); n != 1 {
		t.Fatalf("nanaoki pre-tool-use hook %d times, want 1", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "past-sessions", "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	if err := setupCodex(dir, true, &sink); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "hooks.json"))
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "nanaoki") || !strings.Contains(string(b), "my-guard") {
		t.Fatalf("uninstall result wrong:\n%s", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "past-sessions")); !os.IsNotExist(err) {
		t.Fatal("skill not removed")
	}
}
