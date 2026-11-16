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
	// Commands whose exit 1 is a yes/no answer: not ignored, files differ, not an ancestor, no such ref.
	probeAnswer = regexp.MustCompile(`^\s*(git\s+(-C\s+\S+\s+)?(check-ignore|merge-base\s+--is-ancestor|ls-remote\s+--exit-code|diff\b.*\s--(quiet|exit-code))\b|cmp\s+-s\b)`)
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
	// The exit code of `a; b` is the exit code of b, so a check at the end of a ; chain is still a probe.
	// test -f, [ -d ], command -v, which, git check-ignore, git diff --quiet.
	if last := lastCommand(c); last != "" && (probeTestHead.MatchString(last) || probeAnswer.MatchString(last)) {
		return true
	}
	single := !strings.Contains(c, "&&") && !strings.Contains(c, ";")
	// grep/rg exit 1 with no output means "no match", not an error.
	if single && probeSearch.MatchString(c) && strings.TrimSpace(stripExit(out)) == "" {
		return true
	}
	return false
}

// lastCommand returns the part after the last top-level ; or newline, or "" when that part is a pipe or an
// && / || list (its exit code can come from another command). Quotes and $( ) are skipped.
func lastCommand(c string) string {
	start, depth := 0, 0
	var q byte
	for i := 0; i < len(c); i++ {
		ch := c[i]
		switch {
		case q != 0:
			if ch == q {
				q = 0
			} else if ch == '\\' && q == '"' {
				i++
			}
		case ch == '\\':
			i++
		case ch == '\'' || ch == '"':
			q = ch
		case ch == '(':
			depth++
		case ch == ')':
			if depth > 0 {
				depth--
			}
		case depth == 0 && (ch == ';' || ch == '\n'):
			start = i + 1
		}
	}
	last := strings.TrimSpace(c[start:])
	if strings.ContainsAny(fdRedirect.ReplaceAllString(last, ""), "|&") || strings.Contains(last, "<<") {
		return ""
	}
	return last
}

var fdRedirect = regexp.MustCompile(`\d*>&\d+|&>>?`)

var exitLine = regexp.MustCompile(`(?im)^\s*(error: )?exit code \d+\s*$`)

func stripExit(out string) string { return exitLine.ReplaceAllString(out, "") }
