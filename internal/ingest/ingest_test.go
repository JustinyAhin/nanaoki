package ingest

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/testfixture"
)

func paths(home string) Paths {
	return Paths{Home: home, ClaudeDir: filepath.Join(home, ".claude"), CodexDir: filepath.Join(home, ".codex")}
}

func count(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func ingestFixture(t *testing.T) (string, *sql.DB, Stats) {
	t.Helper()
	home := testfixture.Home(t)
	dbPath := filepath.Join(t.TempDir(), "events.db")
	st, err := Run(Options{DB: dbPath, Paths: paths(home), Workers: 3})
	if err != nil {
		t.Fatal(err)
	}
	db, err := OpenStoreRO(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return home, db, st
}

func TestIngestFixture(t *testing.T) {
	_, db, st := ingestFixture(t)
	if st.FilesSeen != 7 {
		t.Errorf("files seen = %d, want 7 (hidden file skipped)", st.FilesSeen)
	}
	if st.RowsDeduped != 1 {
		t.Errorf("deduped = %d, want 1 (forked copy of the August session)", st.RowsDeduped)
	}
	if st.SQLiteThreads != 2 || st.SQLiteMissing != 1 {
		t.Errorf("sqlite threads %d missing %d, want 2 and 1", st.SQLiteThreads, st.SQLiteMissing)
	}
	if n := count(t, db, `select count(*) from events`); n != 33 {
		t.Errorf("events = %d, want 33", n)
	}
	bySource := map[string]int{"claude_transcript": 9, "codex_rollout": 17, "claude_history": 3, "codex_history": 2, "codex_sqlite": 2}
	for s, want := range bySource {
		if n := count(t, db, `select count(*) from events where source=?`, s); n != want {
			t.Errorf("source %s = %d, want %d", s, n, want)
		}
	}
}

func TestClaudeTranscript(t *testing.T) {
	_, db, _ := ingestFixture(t)
	var et, sig, text string
	var ec, line int
	err := db.QueryRow(`select event_type, sig, text, exit_code, raw_line from events where source='claude_transcript' and is_error=1 and tool='Bash'`).
		Scan(&et, &sig, &text, &ec, &line)
	if err != nil {
		t.Fatal(err)
	}
	if et != "tool_error" || ec != 1 || line != 4 || text != "ls src/routes/[slug]" {
		t.Errorf("bash error row = %q exit %d line %d text %q", et, ec, line, text)
	}
	if sig != "(eval):N: no matches found: src/routes/[slug]" {
		t.Errorf("sig = %q", sig)
	}
	// the call row carries the timestamp and line of the tool_use, not of the result
	if n := count(t, db, `select count(*) from events where source='claude_transcript' and event_type='commit' and exit_code=0 and is_error=0`); n != 1 {
		t.Errorf("commit rows = %d", n)
	}
	// unanswered tool_use: tool_call with is_error NULL, input dumped as JSON
	var gtext string
	if err := db.QueryRow(`select text from events where tool='Grep' and is_error is null`).Scan(&gtext); err != nil {
		t.Fatal(err)
	}
	if gtext != `{"path": "src", "pattern": "load"}` {
		t.Errorf("grep text = %q", gtext)
	}
	if n := count(t, db, `select count(*) from events where source='claude_transcript' and event_type='user_msg'`); n != 2 {
		t.Errorf("user msgs = %d, want 2 (isMeta skipped)", n)
	}
	var psig string
	if err := db.QueryRow(`select sig from events where source='claude_transcript' and event_type='user_error_paste'`).Scan(&psig); err != nil {
		t.Fatal(err)
	}
	if psig != "Error: Cannot find module <q>" {
		t.Errorf("paste sig = %q", psig)
	}
	if n := count(t, db, `select count(*) from events where side=1 and source='claude_transcript' and tool='Read' and is_error=1`); n != 1 {
		t.Errorf("subagent error rows = %d", n)
	}
	if n := count(t, db, `select count(*) from events where source='claude_transcript' and project='demo-app'`); n != 9 {
		t.Errorf("project mapping: %d rows in demo-app", n)
	}
}

func TestCodexFormats(t *testing.T) {
	_, db, _ := ingestFixture(t)
	// Feb: function_call + function_call_output (exit code in JSON metadata, or "Process exited with code")
	if n := count(t, db, `select count(*) from events where session_id=? and event_type='build_result' and is_error=1 and exit_code=1`, testfixture.CodexFeb); n != 1 {
		t.Errorf("feb failed build = %d", n)
	}
	if n := count(t, db, `select count(*) from events where session_id=? and event_type='build_result' and is_error=0 and exit_code=0`, testfixture.CodexFeb); n != 1 {
		t.Errorf("feb ok build = %d", n)
	}
	// Mar-Jul (zstd): exec_command_end, mcp_tool_call_end, patch_apply_end, compacted; codex_exec = side session
	for q, want := range map[string]int{
		`select count(*) from events where session_id=? and side=1`:                                         5,
		`select count(*) from events where session_id=? and tool='shell' and exit_code=127 and is_error=1`:  1,
		`select count(*) from events where session_id=? and tool='mcp:docs:search' and is_error=1`:          1,
		`select count(*) from events where session_id=? and tool='apply_patch' and event_type='tool_error'`: 1,
		`select count(*) from events where session_id=? and event_type='compaction'`:                        1,
		`select count(*) from events where session_id=? and project='client-x'`:                             5,
	} {
		if n := count(t, db, q, testfixture.CodexMid); n != want {
			t.Errorf("%s = %d, want %d", q, n, want)
		}
	}
	// Aug: JS exec custom tool call, exit code inside the JSON output
	var text string
	var ec int
	if err := db.QueryRow(`select text, exit_code from events where session_id=?`, testfixture.CodexAug).Scan(&text, &ec); err != nil {
		t.Fatal(err)
	}
	if text != "ls src/routes/[id] && echo ok" || ec != 1 {
		t.Errorf("aug row = %q exit %d", text, ec)
	}
	// Sep+: item_completed
	for q, want := range map[string]int{
		`select count(*) from events where session_id=? and event_type='user_msg'`:                         1,
		`select count(*) from events where session_id=? and event_type='assistant_msg'`:                    1,
		`select count(*) from events where session_id=? and tool='shell' and is_error=1 and exit_code=1`:   1,
		`select count(*) from events where session_id=? and tool='shell' and is_error=0`:                   1,
		`select count(*) from events where session_id=? and tool='mcp:docs:search' and is_error=0`:         1,
		`select count(*) from events where session_id=? and event_type='file_change' and is_error is null`: 1,
		`select count(*) from events where session_id=? and event_type='compaction'`:                       1,
		`select count(*) from events where session_id=? and cwd='/Users/alice/work/projects/other-app'`:    7,
	} {
		if n := count(t, db, q, testfixture.CodexSep); n != want {
			t.Errorf("%s = %d, want %d", q, n, want)
		}
	}
}

func TestHistoriesAndGapFill(t *testing.T) {
	_, db, _ := ingestFixture(t)
	if n := count(t, db, `select count(*) from events where source='claude_history' and event_type='user_error_paste'`); n != 1 {
		t.Errorf("claude history paste rows = %d", n)
	}
	var ts string
	if err := db.QueryRow(`select ts from events where source='claude_history' and raw_line=1`).Scan(&ts); err != nil {
		t.Fatal(err)
	}
	if ts != "2026-02-24T02:26:40.000Z" {
		t.Errorf("history ts = %q", ts)
	}
	// codex history takes the cwd of the rollout session
	if n := count(t, db, `select count(*) from events where source='codex_history' and project='demo-app'`); n != 1 {
		t.Errorf("codex history with cwd = %d", n)
	}
	if n := count(t, db, `select count(*) from events where source='codex_sqlite' and session_id=? and is_error=1 and project='gap-app'`, testfixture.CodexGapOnly); n != 1 {
		t.Errorf("gap fill error rows = %d", n)
	}
	var cause string
	if err := db.QueryRow(`select cause from events where source='codex_sqlite' and is_error=1`).Scan(&cause); err != nil {
		t.Fatal(err)
	}
	if cause != "port_in_use" {
		t.Errorf("stored cause = %q", cause)
	}
}

func TestIncremental(t *testing.T) {
	home := testfixture.Home(t)
	dbPath := filepath.Join(t.TempDir(), "events.db")
	p := paths(home)
	if _, err := Run(Options{DB: dbPath, Paths: p}); err != nil {
		t.Fatal(err)
	}
	st, err := Run(Options{DB: dbPath, Paths: p})
	if err != nil {
		t.Fatal(err)
	}
	if st.FilesParsed != 0 || st.FilesUnchanged != 7 {
		t.Errorf("second run parsed %d, unchanged %d", st.FilesParsed, st.FilesUnchanged)
	}
	// grow one file, delete another
	sep := filepath.Join(home, ".codex", "sessions", "2026", "09", "10", "rollout-2026-09-10T15-00-00-"+testfixture.CodexSep+".jsonl")
	f, err := os.OpenFile(sep, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"timestamp":"2026-09-10T15:00:09.000Z","type":"event_msg","payload":{"type":"item_completed","item":{"type":"AgentMessage","id":"i9","content":[{"type":"text","text":"Done."}]}}}` + "\n")
	f.Close()
	later := time.Now().Add(time.Minute)
	os.Chtimes(sep, later, later)
	if err := os.Remove(filepath.Join(home, ".claude", "projects", "-Users-alice-work-projects-demo-app", testfixture.ClaudeSession+".jsonl")); err != nil {
		t.Fatal(err)
	}
	st, err = Run(Options{DB: dbPath, Paths: p})
	if err != nil {
		t.Fatal(err)
	}
	if st.FilesParsed != 1 || st.FilesGone != 1 {
		t.Errorf("third run parsed %d, gone %d", st.FilesParsed, st.FilesGone)
	}
	db, err := OpenStoreRO(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if n := count(t, db, `select count(*) from events`); n != 34 {
		t.Errorf("events after update = %d, want 34 (deleted file kept, one new row)", n)
	}
	if n := count(t, db, `select count(*) from files where present=0`); n != 1 {
		t.Errorf("files marked gone = %d", n)
	}
}

func TestCodexSkip(t *testing.T) {
	cases := map[string]bool{
		`{"timestamp":"x","type":"response_item","payload":{"type":"message","role":"user"}}`:    true,
		`{"timestamp":"x","type":"response_item","payload":{"type":"function_call","name":"a"}}`: false,
		`{"timestamp":"x","type":"event_msg","payload":{"type":"token_count"}}`:                  true,
		`{"timestamp":"x","type":"event_msg","payload":{"type":"exec_command_end"}}`:             false,
		`{"timestamp":"x","type":"turn_context","payload":{}}`:                                   false,
		`{"type":"event_msg","timestamp":"x","payload":{"type":"token_count"}}`:                  false, // other layout: parse
		`{"timestamp":"x","type":"something_new","payload":{}}`:                                  true,
	}
	for line, want := range cases {
		if got := codexSkip([]byte(line)); got != want {
			t.Errorf("codexSkip(%s) = %v, want %v", line, got, want)
		}
	}
}

func TestPyDumpsAndTr(t *testing.T) {
	if got := pyDumps(map[string]any{"b": 1.0, "a": []any{"x", nil, true}, "c": "é\n"}); got != `{"a": ["x", null, true], "b": 1, "c": "é\n"}` {
		t.Errorf("pyDumps = %s", got)
	}
	if got, _ := tr("héllo", 2); got != "hé…" {
		t.Errorf("tr = %q", got)
	}
	if _, ok := tr(nil, 3); ok {
		t.Error("tr(nil) should be None")
	}
	if got := isoMs(1771900000123); got != "2026-02-24T02:26:40.123Z" {
		t.Errorf("isoMs = %s", got)
	}
}
