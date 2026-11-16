package hook

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/guards"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
)

// Synthetic index: one cause lesson, one signature lesson, one rule restated in two sessions.
func testIndex() *lessons.Index {
	ls := []*lessons.Lesson{
		{ID: "cause:aaaa", Kind: "cause", Key: "zsh_glob_nomatch", Text: "zsh fails on unmatched globs. Quote them.",
			Projects: map[string]int{"demo": 2}, Heads: []string{"ls"}, Sessions: 2},
		{ID: "sig:bbbb", Kind: "sig", Key: "Error: widget config missing key <q> in project settings", Text: "Error `widget` was fixed by: `make config`.",
			Projects: map[string]int{"demo": 1}, Sessions: 1},
		{ID: "pref:cccc", Kind: "pref", Key: "do not commit or push until i say so", Text: "Do not commit or push until I say so.",
			Projects: map[string]int{"demo": 3}, Sessions: 3},
	}
	st := lessons.NewStore(ls, nil)
	return &lessons.Index{Lessons: ls, IDF: st.IDF, RepoStacks: map[string][]string{"demo": {"sveltekit"}}}
}

func testDeps(t *testing.T) Deps {
	dir := t.TempDir()
	ix := testIndex()
	return Deps{
		Index:    func() (*lessons.Index, error) { return ix, nil },
		StateDir: filepath.Join(dir, "state"),
		ShownLog: filepath.Join(dir, "shown.jsonl"),
		Env: func(cwd string) *guards.Env {
			return &guards.Env{Cwd: cwd, Shell: "/bin/zsh", LookPath: func(string) (string, error) { return "", errors.New("no") }}
		},
	}
}

func ctxOf(t *testing.T, o Output) string {
	t.Helper()
	if o == nil {
		return ""
	}
	h := o["hookSpecificOutput"].(map[string]any)
	s, _ := h["additionalContext"].(string)
	return s
}

func raw(s string) json.RawMessage { return json.RawMessage(s) }

func TestFailureShowsLessonOnce(t *testing.T) {
	d := testDeps(t)
	in := Input{SessionID: "s1", Cwd: "/Users/x/work/projects/demo", ToolName: "Bash", ToolInput: raw(`{"command":"ls src/[slug]"}`),
		Error: "Exit code 1\n(eval):1: no matches found: src/[slug]"}
	o, err := Handle("claude", "PostToolUseFailure", in, d)
	if err != nil || !strings.Contains(ctxOf(t, o), "[L1] zsh fails on unmatched globs") {
		t.Fatalf("got %v %v", o, err)
	}
	if o, _ := Handle("claude", "PostToolUseFailure", in, d); o != nil {
		t.Errorf("a lesson shows once per session, got %v", o)
	}
	b, _ := os.ReadFile(d.ShownLog)
	if !strings.Contains(string(b), `"lesson":"cause:aaaa"`) {
		t.Errorf("shown log: %s", b)
	}
	in.SessionID = "s2"
	in.Error = "Exit code 1\nall good but different"
	if o, _ := Handle("claude", "PostToolUseFailure", in, d); o != nil {
		t.Errorf("no match must print nothing, got %v", o)
	}
}

func TestPreToolUse(t *testing.T) {
	d := testDeps(t)
	o, _ := Handle("claude", "PreToolUse", Input{SessionID: "s", Cwd: t.TempDir(), ToolName: "Bash", ToolInput: raw(`{"command":"echo ====="}`)}, d)
	h := o["hookSpecificOutput"].(map[string]any)
	if h["permissionDecision"] != "deny" || !strings.Contains(h["permissionDecisionReason"].(string), "echo '====='") {
		t.Errorf("zsh_eq must block with the fixed command: %v", h)
	}
	o, _ = Handle("codex", "PreToolUse", Input{SessionID: "s", Cwd: t.TempDir(), ToolName: "Bash", ToolInput: raw(`{"command":["zsh","-lc","curl https://x.test/?a=1"]}`)}, d)
	h = o["hookSpecificOutput"].(map[string]any)
	if h["permissionDecision"] != "deny" || !strings.Contains(h["permissionDecisionReason"].(string), "'https://x.test/?a=1'") {
		t.Errorf("unmatched glob in zsh must block with the fixed command: %v", h)
	}
	// bash passes an unmatched glob through as text: no zsh guard, even when the login shell is zsh
	if o, _ := Handle("codex", "PreToolUse", Input{SessionID: "s", Cwd: t.TempDir(), ToolName: "Bash", ToolInput: raw(`{"command":["bash","-lc","curl https://x.test/?a=1"]}`)}, d); o != nil {
		t.Errorf("bash command: %v", o)
	}
	if o, _ := Handle("claude", "PreToolUse", Input{SessionID: "s", Cwd: t.TempDir(), ToolName: "Read", ToolInput: raw(`{"file_path":"x"}`)}, d); o != nil {
		t.Errorf("non-shell tool: %v", o)
	}
}

func TestCodexPostToolUse(t *testing.T) {
	d := testDeps(t)
	base := Input{SessionID: "c1", Cwd: "/Users/x/work/projects/demo", ToolName: "Bash", ToolInput: raw(`{"command":"ls a?"}`)}
	in := base
	in.ToolResponse = raw(`{"exit_code":0,"output":"(eval):1: no matches found: a?"}`)
	if o, _ := Handle("codex", "PostToolUse", in, d); o != nil {
		t.Errorf("exit 0 is not a failure: %v", o)
	}
	in.ToolResponse = raw(`"Exit code: 1\nWall time: 0.1 seconds\nOutput:\n(eval):1: no matches found: a?"`)
	if o, _ := Handle("codex", "PostToolUse", in, d); !strings.Contains(ctxOf(t, o), "[L1]") {
		t.Errorf("text exit code: %v", o)
	}
}

func TestFirstPromptStrictStart(t *testing.T) {
	d := testDeps(t)
	if o, _ := Handle("claude", "SessionStart", Input{SessionID: "p1", Cwd: "/Users/x/work/projects/demo", Source: "startup"}, d); o != nil {
		t.Errorf("SessionStart has no text yet: %v", o)
	}
	in := Input{SessionID: "p1", Cwd: "/Users/x/work/projects/demo", Prompt: "Fix the header. Do not commit or push until I say so."}
	if o, _ := Handle("claude", "UserPromptSubmit", in, d); !strings.Contains(ctxOf(t, o), "Do not commit or push") {
		t.Errorf("first prompt: %v", o)
	}
	if o, _ := Handle("claude", "UserPromptSubmit", in, d); o != nil {
		t.Errorf("second prompt: %v", o)
	}
	in = Input{SessionID: "p2", Cwd: "/Users/x/work/projects/other", Prompt: "Do not commit or push until I say so."}
	if o, _ := Handle("claude", "UserPromptSubmit", in, d); o != nil {
		t.Errorf("other repo with no shared stack: %v", o)
	}
}

func TestIndexErrorFailsOpen(t *testing.T) {
	d := testDeps(t)
	d.Index = func() (*lessons.Index, error) { return nil, errors.New("no index") }
	o, err := Handle("claude", "PostToolUseFailure", Input{SessionID: "e", ToolInput: raw(`{"command":"x"}`), Error: "boom"}, d)
	if o != nil || err == nil {
		t.Errorf("want nil output and an error to log, got %v %v", o, err)
	}
}
