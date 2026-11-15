package lessons

import (
	"regexp"
	"strconv"
	"strings"
)

var pathArgRx = regexp.MustCompile(`^['"]?[\w.@~-]*/[^\s]*$|^['"]?[\w@-]+\.[a-z]{1,5}['"]?$`)

func isPathArg(tok string) bool {
	return !strings.HasPrefix(tok, "-") && !strings.Contains(tok, "://") && pathArgRx.MatchString(tok)
}

// ShortCmd shortens a fix command for lesson text. A run of 2+ file arguments becomes "<N files>": the
// agent needs the tool and the flags, and a file list from another project only adds noise and private
// names. A single path stays, because the fix is often its quoting (ls 'src/[slug]').
func ShortCmd(cmd string, max int) string {
	toks := strings.Fields(cmd)
	if n := len(toks); n > 0 && strings.HasSuffix(toks[n-1], "…") {
		// Redact cut the command: the last token is a broken path or word
		toks[n-1] = strings.TrimSuffix(toks[n-1], "…")
		if toks[n-1] == "" {
			toks = toks[:n-1]
		}
	}
	var out []string
	for i := 0; i < len(toks); {
		j := i
		for j < len(toks) && isPathArg(toks[j]) {
			j++
		}
		if j-i >= 2 {
			out = append(out, "<"+strconv.Itoa(j-i)+" files>")
			i = j
			continue
		}
		if j == i {
			j = i + 1
		}
		out = append(out, toks[i:j]...)
		i = j
	}
	s := strings.Join(out, " ")
	if len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)[:max-1]
	if k := strings.LastIndexByte(string(r), ' '); k > max/2 {
		return string(r)[:k] + " …"
	}
	return string(r) + "…"
}

func fixNote(ex *Example) string {
	day := ex.Ts
	if len(day) > 10 {
		day = day[:10]
	}
	if day == "" {
		return " (" + ex.Project + ")"
	}
	return " (" + ex.Project + ", " + day + ")"
}
