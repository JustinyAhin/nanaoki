package style

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Table is an aligned text table. Number columns are right-aligned. The Flex column (default: the last)
// is cut to fit the width.
type Table struct {
	Head   []string
	Rows   [][]string
	Indent int
	Flex   int // -1 = the last
}

var numRx = regexp.MustCompile(`^(-|[-+]?[\d,.]+%?( \([\d,.]+%?\)|%| h)?)$`)

func numeric(rows [][]string, j int) bool {
	any := false
	for _, r := range rows {
		if j >= len(r) {
			continue
		}
		c := Strip(r[j])
		if c == "" {
			continue
		}
		if !numRx.MatchString(c) {
			return false
		}
		any = true
	}
	return any
}

func (s Style) Render(w io.Writer, t Table) {
	if len(t.Head) == 0 && len(t.Rows) == 0 {
		return
	}
	n := len(t.Head)
	for _, r := range t.Rows {
		n = max(n, len(r))
	}
	flex := t.Flex
	if flex < 0 || flex >= n {
		flex = n - 1
	}
	right := make([]bool, n)
	widths := make([]int, n)
	for j := range n {
		right[j] = j != flex && numeric(t.Rows, j)
	}
	all := append([][]string{t.Head}, t.Rows...)
	for _, r := range all {
		for j, c := range r {
			widths[j] = max(widths[j], Len(c))
		}
	}
	const gap = 2
	if s.Width > 0 {
		total := t.Indent + gap*(n-1)
		for j, wd := range widths {
			if j != flex {
				total += wd
			}
		}
		if room := s.Width - total; room < widths[flex] {
			widths[flex] = max(room, 12)
		}
	}
	ind := strings.Repeat(" ", t.Indent)
	for i, r := range all {
		if i == 0 && len(t.Head) == 0 {
			continue
		}
		var b strings.Builder
		b.WriteString(ind)
		for j := range n {
			c := ""
			if j < len(r) {
				c = r[j]
			}
			if j == flex && Len(c) > widths[j] {
				c = Trunc(Strip(c), widths[j])
			}
			if i == 0 {
				c = s.Dim(c)
			}
			if j == n-1 && !right[j] {
				b.WriteString(c)
			} else {
				b.WriteString(Pad(c, widths[j], right[j]))
			}
			if j < n-1 {
				b.WriteString(strings.Repeat(" ", gap))
			}
		}
		fmt.Fprintln(w, strings.TrimRight(b.String(), " "))
	}
}
