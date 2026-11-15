package rules

import "testing"

func TestIsProbe(t *testing.T) {
	probes := []struct{ cmd, out string }{
		{"ls && cat MEMORY.md 2>/dev/null; cat ~/.claude/x/MEMORY.md 2>/dev/null; git log --oneline | head -30", "Exit code 1"},
		{"test -f package.json", "Exit code 1"},
		{"[ -d node_modules ]", ""},
		{"command -v bun", "Exit code 1"},
		{"which pnpm", ""},
		{"ls src/app src/routes || true", "ls: src/app: No such file or directory"},
		{"rg -n 'TODO' src", "Exit code 1"},
		{"grep -q foo bar.txt", ""},
		{"some-cmd &>/dev/null", "Exit code 2"},
	}
	for _, p := range probes {
		if !IsProbe(p.cmd, p.out) {
			t.Errorf("IsProbe(%q, %q) = false, want true", p.cmd, p.out)
		}
	}
	real := []struct{ cmd, out string }{
		{"bun run build", "error: Script not found \"build\""},
		{"cat MEMORY.md", "cat: MEMORY.md: No such file or directory"},
		{"rg -n foo src", "rg: src: No such file or directory (os error 2)"},
		{"echo ===== done 2>/dev/null", "(eval):1: ===== not found"},
		{"ls src/[slug] 2>/dev/null", "zsh:1: no matches found: src/[slug]"},
		{"test -f a && bun test", "1 failing"},
		{"cmd 2>/dev/null", "Traceback (most recent call last):"},
		{"", "Exit code 1"},
	}
	for _, r := range real {
		if IsProbe(r.cmd, r.out) {
			t.Errorf("IsProbe(%q, %q) = true, want false", r.cmd, r.out)
		}
	}
}
