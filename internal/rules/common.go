package rules

import (
	"regexp"
	"strings"
)

// Project maps a cwd to a project name (port of common.project).
func Project(cwd string) string {
	if cwd == "" {
		return "?"
	}
	cwd = strings.ReplaceAll(cwd, "file://", "")
	parts := strings.Split(strings.TrimRight(cwd, "/"), "/")
	for _, key := range []string{"projects", "consulting"} {
		for i, p := range parts {
			if p == key {
				if i+1 < len(parts) {
					return parts[i+1]
				}
				return "?"
			}
		}
	}
	if strings.Contains(cwd, "scratch-workspaces") || strings.Contains(cwd, "/scratch") || strings.Contains(cwd, "/tmp/") || strings.HasPrefix(cwd, "/private/") {
		return "scratch"
	}
	if len(parts) <= 3 {
		return "home"
	}
	return parts[len(parts)-1]
}

// ---------- error signatures ----------

var KEY = py(`(error|failed|failure|cannot|can\'t|not found|denied|refused|invalid|unknown|missing|no such|`+
	`unable|timed? ?out|exception|fatal|forbidden|unauthori[sz]ed|not allowed|requires|blocked|`+
	`expected|undefined|conflict|rejected|not permitted|ENOENT|EACCES|429|401|403|404|500|502|503)`, "(?i)")

var skipRx = py(`^(exit code \d+|\s*$|warning: truncated|total output lines|script (completed|failed)|wall time|`+
	`output:|chunk id|process exited|original token count|script error:)`, "(?i)")

var (
	ansiRx  = py(`\x1b\[[0-9;]*m`, "")
	urlRx   = py(`https?://\S+`, "")
	pathRx  = py(`(~|\.{0,2})?(/[\w.@+-]+){2,}`, "")
	hexRx   = py(`\b[0-9a-f]{7,}\b`, "")
	emailRx = py(`\b[\w.+-]+@[\w.-]+\.\w+\b`, "")
	digitRx = py(`\d+`, "")
	wsRx    = py(`\s+`, "")
)

// Norm normalizes an error line (port of common.norm).
func Norm(s string) string {
	s = ansiRx.ReplaceAllLiteralString(s, "")
	s = urlRx.ReplaceAllLiteralString(s, "<url>")
	s = pathRx.ReplaceAllLiteralString(s, "<path>")
	s = hexRx.ReplaceAllLiteralString(s, "<hex>")
	s = emailRx.ReplaceAllLiteralString(s, "<email>")
	s = digitRx.ReplaceAllLiteralString(s, "N")
	s = quoteRepl(s)
	s = Strip(wsRx.ReplaceAllLiteralString(s, " "))
	return Trunc(s, 140)
}

// Signature returns the normalized first error-like line (port of common.signature).
func Signature(text string) string {
	var lines []string
	for _, l := range Splitlines(text) {
		if !skipRx.MatchString(Strip(l)) {
			lines = append(lines, l)
		}
	}
	for _, l := range lines {
		if KEY.MatchString(l) {
			return Norm(l)
		}
	}
	if len(lines) > 0 {
		return Norm(lines[0])
	}
	return "<empty>"
}

// ---------- command classes ----------

var (
	testRx = py(`\b(vitest|jest|pytest|playwright test|bun test|(npm|pnpm|yarn|bun)( run)? test\b|test:(unit|e2e|int)|`+
		`go test|cargo test|phpunit|pest\b|rspec)`, "(?i)")
	buildRx = py(`\b(vite build|(npm|pnpm|yarn|bun)( run)? (build|check|lint|typecheck)\b|next build|\btsc\b|`+
		`svelte-check|svelte-kit sync|astro build|cargo build|go build|eslint)`, "(?i)")
	deployRx = py(`\b(wrangler (deploy|pages deploy|versions deploy)|netlify deploy|vercel (deploy|--prod)|`+
		`fly deploy|kamal deploy|railway up|(npm|bun|pnpm)( run)? deploy\b)`, "(?i)")
	commitRx = py(`\bgit (-C \S+ )?commit\b`, "")
	pushRx   = py(`\bgit (-C \S+ )?push\b`, "")
	prRx     = py(`\bgh pr (create|edit)\b`, "")
)

