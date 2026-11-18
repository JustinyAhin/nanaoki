// Package anon hides project names, session ids, home paths and private data in output that a user wants to share.
package anon

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Anon maps real names to aliases. A nil *Anon changes nothing.
type Anon struct {
	alias map[string]string
	known []string
	rx    *regexp.Regexp
	home  string
}

// New returns an Anon that also replaces the known project names when they occur inside text.
func New(known []string) *Anon {
	a := &Anon{alias: map[string]string{}}
	a.home, _ = os.UserHomeDir()
	seen := map[string]bool{}
	for _, k := range known {
		if len(k) < 3 || k == "?" || seen[strings.ToLower(k)] || reserved[strings.ToLower(k)] {
			continue
		}
		seen[strings.ToLower(k)] = true
		a.known = append(a.known, k)
	}
	if len(a.known) > 0 {
		sort.SliceStable(a.known, func(i, j int) bool { return len(a.known[i]) > len(a.known[j]) })
		q := make([]string, len(a.known))
		for i, k := range a.known {
			q[i] = regexp.QuoteMeta(k)
			// short names are often also words ("cms", "oku"): match them only as written
			if len(k) >= 5 {
				q[i] = "(?i:" + q[i] + ")"
			}
		}
		a.rx = regexp.MustCompile(`(^|[^\p{L}\p{N}])(` + strings.Join(q, "|") + `)($|[^\p{L}\p{N}])`)
	}
	return a
}

// Words that are not private even when a project has this name. Folders such as "remote", "runs" or "can" are
// projects too, and replacing them in text gives "project-bz't". The field that names the project still gets its alias.
var reserved = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`claude codex nanaoki tmp temp home users projects work src web app apps api cli
		scratch desktop documents downloads applications can run runs remote merged merge exact set sets test tests
		timeline worktree worktrees site sites game games pdf ocr data docs main master build dist public private config
		scripts tools server client local origin branch release assets content notes lib packages backend frontend
		admin dashboard design research eval cache logs output input files images invoices witness`) {
		m[w] = true
	}
	return m
}()

func letters(n int) string {
	s := ""
	for {
		s = string(rune('a'+n%26)) + s
		n = n/26 - 1
		if n < 0 {
			return s
		}
	}
}

// Project returns the alias of a project name. Empty and "?" stay as they are.
func (a *Anon) Project(p string) string {
	if a == nil || p == "" || p == "?" {
		return p
	}
	key := strings.ToLower(p)
	if v, ok := a.alias[key]; ok {
		return v
	}
	v := "project-" + letters(len(a.alias))
	a.alias[key] = v
	return v
}

func (a *Anon) Session(s string) string {
	if a == nil || len(s) <= 6 {
		return s
	}
	return s[:6]
}

// Text redacts private data, writes the home folder as ~ and replaces known project names with their alias.
func (a *Anon) Text(s string) string {
	if a == nil || s == "" {
		return s
	}
	if a.home != "" {
		s = strings.ReplaceAll(s, a.home, "~")
		// Claude Code project folders: -Users-name-work-projects-x
		s = strings.ReplaceAll(s, strings.ReplaceAll(a.home, "/", "-"), "~")
	}
	s = rules.Redact(s, 0)
	if a.rx != nil {
		// two passes: adjacent names share the separator between them
		for range 2 {
			s = a.rx.ReplaceAllStringFunc(s, func(m string) string {
				sub := a.rx.FindStringSubmatch(m)
				return sub[1] + a.Project(sub[2]) + sub[3]
			})
		}
	}
	return s
}

// A file path: two or more slashes, or one slash and a file extension. One slash alone is often not a path
// ("hrefs/goto", "svelte/no-navigation").
var pathRx = regexp.MustCompile(`(^|[^\w$@/.~-])((?:~|\.{1,2})?/?(?:[\w.@\[\]+-]+/){2,}[\w.@\[\]+-]*|(?:~|\.{1,2})?/?[\w@\[\]+-]+/[\w@\[\]+-]+\.[a-z]{1,5}\b)`)

// Lesson is Text plus file paths: a lesson can quote a fix command whose paths name a project.
func (a *Anon) Lesson(s string) string {
	if a == nil || s == "" {
		return s
	}
	return pathRx.ReplaceAllString(a.Text(s), "${1}<path>")
}
