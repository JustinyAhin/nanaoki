package ingest

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Paths are the input locations. Zero values are filled by DefaultPaths.
type Paths struct {
	Home      string // used to show paths as ~/...
	ClaudeDir string // $CLAUDE_CONFIG_DIR or ~/.claude
	CodexDir  string // $CODEX_HOME or ~/.codex
}

// DefaultPaths respects CLAUDE_CONFIG_DIR and CODEX_HOME.
func DefaultPaths() Paths {
	home, _ := os.UserHomeDir()
	p := Paths{Home: home, ClaudeDir: filepath.Join(home, ".claude"), CodexDir: filepath.Join(home, ".codex")}
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		p.ClaudeDir = v
	}
	if v := os.Getenv("CODEX_HOME"); v != "" {
		p.CodexDir = v
	}
	return p
}

// Rel shows a path with the home folder as ~.
func (p Paths) Rel(path string) string {
	if p.Home != "" && strings.HasPrefix(path, p.Home) {
		return "~" + path[len(p.Home):]
	}
	return path
}

type SrcFile struct {
	Path  string
	Kind  string // "claude" or "codex"
	Size  int64
	Mtime int64 // ns
}

func hidden(name string) bool { return strings.HasPrefix(name, ".") }

// Discover lists the files the Python reference globs:
// codex sessions/**/*.jsonl[.zst], archived_sessions/*.jsonl*, claude projects/*/*.jsonl and projects/*/*/subagents/*.jsonl.
// Hidden files and folders are skipped, like Python glob.
func Discover(p Paths) []SrcFile {
	var out []SrcFile
	add := func(path, kind string) {
		st, err := os.Stat(path)
		if err != nil || !st.Mode().IsRegular() {
			return
		}
		out = append(out, SrcFile{Path: path, Kind: kind, Size: st.Size(), Mtime: st.ModTime().UnixNano()})
	}
	sess := filepath.Join(p.CodexDir, "sessions")
	_ = filepath.WalkDir(sess, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path != sess && hidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() && (strings.HasSuffix(d.Name(), ".jsonl") || strings.HasSuffix(d.Name(), ".jsonl.zst")) {
			add(path, "codex")
		}
		return nil
	})
	if ents, err := os.ReadDir(filepath.Join(p.CodexDir, "archived_sessions")); err == nil {
		for _, e := range ents {
			if !hidden(e.Name()) && strings.Contains(e.Name(), ".jsonl") {
				add(filepath.Join(p.CodexDir, "archived_sessions", e.Name()), "codex")
			}
		}
	}
	proj := filepath.Join(p.ClaudeDir, "projects")
	for _, pat := range []string{"*/*.jsonl", "*/*/subagents/*.jsonl"} {
		m, _ := filepath.Glob(filepath.Join(proj, pat))
		for _, f := range m {
			rel, _ := filepath.Rel(proj, f)
			skip := false
			for _, part := range strings.Split(rel, string(filepath.Separator)) {
				if hidden(part) {
					skip = true
				}
			}
			if !skip {
				add(f, "claude")
			}
		}
	}
	// Big files first (better balance), then path for a stable order.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Size != out[j].Size {
			return out[i].Size > out[j].Size
		}
		return out[i].Path < out[j].Path
	})
	return out
}
