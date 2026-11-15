// Package style prints colors and width-aware tables on a terminal, plain stable text otherwise. The hook path must not import it.
package style

import (
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-isatty"
)

// DefaultWidth is the snippet width for output that does not go to a terminal.
const DefaultWidth = 100

// Style holds the output settings. The zero value is plain text with no width limit.
type Style struct {
	Color bool
	Width int // 0 = no limit
}

// Plain is plain text with no width limit: piped output is not wrapped or cut, so scripts can read it.
var Plain = Style{}

// Detect returns colors and the terminal width when w is a terminal, NO_COLOR is not set and noColor is false.
// Other output gets Plain. $COLUMNS overrides the width; CLICOLOR_FORCE=1 forces colors (for captures).
func Detect(w io.Writer, noColor bool) Style {
	s := Plain
	f, ok := w.(*os.File)
	tty := ok && (isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()))
	if tty {
		if n := termWidth(f); n > 0 {
			s.Width = n
		}
		s.Color = !noColor && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	}
	if os.Getenv("CLICOLOR_FORCE") != "" && os.Getenv("CLICOLOR_FORCE") != "0" {
		s.Color = !noColor && os.Getenv("NO_COLOR") == ""
	}
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n > 20 {
		s.Width = n
	}
	if s.Width > 160 {
		s.Width = 160 // long lines are hard to read
	}
	return s
}

func (s Style) wrap(code, x string) string {
	if !s.Color || x == "" {
		return x
	}
	return "\x1b[" + code + "m" + x + "\x1b[0m"
}

func (s Style) Bold(x string) string    { return s.wrap("1", x) }
func (s Style) Dim(x string) string     { return s.wrap("2", x) }
func (s Style) Red(x string) string     { return s.wrap("31", x) }
func (s Style) Green(x string) string   { return s.wrap("32", x) }
func (s Style) Yellow(x string) string  { return s.wrap("33", x) }
func (s Style) Blue(x string) string    { return s.wrap("34", x) }
func (s Style) Magenta(x string) string { return s.wrap("35", x) }
func (s Style) Cyan(x string) string    { return s.wrap("36", x) }

func (s Style) Title(x string) string {
	if !s.Color {
		return strings.ToUpper(x)
	}
	return s.wrap("1;36", x)
}

func (s Style) Mark(x string) string { return s.wrap("1;33", x) }

var ansiRx = regexp.MustCompile("\x1b\\[[0-9;]*m")

func Strip(x string) string { return ansiRx.ReplaceAllString(x, "") }

// Len is the visible width of x (code points, ANSI codes left out).
func Len(x string) int { return utf8.RuneCountInString(Strip(x)) }

// Trunc cuts plain text x to n visible code points, with "…" at the end when it cuts.
func Trunc(x string, n int) string {
	if n <= 0 || utf8.RuneCountInString(x) <= n {
		return x
	}
	if n == 1 {
		return "…"
	}
	r := []rune(x)
	return strings.TrimRight(string(r[:n-1]), " ") + "…"
}

// Pad pads x (which can hold ANSI codes) to n visible columns, on the left when right is true.
func Pad(x string, n int, right bool) string {
	d := n - Len(x)
	if d <= 0 {
		return x
	}
	if right {
		return strings.Repeat(" ", d) + x
	}
	return x + strings.Repeat(" ", d)
}

func OneLine(x string) string { return strings.Join(strings.Fields(x), " ") }

// Wrap breaks x (which can hold ANSI codes) into lines of at most width visible columns, each line starting
// with indent. Width 0 gives one line.
func (s Style) Wrap(x, indent string, width int) []string {
	words := strings.Fields(x)
	if width <= 0 {
		return []string{indent + strings.Join(words, " ")}
	}
	var out []string
	line, n := indent, Len(indent)
	empty := true
	for _, wd := range words {
		l := Len(wd)
		if !empty && n+1+l > width {
			out = append(out, line)
			line, n, empty = indent, Len(indent), true
		}
		if !empty {
			line += " "
			n++
		}
		line += wd
		n += l
		empty = false
	}
	return append(out, line)
}
