package rules

import (
	"regexp"
	"strings"
)

// A probe is a command the agent ran to find out if something exists. A non-zero exit is the answer, not a
// mistake: `cat MEMORY.md 2>/dev/null`, `test -f x`, `command -v bun`, `ls a b || true`, or a grep that
// found nothing. The hooks stay quiet on probes and the audit does not count them (unless --no-filters).
var (
	probeSilenced = regexp.MustCompile(`(^|[^0-9&])(2|&)>\s*/dev/null`)
	probeOrTrue   = regexp.MustCompile(`\|\|\s*(true|:|echo\b|printf\b)`)
	probeTestHead = regexp.MustCompile(`^\s*(test\s+-[a-zA-Z]|\[\[?\s+-[a-zA-Z]|command\s+-v\b|which\b|type\s|hash\s)`)
	probeSearch   = regexp.MustCompile(`^\s*(grep|rg|egrep|fgrep)\b`)
	// Output that shows a real failure even when the command was written as a probe.
	probeHardFail = regexp.MustCompile(`(?im)(syntax error|permission denied|command not found|no matches found|parse error|segmentation fault|traceback|panic:|fatal:|^(\(eval\)|zsh|bash|sh):\d*:? .*not found)`)
)

// IsProbe reports whether a failed shell command was only a check whose failure is expected.
// cmd is the command text; out is the error output that the tool returned.
func IsProbe(cmd, out string) bool {
	c := strings.TrimSpace(cmd)
	if c == "" {
		return false
	}
	if probeHardFail.MatchString(out) {
		return false
	}
	if probeSilenced.MatchString(c) || probeOrTrue.MatchString(c) {
		return true
	}
	// A single check command (no && or ; chain): test -f, [ -d ], command -v, which.
	single := !strings.Contains(c, "&&") && !strings.Contains(c, ";")
	if single && probeTestHead.MatchString(c) {
		return true
	}
	// grep/rg exit 1 with no output means "no match", not an error.
	if single && probeSearch.MatchString(c) && strings.TrimSpace(stripExit(out)) == "" {
		return true
	}
	return false
}

var exitLine = regexp.MustCompile(`(?im)^\s*(error: )?exit code \d+\s*$`)

func stripExit(out string) string { return exitLine.ReplaceAllString(out, "") }
