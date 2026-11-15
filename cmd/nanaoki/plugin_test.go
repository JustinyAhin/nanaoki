package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
	"github.com/JustinyAhin/nanaoki/internal/rules"
	"github.com/JustinyAhin/nanaoki/internal/rulesuggest"
	"github.com/JustinyAhin/nanaoki/internal/search"
	"github.com/JustinyAhin/nanaoki/internal/testfixture"
)

func pluginDB(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	rules.SetNames(nil)
	return testfixture.EventsDB(t, ingest.OpenStore, []testfixture.Ev{
		{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:00:00.000Z", Type: "tool_error", Text: "ls src/routes/[slug]", Out: "zsh:1: no matches found: src/routes/[slug]", Err: true},
		{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:01:00.000Z", Type: "tool_call", Text: "ls 'src/routes/[slug]'"},
		{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:02:00.000Z", Type: "user_msg", Text: "Do not commit or push until I say so."},
		{Agent: "claude", Sid: "s2", Project: "beta", Ts: "2026-09-02T10:00:00.000Z", Type: "tool_error", Text: "ls src/routes/[id]", Out: "zsh:1: no matches found: src/routes/[id]", Err: true},
		{Agent: "claude", Sid: "s2", Project: "beta", Ts: "2026-09-02T10:01:00.000Z", Type: "tool_call", Text: "ls 'src/routes/[id]'"},
		{Agent: "claude", Sid: "s2", Project: "beta", Ts: "2026-09-02T10:02:00.000Z", Type: "user_msg", Text: "Do not commit or push until I say so."},
	})
}

func TestLessonsAndHookCommands(t *testing.T) {
	db := pluginDB(t)
	var out, errb bytes.Buffer
	if err := run([]string{"lessons", "build", "--db", db}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"lessons", "list"}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "confirmed") != 2 || !strings.Contains(out.String(), "Do not commit or push") ||
		!strings.Contains(out.String(), "zsh fails on unmatched globs") {
		t.Fatalf("listed lessons: %s", out.String())
	}
	dir := t.TempDir()
	if err := run([]string{"lessons", "export", "--dir", dir}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil || len(files) != 2 {
		t.Fatalf("exported files = %v, %v", files, err)
	}
	var exported strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		exported.Write(b)
	}
	if !strings.Contains(exported.String(), "cause: zsh_glob_nomatch") || !strings.Contains(exported.String(), "Do not commit or push") {
		t.Fatalf("exported lessons: %s", exported.String())
	}

	input := `{"session_id":"live","cwd":"/Users/x/work/projects/alpha","tool_name":"Bash","tool_input":{"command":"ls src/routes/[slug]"},"error":"zsh:1: no matches found: src/routes/[slug]"}`
	out.Reset()
	args := []string{"--agent", "claude", "post-tool-use-failure"}
	if code := runHook(args, strings.NewReader(input), &out); code != 0 {
		t.Fatalf("hook exit code = %d", code)
	}
	var reply struct {
		HookSpecificOutput struct {
			HookEventName, AdditionalContext, PermissionDecision string
		}
	}
	if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
		t.Fatalf("hook reply: %v (%s)", err, out.String())
	}
	h := reply.HookSpecificOutput
	if h.HookEventName != "PostToolUseFailure" || !strings.Contains(h.AdditionalContext, "zsh fails on unmatched globs") || h.PermissionDecision != "" {
		t.Fatalf("hook reply = %+v", h)
	}
	out.Reset()
	if code := runHook(args, strings.NewReader(input), &out); code != 0 || out.Len() != 0 {
		t.Errorf("repeated hook must stay silent: code %d, %s", code, out.String())
	}
}

func TestHookErrorsStaySilent(t *testing.T) {
	for _, tc := range []struct{ name, event, input, log string }{
		{"invalid JSON", "post-tool-use-failure", "{", "bad hook JSON"},
		{"unknown event", "unknown", "{}", "unknown event"},
		{"missing index", "post-tool-use-failure", `{"session_id":"s","tool_input":{"command":"bun run build"},"error":"Error: build failed"}`, "lessons.gob"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			var out bytes.Buffer
			if code := runHook([]string{tc.event}, strings.NewReader(tc.input), &out); code != 0 || out.Len() != 0 {
				t.Fatalf("hook must fail open: code %d, %s", code, out.String())
			}
			b, err := os.ReadFile(filepath.Join(lessons.DefaultDir(), "hook.log"))
			if err != nil || !strings.Contains(string(b), tc.log) {
				t.Errorf("hook log = %q, %v; want %q", b, err, tc.log)
			}
		})
	}
}

func TestSearchCommand(t *testing.T) {
	db := pluginDB(t)
	var out, errb bytes.Buffer
	if err := run([]string{"search", "commit", "--db", db, "--agent", "claude", "--format", "json"}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	var hits []search.Result
	if err := json.Unmarshal(out.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Session != "s2" || hits[0].Pointer != "/Users/x/.claude/sessions/s2.jsonl:6" {
		t.Fatalf("search results = %+v", hits)
	}
}

func TestRulesSuggestCommand(t *testing.T) {
	db := pluginDB(t)
	paths := ingest.DefaultPaths()
	targets := []string{filepath.Join(paths.ClaudeDir, "CLAUDE.md"), filepath.Join(paths.CodexDir, "AGENTS.md")}
	const original = "# Local rules\n"
	for _, p := range targets {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out, errb bytes.Buffer
	if err := run([]string{"rules", "suggest", "--db", db, "--format", "json"}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	var proposed []rulesuggest.Rule
	if err := json.Unmarshal(out.Bytes(), &proposed); err != nil {
		t.Fatal(err)
	}
	if len(proposed) != 1 || proposed[0].Key != "commit_push_only_when_told" || proposed[0].Scope != "global" || proposed[0].Sessions != 2 {
		t.Fatalf("proposed rules = %+v", proposed)
	}
	out.Reset()
	if err := run([]string{"rules", "suggest", "--db", db}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "+- "+proposed[0].Text) != 2 {
		t.Errorf("expected a proposed rule for both agents: %s", out.String())
	}
	if err := run([]string{"rules", "suggest", "--db", db, "--write"}, &out, &errb); err == nil {
		t.Error("--write must return an error")
	}
	for _, p := range targets {
		if b, err := os.ReadFile(p); err != nil || string(b) != original {
			t.Errorf("rule file changed: %s = %q, %v", p, b, err)
		}
	}
}
