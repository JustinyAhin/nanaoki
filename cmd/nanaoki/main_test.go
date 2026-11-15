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
	for _, want := range []string{"Method: fixed split.", "1 of 3 TEST error episodes (33%)"} {
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
}
