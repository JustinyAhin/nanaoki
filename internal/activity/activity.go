// Package activity writes and reads the local activity logs of the live test: guard events, hook calls and
// user notes. The hook path imports it, so it stays small.
package activity

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/JustinyAhin/nanaoki/internal/lessons"
	"github.com/JustinyAhin/nanaoki/internal/rules"
)

func GuardLogPath() string { return filepath.Join(lessons.DefaultDir(), "guards.jsonl") }
func CallLogPath() string  { return filepath.Join(lessons.DefaultDir(), "calls.jsonl") }
func NotesPath() string    { return filepath.Join(lessons.DefaultDir(), "notes.jsonl") }
func HookLogPath() string  { return filepath.Join(lessons.DefaultDir(), "hook.log") }

// MaxCmd caps stored commands. Stats compares commands after the same Cmd, so the cap does not break matching.
const MaxCmd = 400

// Cmd is the stored form of a command: redacted, on one line, capped.
func Cmd(s string) string {
	return rules.Redact(strings.Join(strings.Fields(s), " "), MaxCmd)
}

type GuardEntry struct {
	Ts       string  `json:"ts"`
	Agent    string  `json:"agent"`
	Session  string  `json:"session"`
	Project  string  `json:"project"`
	Guard    string  `json:"guard"`
	Cause    string  `json:"cause"`
	Decision string  `json:"decision"`
	Cmd      string  `json:"cmd"`
	Fixed    string  `json:"fixed,omitempty"`
	Ms       float64 `json:"ms"`
}

// CallEntry is one hook call. Ms is the time inside the process (no process start).
type CallEntry struct {
	Ts      string  `json:"ts"`
	Agent   string  `json:"agent"`
	Event   string  `json:"event"`
	Session string  `json:"session,omitempty"`
	Project string  `json:"project,omitempty"`
	Ms      float64 `json:"ms"`
	Out     bool    `json:"out,omitempty"`
	Err     bool    `json:"err,omitempty"`
}

type Note struct {
	ID      string `json:"id"`
	Ts      string `json:"ts"`
	Project string `json:"project"`
	Agent   string `json:"agent,omitempty"`
	Session string `json:"session,omitempty"`
	Lesson  string `json:"lesson,omitempty"`
	Guard   string `json:"guard,omitempty"`
	Tag     string `json:"tag,omitempty"` // useful, noise, wrong, missing
	Text    string `json:"text"`
}

var Tags = []string{"useful", "noise", "wrong", "missing"}

func NoteID(ts, text string) string {
	h := sha1.Sum([]byte(ts + "\x00" + text))
	return "N-" + hex.EncodeToString(h[:])[:8]
}

// Append adds JSON lines to path. The hooks ignore its error (they fail open).
func Append[T any](path string, es ...T) error {
	if len(es) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	var b []byte
	for _, e := range es {
		j, err := json.Marshal(e)
		if err != nil {
			f.Close()
			return err
		}
		b = append(append(b, j...), '\n')
	}
	// one write per call: concurrent hooks append whole lines
	_, err = f.Write(b)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// Line is one decoded log line with its 1-based line number (the pointer that stats prints).
type Line[T any] struct {
	N int `json:"line"`
	V T   `json:"entry"`
}

// Read decodes every valid line of path. A missing file gives no lines and no error.
func Read[T any](path string) ([]Line[T], error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Line[T]
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	n := 0
	for sc.Scan() {
		n++
		var v T
		if json.Unmarshal(sc.Bytes(), &v) == nil {
			out = append(out, Line[T]{n, v})
		}
	}
	return out, sc.Err()
}
