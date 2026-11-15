// Package testfixture builds a fake home with hand-written synthetic Claude Code and Codex logs (no real log content).
package testfixture

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

const (
	ClaudeSession = "aaaaaaaa-0000-4000-8000-000000000001"
	CodexFeb      = "019c0000-0000-7000-8000-00000000000a"
	CodexMid      = "019d0000-0000-7000-8000-00000000000b"
	CodexAug      = "019e0000-0000-7000-8000-00000000000c"
	CodexSep      = "01a00000-0000-7000-8000-00000000000d"
	CodexFork     = "019e0000-0000-7000-8000-00000000000e"
	CodexGapOnly  = "01a10000-0000-7000-8000-0000000000aa"
)

func testdata(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "ingest", "testdata", name)
}

func copyFile(t *testing.T, src, dst string, compress bool) {
	t.Helper()
	b, err := os.ReadFile(testdata(src))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if compress {
		enc, _ := zstd.NewWriter(nil)
		b = enc.EncodeAll(b, nil)
		enc.Close()
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Home builds the fake home and returns its path. Layout mirrors ~/.claude and ~/.codex.
func Home(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	cp := filepath.Join(home, ".claude", "projects", "-Users-alice-work-projects-demo-app")
	copyFile(t, "claude_session.jsonl", filepath.Join(cp, ClaudeSession+".jsonl"), false)
	copyFile(t, "claude_subagent.jsonl", filepath.Join(cp, ClaudeSession, "subagents", "agent-a1.jsonl"), false)
	copyFile(t, "claude_history.jsonl", filepath.Join(home, ".claude", "history.jsonl"), false)
	cx := filepath.Join(home, ".codex")
	copyFile(t, "codex_feb.jsonl", filepath.Join(cx, "sessions", "2026", "02", "25", "rollout-2026-02-25T09-00-00-"+CodexFeb+".jsonl"), false)
	copyFile(t, "codex_mid.jsonl", filepath.Join(cx, "sessions", "2026", "05", "10", "rollout-2026-05-10T12-00-00-"+CodexMid+".jsonl.zst"), true)
	copyFile(t, "codex_aug.jsonl", filepath.Join(cx, "sessions", "2026", "08", "11", "rollout-2026-08-11T08-00-00-"+CodexAug+".jsonl"), false)
	copyFile(t, "codex_sep.jsonl", filepath.Join(cx, "sessions", "2026", "09", "10", "rollout-2026-09-10T15-00-00-"+CodexSep+".jsonl"), false)
	// a forked copy of the August session: every line is a duplicate and must be dropped
	copyFile(t, "codex_aug.jsonl", filepath.Join(cx, "archived_sessions", "rollout-2026-08-12T08-00-00-"+CodexFork+".jsonl.zst"), true)
	// hidden files are ignored, like Python glob
	copyFile(t, "codex_sep.jsonl", filepath.Join(cx, "sessions", "2026", "09", "10", ".rollout-hidden.jsonl"), false)
	copyFile(t, "codex_history.jsonl", filepath.Join(cx, "history.jsonl"), false)
	threadDB(t, filepath.Join(cx, "thread_history_1.sqlite"))
	return home
}

func threadDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stmts := []string{
		`create table thread_items (thread_id text not null, turn_id text not null, item_id text not null,
			rollout_ordinal integer not null, created_at_ms integer not null, item_json text not null,
			item_type text not null default '', updated_at_ordinal integer not null default 0,
			started_at_ms integer, completed_at_ms integer, primary key (thread_id, turn_id, item_id))`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	rows := [][]any{
		// a thread that also has a rollout file: skipped
		{CodexSep, "t1", "i1", 1, int64(1789052400000), `{"type":"userMessage","content":[{"type":"text","text":"dup"}]}`, "userMessage"},
		// a thread with no rollout file: gap fill
		{CodexGapOnly, "t1", "i1", 1, int64(1789052400000), `{"type":"userMessage","content":[{"type":"text","text":"start the dev server"}]}`, "userMessage"},
		{CodexGapOnly, "t1", "i2", 2, int64(1789052401000), `{"type":"commandExecution","command":"bun run dev","cwd":"/Users/alice/work/projects/gap-app","exitCode":1,"status":"failed","aggregatedOutput":"error: Port 5173 is in use"}`, "commandExecution"},
	}
	for _, r := range rows {
		if _, err := db.Exec(`insert into thread_items (thread_id, turn_id, item_id, rollout_ordinal, created_at_ms, item_json, item_type) values (?,?,?,?,?,?,?)`, r...); err != nil {
			t.Fatal(err)
		}
	}
}
