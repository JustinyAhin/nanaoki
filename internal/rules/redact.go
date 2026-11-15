package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Redaction for lesson text. It must match the research prototype, or the eval numbers drift.
// The lookbehinds and the lookahead of the Python patterns are ported by hand (RE2 has none).

var baseRed = []struct {
	rx   *regexp.Regexp
	repl string
}{
	{py(`\b[\w.+-]+@[\w-]+\.[\w.-]+\b`, ""), "<email>"},
	// phone: handled by phoneIntl (lookbehind)
	{py(`\b(sk|pk|rk|ghp|gho|github_pat|xox[abp]|AKIA|eyJ)[\w-]{10,}`, ""), "<secret>"},
	{py(`(token|secret|password|api[_-]?key|authorization)(["\']?\s*[:=]\s*)\S+`, "(?i)"), "${1}${2}<redacted>"},
	{py(`/Users/[\w.-]+`, ""), "~"},
	{py(`\b[0-9a-f]{32,}\b`, ""), "<hex>"},
}

var (
	phoneIntlRx = py(`\+\d[\d ()-]{8,}\d`, "")
	bearerRx    = py(`\bbearer\s+[\w.~+/=-]{16,}`, "(?i)")
	skRx        = py(`\b(sk-ant-|sk-proj-|sk-)[\w-]{10,}`, "")
	longTokRx   = regexp.MustCompile(`\b[A-Za-z0-9_-]{40,}\b`)
	dbURLRx     = py(`\b(postgres(ql)?|mysql|redis|mongodb(\+srv)?)://\S+`, "(?i)")
	credURLRx   = py(`https?://[^\s/]*:[^\s/@]*@\S+`, "")
	phoneUSRx   = py(`\(?\d{3}\)?[ .-]\d{3}[ .-]\d{4}\b`, "")
	handleRx    = py(`@[A-Za-z][\w-]{2,}`, "")
	handleExtRx = py(`^[\w/.-]*\.(js|ts|svelte|json)`, "")
	greetRx     = py(`\b(hi|hey|hello|dear|thanks|thank you|cheers),?\s+[A-Z][a-z]+\b`, "(?i)")
	ipRx        = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
)

func isWordRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) }

func prevRune(s string, i int) (rune, bool) {
	if i == 0 {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return r, true
}

// subGuarded is re.sub for a pattern with a lookbehind (badPrev) and/or a check on the match (ok).
// A rejected start moves the search one rune forward, like the Python engine.
func subGuarded(s string, rx *regexp.Regexp, badPrev func(rune) bool, ok func(s string, a, b int) bool, repl string) string {
	var out strings.Builder
	pos, last := 0, 0
	for pos <= len(s) {
		loc := rx.FindStringIndex(s[pos:])
		if loc == nil {
			break
		}
		a, b := pos+loc[0], pos+loc[1]
		reject := false
		if r, has := prevRune(s, a); has && badPrev != nil && badPrev(r) {
			reject = true
		}
		if !reject && ok != nil && !ok(s, a, b) {
			reject = true
		}
		if reject {
			_, sz := utf8.DecodeRuneInString(s[a:])
			if sz == 0 {
				sz = 1
			}
			pos = a + sz
			continue
		}
		out.WriteString(s[last:a])
		out.WriteString(repl)
		last, pos = b, b
		if a == b {
			pos++
		}
	}
	out.WriteString(s[last:])
	return out.String()
}

func baseRedact(s string) string {
	s = baseRed[0].rx.ReplaceAllString(s, baseRed[0].repl)
	// (?<![\w/.-])\+\d[\d ()-]{8,}\d
	s = subGuarded(s, phoneIntlRx, func(r rune) bool { return isWordRune(r) || r == '/' || r == '.' || r == '-' }, nil, "<phone>")
	for _, x := range baseRed[1:] {
		s = x.rx.ReplaceAllString(s, x.repl)
	}
	return s
}

// longTokOK ports the two lookaheads of \b(?=[A-Za-z0-9_-]*\d)(?=[A-Za-z0-9_-]*[A-Za-z])[A-Za-z0-9_-]{40,}\b:
// the whole run from the start must hold a digit and a letter.
func longTokOK(s string, a, _ int) bool {
	digit, letter := false, false
	for i := a; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			digit = true
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			letter = true
		case c == '_' || c == '-':
		default:
			return digit && letter
		}
	}
	return digit && letter
}

