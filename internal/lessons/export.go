package lessons

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// GuardCauses are the causes that a PreToolUse guard checks before the command runs (enforce: hook).
var GuardCauses = map[string]string{
	"zsh_glob_nomatch": "zsh_glob",
	"zsh_eq_quote":     "zsh_eq",
	"prettier_check":   "format_before_commit",
	"node_version":     "node_version",
	"ssh_vps_user":     "ssh_user",
}

// StatusOf is candidate (fix or rule seen in one session) or confirmed (2+ sessions, the confidence rule of
// the strict session-start trigger). "retired" is set by hand in the markdown file and kept on export.
func StatusOf(L *Lesson) string {
	if L.Sessions >= StartMinSessions {
		return "confirmed"
	}
	return "candidate"
}

// Global reports a lesson that holds outside one project: an env/stack cause, or a signature or rule seen in
// 2+ projects.
func Global(L *Lesson) bool {
	if L.Kind == "cause" {
		return true
	}
	return len(L.Projects) >= 2
}

// ShownLogPath is the log of lessons the hooks showed (one JSON line each).
func ShownLogPath() string { return filepath.Join(DefaultDir(), "shown.jsonl") }

type ShownEntry struct {
	Ts      string `json:"ts"`
	Lesson  string `json:"lesson"`
	Event   string `json:"event"`
	Agent   string `json:"agent"`
	Session string `json:"session"`
	Project string `json:"project"`
}

// AppendShown adds lines to the shown log. Errors are ignored by the caller (the hooks fail open).
func AppendShown(path string, es []ShownEntry) error {
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
	defer f.Close()
	for _, e := range es {
		b, _ := json.Marshal(e)
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

type shownStat struct {
	n     int
	first string
}

func readShown(path string) map[string]shownStat {
	out := map[string]shownStat{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e ShownEntry
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		s := out[e.Lesson]
		s.n++
		if s.first == "" || e.Ts < s.first {
			s.first = e.Ts
		}
		out[e.Lesson] = s
	}
	return out
}

type Score struct {
	Shown        int
	RepeatsAfter int // episodes with this key that started after the first time a hook showed the lesson
	LastSeen     string
}

func scoreOf(L *Lesson, sh shownStat) Score {
	s := Score{Shown: sh.n, LastSeen: day(L.LastTs)}
	if sh.first != "" {
		for _, ts := range L.EpisodeTs {
			if ts > sh.first {
				s.RepeatsAfter++
			}
		}
	}
	return s
}

func day(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

type ExportOptions struct {
	Dir      string // target folder ("" = default)
	Project  bool   // project lessons of Cwd into Cwd/.lessons
	Cwd      string
	ShownLog string
}

type ExportResult struct {
	Dir                         string
	Written, Retired, Unchanged int
}

var retiredRx = regexp.MustCompile(`(?m)^status:\s*retired\s*$`)

// Export writes one markdown file per confirmed lesson. Global lessons go to the global folder; with
// Project, the lessons of that project only (not the global ones) go to <cwd>/.lessons. A file whose
// status was set to retired by hand is kept as it is.
func Export(ix *Index, o ExportOptions) (ExportResult, error) {
	res := ExportResult{Dir: o.Dir}
	proj := ""
	if o.Project {
		proj = rules.Project(o.Cwd)
		if res.Dir == "" {
			res.Dir = filepath.Join(o.Cwd, ".lessons")
		}
	} else if res.Dir == "" {
		res.Dir = filepath.Join(DefaultDir(), "lessons")
	}
	if err := os.MkdirAll(res.Dir, 0o755); err != nil {
		return res, err
	}
	shown := readShown(o.ShownLog)
	for _, L := range ix.Lessons {
		if StatusOf(L) != "confirmed" {
			continue
		}
		if o.Project {
			if Global(L) || L.Projects[proj] == 0 {
				continue
			}
		} else if !Global(L) {
			continue
		}
		path := filepath.Join(res.Dir, strings.Replace(L.ID, ":", "-", 1)+".md")
		old, err := os.ReadFile(path)
		if err == nil && retiredRx.Match(old) {
			res.Retired++
			continue
		}
		md := Markdown(L, scoreOf(L, shown[L.ID]))
		if err == nil && string(old) == md {
			res.Unchanged++
			continue
		}
		if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
			return res, err
		}
		res.Written++
	}
	return res, nil
}

// q writes a YAML double-quoted string (JSON syntax, without HTML escapes).
func q(s string) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(b.String(), "\n")
}

func HomeTilde(p string) string {
	if h, err := os.UserHomeDir(); err == nil && h != "" && strings.HasPrefix(p, h+"/") {
		return "~" + p[len(h):]
	}
	return p
}

func qList(v []string) string {
	parts := make([]string, len(v))
	for i, s := range v {
		parts[i] = q(s)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// Markdown renders one lesson file: YAML front matter, then the lesson text.
func Markdown(L *Lesson, sc Score) string {
	var b strings.Builder
	w := func(s ...string) {
		for _, x := range s {
			b.WriteString(x)
		}
		b.WriteByte('\n')
	}
	w("---")
	w("id: ", q(L.ID))
	w("kind: ", L.Kind)
	w("when:")
	switch L.Kind {
	case "cause":
		w("  event: tool_failure")
		w("  cause: ", L.Key)
	case "sig":
		w("  event: tool_failure")
		w("  signature: ", q(L.Key))
	default:
		w("  event: session_start")
		w("  match: first message overlaps this rule (idf >= 0.25), same repo or shared stack")
	}
	if len(L.Heads) > 0 {
		w("  commands: ", qList(L.Heads))
	}
	projs := make([]string, 0, len(L.Projects))
	for p := range L.Projects {
		projs = append(projs, p)
	}
	sort.Strings(projs)
	w("applies_to:")
	w("  stack: ", qList(L.Stacks))
	w("  project: ", qList(projs))
	w("  files: []")
	w("status: ", StatusOf(L))
	enforce := "reminder"
	if L.Kind == "cause" && GuardCauses[L.Key] != "" {
		enforce = "hook"
	}
	w("enforce: ", enforce)
	if enforce == "hook" {
		w("guard: ", GuardCauses[L.Key])
	}
	w("evidence:")
	if len(L.Examples) == 0 {
		w("  []")
	}
	for _, e := range L.Examples {
		w("  - project: ", q(e.Project))
		w("    date: ", day(e.Ts))
		sess := e.Session
		if e.Agent != "" {
			sess = e.Agent + ":" + sess
		}
		w("    session: ", q(sess))
		if e.Pointer != "" {
			w("    pointer: ", q(e.Pointer))
		}
		if e.Error != "" {
			w("    error: ", q(e.Error))
		}
		if e.Fix != "" {
			w("    fix: ", q(e.Fix))
		}
	}
	w("score:")
	w("  shown: ", strconv.Itoa(sc.Shown))
	w("  repeats_after: ", strconv.Itoa(sc.RepeatsAfter))
	w("  last_seen: ", sc.LastSeen)
	w("  sessions: ", strconv.Itoa(L.Sessions))
	w("  episodes: ", strconv.Itoa(L.Episodes))
	w("---")
	w("")
	w(L.Text)
	return b.String()
}
