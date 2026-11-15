package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestResolveDate(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	cases := map[string]string{
		"":           "",
		"today":      "2026-11-15",
		"2026-01-02": "2026-01-02",
		"1d":         "2026-11-15",
		"30d":        "2026-10-17",
		"4w":         "2026-10-19",
		"3m":         "2026-08-16",
		"1y":         "2025-11-16",
	}
	for in, want := range cases {
		got, err := resolveDate(in, now)
		if err != nil || got != want {
			t.Errorf("resolveDate(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"0d", "30", "x", "2026-1-2", "3h"} {
		if _, err := resolveDate(bad, now); err == nil {
			t.Errorf("resolveDate(%q): want error", bad)
		}
	}
}

func testDB(t *testing.T, days ...string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "e.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`create table events (ts text)`); err != nil {
		t.Fatal(err)
	}
	for _, d := range days {
		if _, err := db.Exec(`insert into events (ts) values (?)`, d+"T10:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestResolveSplit(t *testing.T) {
	now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	db := testDB(t, "2026-02-01", "2026-05-01", "2026-11-14")
	cases := map[string][4]string{
		"auto":     {"2026-02-01", "2026-08-10", "2026-08-11", "2026-11-14"},
		"last:30d": {"2026-02-01", "2026-10-15", "2026-10-16", "2026-11-14"},
		"last:3m":  {"2026-02-01", "2026-08-14", "2026-08-15", "2026-11-14"},
		"2026-02-01:2026-08-31:2026-09-01:2026-11-14": {"2026-02-01", "2026-08-31", "2026-09-01", "2026-11-14"},
	}
	for in, want := range cases {
		got, err := resolveSplit(in, now, db)
		if err != nil || got != want {
			t.Errorf("resolveSplit(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"last:1y", "last:30", "last:", "a:b", "2026-09-01:2026-08-31:2026-09-01:2026-11-14"} {
		if _, err := resolveSplit(bad, now, db); err == nil {
			t.Errorf("resolveSplit(%q): want error", bad)
		}
	}
	if _, err := resolveSplit("auto", now, testDB(t)); err == nil {
		t.Error("auto on an empty store: want error")
	}
}