// subLongTok replaces long mixed tokens. A rejected run holds no later match (its suffixes fail too), so
// FindAll on the whole string keeps the \b context right.
func subLongTok(s string) string {
	var out strings.Builder
	last := 0
	for _, m := range longTokRx.FindAllStringIndex(s, -1) {
		if !longTokOK(s, m[0], m[1]) {
			continue
		}
		out.WriteString(s[last:m[0]])
		out.WriteString("<secret>")
		last = m[1]
	}
	out.WriteString(s[last:])
	return out.String()
}

// Names to redact: set once per process with SetNames, else the defaults (login name, git user name, and
// the lines of ~/.local/share/nanaoki/private_names.txt).
var (
	namesMu sync.Mutex
	namesRx *regexp.Regexp
	namesOK bool
)

func SetNames(names []string) {
	namesMu.Lock()
	defer namesMu.Unlock()
	namesRx, namesOK = compileNames(names), true
}

func compileNames(names []string) *regexp.Regexp {
	var clean []string
	seen := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || seen[strings.ToLower(n)] {
			continue
		}
		seen[strings.ToLower(n)] = true
		clean = append(clean, n)
	}
	if len(clean) == 0 {
		return nil
	}
	sort.SliceStable(clean, func(i, j int) bool { return len(clean[i]) > len(clean[j]) })
	q := make([]string, len(clean))
	for i, n := range clean {
		q[i] = regexp.QuoteMeta(n)
	}
	return regexp.MustCompile(`(?i)\b(` + strings.Join(q, "|") + `)\b`)
}

// DefaultNames returns the login name, the git user name (and its parts) and the extra names file.
func DefaultNames() []string {
	var out []string
	if u := os.Getenv("USER"); u != "" {
		out = append(out, u)
	}
	home, _ := os.UserHomeDir()
	if b, err := os.ReadFile(filepath.Join(home, ".gitconfig")); err == nil {
		inUser := false
		for _, l := range strings.Split(string(b), "\n") {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "[") {
				inUser = strings.HasPrefix(l, "[user")
				continue
			}
			if inUser && strings.HasPrefix(l, "name") {
				if i := strings.Index(l, "="); i >= 0 {
					n := strings.Trim(strings.TrimSpace(l[i+1:]), `"`)
					out = append(out, n)
					out = append(out, strings.Fields(n)...)
				}
			}
		}
	}
	if b, err := os.ReadFile(filepath.Join(home, ".local", "share", "nanaoki", "private_names.txt")); err == nil {
		out = append(out, strings.Split(string(b), "\n")...)
	}
	return out
}

func names() *regexp.Regexp {
	namesMu.Lock()
	defer namesMu.Unlock()
	if !namesOK {
		namesRx, namesOK = compileNames(DefaultNames()), true
	}
	return namesRx
}

// Redact removes secrets, emails, phones, IPs, handles, greetings with a name, the
// home folder and private names. limit > 0 cuts to limit code points and adds "…".
func Redact(s string, limit int) string {
	s = baseRedact(s)
	s = bearerRx.ReplaceAllString(s, "Bearer <secret>")
	s = skRx.ReplaceAllString(s, "<secret>")
	s = subLongTok(s)
	s = dbURLRx.ReplaceAllString(s, "<db-url>")
	s = credURLRx.ReplaceAllString(s, "<url-with-credentials>")
	s = subGuarded(s, phoneUSRx, func(r rune) bool { return isWordRune(r) || r == '.' }, nil, "<phone>")
	s = subGuarded(s, handleRx, func(r rune) bool { return isWordRune(r) || r == '/' },
		func(s string, _, b int) bool { return !handleExtRx.MatchString(s[b:]) }, "@<handle>")
	s = greetRx.ReplaceAllString(s, "${1} Nanaoki")
	s = ipRx.ReplaceAllString(s, "<ip>")
	if rx := names(); rx != nil {
		s = rx.ReplaceAllString(s, "Nanaoki")
	}
	if limit > 0 && RuneLen(s) > limit {
		s = Trunc(s, limit) + "…"
	}
	return s
}

// Py compiles a Python re pattern for other packages (see py).
func Py(pattern, flags string) *regexp.Regexp { return py(pattern, flags) }
