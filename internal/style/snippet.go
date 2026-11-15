package style

import "strings"

// Snippet lays out a search snippet whose matches sit between the runes open and close: white space collapsed,
// at most lines lines of width runes, starting near the first match, cut with "…". Matches are highlighted
// with Mark; plain output drops the markers.
func (s Style) Snippet(text string, open, close rune, width, lines int) []string {
	text = OneLine(text)
	var rs []rune
	var hl []bool
	on := false
	for _, r := range text {
		switch r {
		case open:
			on = true
		case close:
			on = false
		default:
			rs = append(rs, r)
			hl = append(hl, on)
		}
	}
	if width < 20 {
		width = 20
	}
	budget := width * lines
	start := 0
	first := -1
	for i, h := range hl {
		if h {
			first = i
			break
		}
	}
	lead := false
	if first > budget/3 {
		start = first - budget/6
		for start < first && rs[start] != ' ' {
			start++
		}
		for start < first && rs[start] == ' ' {
			start++
		}
		lead = true
	}
	if start == 0 && len(rs) > 0 && rs[0] == '…' {
		lead = false
	}
	rs, hl = rs[start:], hl[start:]
	if lead {
		rs = append([]rune{'…'}, rs...)
		hl = append([]bool{false}, hl...)
	}
	var out []string
	pos := 0
	for len(out) < lines && pos < len(rs) {
		end := pos + width
		last := len(out) == lines-1
		if end >= len(rs) {
			end = len(rs)
		} else {
			if last {
				end-- // room for "…"
			}
			if k := lastSpace(rs, pos, end); k > pos+width/2 {
				end = k
			}
		}
		line := s.paint(rs[pos:end], hl[pos:end])
		next := end
		for next < len(rs) && rs[next] == ' ' {
			next++
		}
		if last && next < len(rs) {
			line = strings.TrimRight(line, " ") + "…"
		}
		out = append(out, strings.TrimRight(line, " "))
		pos = next
	}
	return out
}

func lastSpace(rs []rune, from, to int) int {
	for k := to; k > from; k-- {
		if k < len(rs) && rs[k] == ' ' {
			return k
		}
	}
	return -1
}

func (s Style) paint(rs []rune, hl []bool) string {
	var b strings.Builder
	for i := 0; i < len(rs); {
		j := i
		for j < len(rs) && hl[j] == hl[i] {
			j++
		}
		seg := string(rs[i:j])
		if hl[i] {
			seg = s.Mark(seg)
		}
		b.WriteString(seg)
		i = j
	}
	return b.String()
}