// CommandClass ports common.command_class.
func CommandClass(c string) string {
	switch {
	case commitRx.MatchString(c):
		return "commit"
	case pushRx.MatchString(c):
		return "push"
	case prRx.MatchString(c):
		return "pr"
	case deployRx.MatchString(c):
		return "deploy"
	case testRx.MatchString(c):
		return "test_result"
	case buildRx.MatchString(c):
		return "build_result"
	}
	return ""
}

// EventTypeForCall ports common.event_type_for_call.
func EventTypeForCall(cmd string, isError bool) string {
	if k := CommandClass(cmd); k != "" {
		return k
	}
	if isError {
		return "tool_error"
	}
	return "tool_call"
}

// ---------- paste detection (build_event_store.paste_error_rows) ----------

var (
	pasteLineRx = py(`(error|failed|exception|exit|HTTP \d{3}|fatal)`, "(?i)")
)

// PasteError returns (text, sig, ok) for a user message that pastes an error log.
func PasteError(text string) (string, string, bool) {
	if RuneLen(text) < 80 {
		return "", "", false
	}
	var lines []string
	for _, l := range Splitlines(text) {
		if KEY.MatchString(l) && pasteLineRx.MatchString(l) {
			lines = append(lines, l)
		}
	}
	if len(lines) < 1 || strings.Count(text, "\n") < 3 {
		return "", "", false
	}
	n5 := lines
	if len(n5) > 5 {
		n5 = n5[:5]
	}
	sig := Signature(strings.Join(n5, "\n"))
	n8 := lines
	if len(n8) > 8 {
		n8 = n8[:8]
	}
	return strings.Join(n8, "\n"), sig, true
}

// ---------- repeat_after_fix helpers ----------

var genericSigRx = py(`^(<empty>|<path>|<q>.*|\{|---|N.*|✓ Done|[^a-zA-Z]*)$|^(const|let|var|import|export|return|type|if|for|async|await|function)\b|`+
	`=>|^[\w.]+\($|throw new Error\(<q>\)|^\W|^(Checking formatting|Command:|COMMAND PID)`, "")
var word3Rx = regexp.MustCompile(`[A-Za-z]{3,}`)

// Informative ports repeat_after_fix.informative.
func Informative(sig string) bool {
	return sig != "" && !genericSigRx.MatchString(sig) && len(word3Rx.FindAllStringIndex(sig, -1)) >= 3
}

var (
	cdPrefixRx  = py(`^(cd \S+ (&&|;) ?)+`, "")
	envPrefixRx = py(`^(\w+=\S+ )+`, "")
	wordRx      = py(`^[\w./@:-]+`, "")
	word2Rx     = py(`^[\w./@:-]+\s+(?:run\s+)?([\w.:@/-]+)`, "")
	runners     = map[string]bool{"npx": true, "bunx": true, "bun": true, "npm": true, "pnpm": true, "yarn": true, "uv": true,
		"python3": true, "python": true, "node": true, "git": true, "gh": true}
)

// Head ports repeat_after_fix.head: the command head used to match a later success.
func Head(tool, cmd string) string {
	if tool != "" && tool != "shell" && tool != "Bash" {
		return tool
	}
	c := Strip(cmd)
	c = cdPrefixRx.ReplaceAllLiteralString(c, "")
	c = envPrefixRx.ReplaceAllLiteralString(c, "")
	w := ""
	if m := wordRx.FindString(c); m != "" {
		parts := strings.Split(m, "/")
		w = parts[len(parts)-1]
	}
	if runners[w] {
		if m := word2Rx.FindStringSubmatch(c); m != nil {
			w += " " + m[1]
		}
	}
	return w
}
