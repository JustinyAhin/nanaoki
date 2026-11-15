// Package guards holds small deterministic checks that run before a shell command (PreToolUse). Each guard
// matches a cause of rules.Lesson. A guard warns and suggests the fixed command; it blocks only when it is
// marked safe (the command fails for certain). It never changes the command.
package guards

import "strings"

// Word is one shell word with its source span and, per byte of its value, whether the byte was quoted.
type Word struct {
	Start, End int    // byte span in the command
	Value      string // the word after quote removal
	Quoted     []bool // len(Value): true when the byte came from inside quotes or after a backslash
	HasSubst   bool   // the word holds $(...), ${...}, $var or a backquote: skip it
}

// Command is one simple command: its words.
type Command struct{ Words []Word }

func isOp(c byte) bool {
	return c == ';' || c == '&' || c == '|' || c == '(' || c == ')' || c == '<' || c == '>' || c == '\n'
}

// Parse splits a command line into simple commands. It stops at a heredoc (<<), since the body is not
// shell syntax. It is a small tokenizer, not a full parser: on anything it does not understand it still
// returns words, and guards stay conservative.
func Parse(s string) []Command {
	var cmds []Command
	var cur Command
	i := 0
	flush := func() {
		if len(cur.Words) > 0 {
			cmds = append(cmds, cur)
		}
		cur = Command{}
	}
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '#' { // comment to end of line
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if isOp(c) {
			if c == '<' && i+1 < len(s) && s[i+1] == '<' {
				flush()
				return cmds // heredoc: stop
			}
			if c == '<' || c == '>' { // redirection: the next word is a file name, not an argument
				i++
				for i < len(s) && (s[i] == '>' || s[i] == '&' || s[i] == '|') {
					i++
				}
				for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
					i++
				}
				_, n := readWord(s, i)
				i = n
				continue
			}
			flush()
			i++
			continue
		}
		w, n := readWord(s, i)
		cur.Words = append(cur.Words, w)
		i = n
	}
	flush()
	return cmds
}

func readWord(s string, i int) (Word, int) {
	w := Word{Start: i}
	var val strings.Builder
	add := func(b byte, q bool) {
		val.WriteByte(b)
		w.Quoted = append(w.Quoted, q)
	}
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || isOp(c):
			w.End, w.Value = i, val.String()
			return w, i
		case c == '\\':
			if i+1 < len(s) {
				if s[i+1] != '\n' {
					add(s[i+1], true)
				}
				i += 2
			} else {
				i++
			}
		case c == '\'':
			i++
			for i < len(s) && s[i] != '\'' {
				add(s[i], true)
				i++
			}
			i++
		case c == '"':
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' && i+1 < len(s) {
					i++
				}
				if s[i] == '$' || s[i] == '`' {
					w.HasSubst = true
				}
				add(s[i], true)
				i++
			}
			i++
		case c == '$' || c == '`':
			w.HasSubst = true
			if c == '$' && i+1 < len(s) && (s[i+1] == '(' || s[i+1] == '{') {
				// copy the balanced group as one unit
				open, close := s[i+1], byte(')')
				if open == '{' {
					close = '}'
				}
				depth := 0
				for i < len(s) {
					add(s[i], true)
					if s[i] == open {
						depth++
					} else if s[i] == close {
						depth--
						if depth == 0 {
							i++
							break
						}
					}
					i++
				}
				continue
			}
			add(c, true)
			i++
		default:
			add(c, false)
			i++
		}
	}
	w.End, w.Value = i, val.String()
	return w, i
}

// singleQuote returns the value as one single-quoted shell word.
func singleQuote(v string) string { return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'" }

// replaceSpans replaces word spans in s (spans sorted, not overlapping).
func replaceSpans(s string, ws []Word, repl func(Word) string) string {
	var b strings.Builder
	last := 0
	for _, w := range ws {
		b.WriteString(s[last:w.Start])
		b.WriteString(repl(w))
		last = w.End
	}
	b.WriteString(s[last:])
	return b.String()
}
