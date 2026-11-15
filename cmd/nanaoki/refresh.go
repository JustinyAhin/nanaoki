package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/hook"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
)

const refreshEvery = 24 * time.Hour

func refreshLockPath() string { return filepath.Join(lessons.DefaultDir(), "refresh.lock") }

// runRefresh is ingest + lessons build, for the daily background refresh and for cron.
func runRefresh(stdout, stderr io.Writer) error {
	lock := refreshLockPath()
	_ = os.MkdirAll(filepath.Dir(lock), 0o755)
	_ = os.WriteFile(lock, []byte(fmt.Sprint(os.Getpid())), 0o644)
	defer os.Remove(lock)
	if err := run([]string{"ingest", "--quiet"}, stdout, stderr); err != nil {
		return err
	}
	return run([]string{"lessons", "build"}, stdout, stderr)
}

// maybeStartRefresh runs in the SessionStart hook. The hook must stay fast and never rebuild the index
// itself, so it only starts a detached `nanaoki refresh` when the index is older than a day.
// NANAOKI_AUTO_REFRESH=0 turns it off.
func maybeStartRefresh(now time.Time) {
	if os.Getenv("NANAOKI_AUTO_REFRESH") == "0" || testing.Testing() {
		return
	}
	if fi, err := os.Stat(lessons.IndexPath()); err == nil && now.Sub(fi.ModTime()) < refreshEvery {
		return
	}
	// A refresh that died leaves its lock; after an hour it no longer counts.
	if fi, err := os.Stat(refreshLockPath()); err == nil && now.Sub(fi.ModTime()) < time.Hour {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		hook.LogError("refresh", err)
		return
	}
	logf, err := os.OpenFile(filepath.Join(lessons.DefaultDir(), "refresh.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		hook.LogError("refresh", err)
		return
	}
	defer logf.Close()
	_ = os.WriteFile(refreshLockPath(), nil, 0o644)
	c := exec.Command(exe, "refresh")
	c.Stdout, c.Stderr = logf, logf
	detach(c)
	if err := c.Start(); err != nil {
		os.Remove(refreshLockPath())
		hook.LogError("refresh", err)
		return
	}
	_ = c.Process.Release()
}
