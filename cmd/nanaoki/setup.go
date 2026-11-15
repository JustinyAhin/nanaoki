package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/plugin"
)

const setupUsage = `Usage:
  nanaoki setup claude [--from SRC]   install the Claude Code plugin (runs the claude CLI)
                                      SRC: GitHub repo or local folder (default JustinyAhin/nanaoki)
  nanaoki setup codex                 add the hooks to ~/.codex/hooks.json and copy the skills
  nanaoki setup codex --uninstall     remove them again (other hooks stay)
`

func runSetup(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, setupUsage)
		return nil
	}
	sub, args := args[0], args[1:]
	fset := flag.NewFlagSet("setup "+sub, flag.ContinueOnError)
	fset.SetOutput(stderr)
	fset.Usage = func() { fmt.Fprint(stderr, setupUsage) }
	from := fset.String("from", "JustinyAhin/nanaoki", "")
	uninstall := fset.Bool("uninstall", false, "")
	if err := fset.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	switch sub {
	case "claude":
		return setupClaude(*from, *uninstall, stdout, stderr)
	case "codex":
		return setupCodex(ingest.DefaultPaths().CodexDir, *uninstall, stdout)
	}
	return fmt.Errorf("unknown setup target %q\n\n%s", sub, setupUsage)
}

func setupClaude(from string, uninstall bool, stdout, stderr io.Writer) error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude is not on PATH")
	}
	steps := [][]string{{"plugin", "marketplace", "add", from}, {"plugin", "install", "nanaoki@nanaoki", "--scope", "user"}}
	if uninstall {
		steps = [][]string{{"plugin", "uninstall", "nanaoki@nanaoki"}, {"plugin", "marketplace", "remove", "nanaoki"}}
	}
	for _, s := range steps {
		fmt.Fprintln(stdout, "$ claude", strings.Join(s, " "))
		c := exec.Command("claude", s...)
		c.Stdout, c.Stderr = stdout, stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("claude %s: %w", strings.Join(s, " "), err)
		}
	}
	if !uninstall {
		fmt.Fprintln(stdout, "Done. Start a new Claude Code session to load the hooks.")
	}
	return nil
}

func setupCodex(codexDir string, uninstall bool, stdout io.Writer) error {
	hooksPath := filepath.Join(codexDir, "hooks.json")
	old, err := os.ReadFile(hooksPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var ours map[string]any
	if !uninstall {
		b, err := plugin.Files.ReadFile("codex/hooks.json")
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &ours); err != nil {
			return err
		}
	}
	merged, err := mergeCodexHooks(old, ours)
	if err != nil {
		return fmt.Errorf("%s: %w", hooksPath, err)
	}
	if !bytes.Equal(old, merged) {
		if err := os.MkdirAll(codexDir, 0o755); err != nil {
			return err
		}
		if len(old) > 0 {
			if err := os.WriteFile(hooksPath+".bak", old, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "backup: %s.bak\n", hooksPath)
		}
		if err := os.WriteFile(hooksPath, merged, 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintln(stdout, "hooks:", hooksPath)

	skillsDir := filepath.Join(codexDir, "skills")
	sub, err := fs.Sub(plugin.Files, "claude-code/skills")
	if err != nil {
		return err
	}
	names, err := fs.ReadDir(sub, ".")
	if err != nil {
		return err
	}
	for _, n := range names {
		dst := filepath.Join(skillsDir, n.Name())
		if uninstall {
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
			continue
		}
		one, err := fs.Sub(sub, n.Name())
		if err != nil {
			return err
		}
		if err := overwriteFS(dst, one); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "skill:", dst)
	}
	if !uninstall {
		fmt.Fprintln(stdout, "Done. In Codex, open /hooks and trust the nanaoki hooks: Codex skips hooks it has not reviewed.")
	}
	return nil
}

// overwriteFS writes every file of src into dst. os.CopyFS refuses to replace files, and an
// upgrade must replace the skill text.
func overwriteFS(dst string, src fs.FS) error {
	return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o644)
	})
}

// mergeCodexHooks removes every handler whose command starts with "nanaoki hook" from the user's
// hooks.json, then adds the handlers in ours (nil = uninstall). Other handlers and keys stay as they are.
func mergeCodexHooks(old []byte, ours map[string]any) ([]byte, error) {
	doc := map[string]any{}
	if len(bytes.TrimSpace(old)) > 0 {
		if err := json.Unmarshal(old, &doc); err != nil {
			return nil, fmt.Errorf("not valid JSON, fix it by hand first: %w", err)
		}
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	for event, v := range hooks {
		groups, _ := v.([]any)
		var keep []any
		for _, g := range groups {
			gm, ok := g.(map[string]any)
			if !ok {
				keep = append(keep, g)
				continue
			}
			hs, _ := gm["hooks"].([]any)
			var left []any
			for _, h := range hs {
				if hm, ok := h.(map[string]any); ok {
					if c, _ := hm["command"].(string); strings.HasPrefix(strings.TrimSpace(c), "nanaoki hook") {
						continue
					}
				}
				left = append(left, h)
			}
			if len(hs) > 0 && len(left) == 0 {
				continue
			}
			gm["hooks"] = left
			keep = append(keep, gm)
		}
		if len(keep) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = keep
		}
	}
	if add, _ := ours["hooks"].(map[string]any); add != nil {
		for event, v := range add {
			groups, _ := v.([]any)
			cur, _ := hooks[event].([]any)
			hooks[event] = append(cur, groups...)
		}
	}
	if len(hooks) == 0 && ours == nil {
		delete(doc, "hooks")
	} else {
		doc["hooks"] = hooks
	}
	if len(doc) == 0 && len(old) == 0 {
		return nil, nil
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
