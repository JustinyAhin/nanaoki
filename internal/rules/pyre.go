// Package rules holds the deterministic rules ported from the Python reference in scripts/.
package rules

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// py compiles a Python `re` pattern with Go RE2.
// Python str patterns use Unicode \w, \d and \s. Go uses ASCII. This rewrite makes the classes Unicode.
// \b stays ASCII in Go (RE2 has no Unicode word boundary). Backreferences and lookarounds are not supported:
// the few Python patterns that use them are ported by hand (see quoteRepl).
func py(pattern string, flags string) *regexp.Regexp {
	return regexp.MustCompile(flags + pyToRE2(pattern))
}

func pyToRE2(p string) string {
	var b strings.Builder
	inClass := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c == '\\' && i+1 < len(p) {
			n := p[i+1]
			i++
			switch n {
			case 'w':
				if inClass {
					b.WriteString(`\p{L}\p{N}_`)
				} else {
					b.WriteString(`[\p{L}\p{N}_]`)
				}
			case 'W':
				if inClass {
					b.WriteString(`\W`) // not used inside classes by the reference patterns
				} else {
					b.WriteString(`[^\p{L}\p{N}_]`)
				}
			case 'd':
				b.WriteString(`\p{Nd}`)
			case 's':
				if inClass {
					b.WriteString(`\s\v\p{Z}\x{85}`)
				} else {
					b.WriteString(`[\s\v\p{Z}\x{85}]`)
				}
			default:
				b.WriteByte('\\')
				b.WriteByte(n)
			}
			continue
		}
		if c == '[' && !inClass {
			inClass = true
			b.WriteByte(c)
			// a ']' right after '[' or '[^' is a literal in both engines
			if i+1 < len(p) && p[i+1] == '^' {
				b.WriteByte('^')
				i++
			}
			if i+1 < len(p) && p[i+1] == ']' {
				b.WriteString(`\]`)
				i++
			}
			continue
		}
		if c == ']' && inClass {
			inClass = false
		}
		b.WriteByte(c)
	}
	return b.String()
}

// Trunc keeps the first n code points, like Python s[:n].
func Trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	i := 0
	for k := range s {
		if i == n {
			return s[:k]
		}
		i++
	}
	return s
}

// RuneLen is Python len(s).
func RuneLen(s string) int { return utf8.RuneCountInString(s) }

// Splitlines is Python str.splitlines().
func Splitlines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		switch r {
		case '\n', '\v', '\f', '\x1c', '\x1d', '\x1e', '\u0085', ' ', ' ':
			out = append(out, s[start:i])
			i += sz
			start = i
			continue
		case '\r':
			out = append(out, s[start:i])
			i += sz
			if i < len(s) && s[i] == '\n' {
				i++
			}
			start = i
			continue
		}
		i += sz
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// Strip is Python str.strip().
func Strip(s string) string { return strings.TrimFunc(s, unicode.IsSpace) }

// quoteRepl ports re.sub(r'(["\'`]).{1,80}?\1', '<q>', s): a quote char, 1-80 non-newline chars
// (the first may be the quote itself), then the same quote char. Leftmost, non-greedy, non-overlapping.
func quoteRepl(s string) string {
	rs := []rune(s)
	var b strings.Builder
	i := 0
	for i < len(rs) {
		q := rs[i]
		if q == '"' || q == '\'' || q == '`' {
			end := -1
			for k := 1; k <= 81 && i+k < len(rs); k++ {
				if rs[i+k] == '\n' {
					break
				}
				if k >= 2 && rs[i+k] == q {
					end = i + k
					break
				}
			}
			// k>=2 above: the closing quote is at offset k, with k-1 >= 1 chars between.
			// The char at offset 1 may itself be the quote (consumed by '.').
			if end >= 0 {
				b.WriteString("<q>")
				i = end + 1
				continue
			}
		}
		b.WriteRune(rs[i])
		i++
	}
	return b.String()
}
