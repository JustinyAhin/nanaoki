package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/testfixture"
)

func TestRunEndToEnd(t *testing.T) {
	home := testfixture.Home(t)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	db := filepath.Join(t.TempDir(), "events.db")
	var out, errb bytes.Buffer
	if err := run([]string{"--db", db}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errb.String(), "ingest: 7 files") {
		t.Errorf("stderr = %s", errb.String())
	}
	if !strings.Contains(out.String(), "1 of ") || !strings.Contains(out.String(), "zsh_glob_nomatch") {
		t.Errorf("report = %s", out.String())
	}
	out.Reset()
	if err := run([]string{"report", "--db", db, "--format", "json", "--agent", "claude", "--since", "2026-10-01"}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"raw_file": "~/.claude/projects/`) {
		t.Error("json report has no evidence pointers")
	}
}

func TestRunFlags(t *testing.T) {
	var out, errb bytes.Buffer
	for _, args := range [][]string{{"bogus"}, {"report", "--since", "11/01"}, {"report", "--agent", "gemini"}, {"report", "--format", "xml"}} {
		if err := run(args, &out, &errb); err == nil {
			t.Errorf("%v: want an error", args)
		}
	}
	if err := run([]string{"report", "--db", filepath.Join(t.TempDir(), "none.db")}, &out, &errb); err == nil || !strings.Contains(err.Error(), "nanaoki ingest") {
		t.Errorf("missing store error = %v", err)
	}
}

func TestRunSplit(t *testing.T) {
	home := testfixture.Home(t)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	db := filepath.Join(t.TempDir(), "events.db")
	var out, errb bytes.Buffer
	if err := run([]string{"ingest", "--db", db, "--quiet"}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	sp := "2026-08-01:2026-09-30:2026-10-01:2026-10-31"
	if err := run([]string{"report", "--db", db, "--split", sp}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"Split: TRAIN 2026-08-01 to 2026-09-30, TEST 2026-10-01 to 2026-10-31", "Method: fixed split.", "1 of 3 TEST error episodes (33%)"} {
		if !strings.Contains(s, want) {
			t.Errorf("text report has no %q:\n%s", want, s)
		}
	}
	out.Reset()
	if err := run([]string{"report", "--db", db}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Method: rolling.") || strings.Contains(out.String(), "Split: TRAIN") {
		t.Error("rolling report must say rolling and print no split dates")
	}
	out.Reset()
	if err := run([]string{"report", "--db", db, "--format", "json", "--split", sp, "--no-filters"}, &out, &errb); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"method": "split"`, `"train_start": "2026-08-01"`, `"test_end": "2026-10-31"`, `"origin_and_spread_filters": false`, `"validation"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("json has no %s", want)
		}
	}
	for _, bad := range []string{"2026-08-01:2026-09-30:2026-10-01", "2026-10-01:2026-09-30:2026-10-01:2026-10-31", "a:b:c:d"} {
		if err := run([]string{"report", "--db", db, "--split", bad}, &out, &errb); err == nil {
			t.Errorf("--split %s: want an error", bad)
		}
	}
}
