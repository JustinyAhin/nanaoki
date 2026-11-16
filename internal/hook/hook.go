// Package hook answers Claude Code and Codex hooks. It fails open: on any error it prints nothing and logs the error.
package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/activity"
	"github.com/JustinyAhin/nanaoki/internal/guards"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Input is the hook JSON on stdin (the fields both agents send that the handler reads).
type Input struct {
	SessionID     string          `json:"session_id"`
	Cwd           string          `json:"cwd"`
	HookEventName string          `json:"hook_event_name"`
	Source        string          `json:"source"`
	Prompt        string          `json:"prompt"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolResponse  json.RawMessage `json:"tool_response"`
	Error         string          `json:"error"`
}

const (
	MaxFailureLessons = 3
	MaxStartLessons   = lessons.StartK
)

// Deps are the parts a test replaces.
type Deps struct {
	Index    func() (*lessons.Index, error)
	StateDir string
	ShownLog string
	GuardLog string
	Env      func(cwd string) *guards.Env // nil: the real environment
	Now      func() time.Time
}

func DefaultDeps() Deps {
	return Deps{
		Index:    func() (*lessons.Index, error) { return lessons.LoadIndex(lessons.IndexPath()) },
		StateDir: filepath.Join(lessons.DefaultDir(), "hook-state"),
		ShownLog: lessons.ShownLogPath(),
		GuardLog: activity.GuardLogPath(),
		Now:      time.Now,
	}
}

func RealEnv(cwd string, ix *lessons.Index) *guards.Env {
	sh := os.Getenv("NANAOKI_SHELL")
	if sh == "" {
		sh = os.Getenv("SHELL")
	}
	e := &guards.Env{Cwd: cwd, Shell: sh}
	if ix != nil {
		e.SSHUsers = ix.SSHUsers
	}
	// Off by default: the hook sees the PATH of the agent process, not the PATH of the shell that runs the
	// command (nvm and version-manager shims set it per shell and per folder), so it warned on correct Node.
	if os.Getenv("NANAOKI_NODE_GUARD") == "1" {
		e.NodeMajor = func() (int, bool) { return cachedNodeMajor() }
	}
	return e
}

// Event names: the CLI argument (kebab case) and the hook_event_name the agents send.
var eventNames = map[string]string{
	"session-start": "SessionStart", "user-prompt-submit": "UserPromptSubmit", "pre-tool-use": "PreToolUse",
	"post-tool-use": "PostToolUse", "post-tool-use-failure": "PostToolUseFailure",
}

func EventName(s string) string {
	if n, ok := eventNames[s]; ok {
		return n
	}
	for _, n := range eventNames {
		if strings.EqualFold(n, s) {
			return n
		}
	}
	return ""
}

// Output is the JSON reply. Nil means "print nothing".
type Output map[string]any

func context(event, text string) Output {
	return Output{"hookSpecificOutput": map[string]any{"hookEventName": event, "additionalContext": text}}
}

func Handle(agent, event string, in Input, d Deps) (Output, error) {
	if event == "" {
		event = EventName(in.HookEventName)
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	start := d.Now()
	st := loadState(d.StateDir, agent, in.SessionID)
	defer st.save()
	switch event {
	case "SessionStart":
		// No user text yet: the strict rule needs the first message, so it runs at the first UserPromptSubmit.
		st.Started = true
		return nil, nil
	case "UserPromptSubmit":
		st.Prompts++
		if st.Prompts != 1 {
			return nil, nil
		}
		ix, err := d.Index()
		if err != nil {
			return nil, err
		}
		tr := lessons.Trigger{Kind: "session_start", Project: rules.Project(in.Cwd), Text: in.Prompt}
		hits := ix.Store().RetrieveStartStrict(tr, ix.RepoStacksFn(), MaxStartLessons)
		hits = st.fresh(hits)
		if len(hits) == 0 {
			return nil, nil
		}
		logShown(d, start, agent, in, event, hits)
		return context(event, "Lessons from your earlier sessions (nanaoki):\n"+lessons.Render(hits)), nil
	case "PreToolUse":
		cmd := commandOf(in.ToolInput)
		if cmd == "" || !shellTool(in.ToolName) {
			return nil, nil
		}
		var ix *lessons.Index
		if x, err := d.Index(); err == nil {
			ix = x
		}
		var e *guards.Env
		if d.Env != nil {
			e = d.Env(in.Cwd)
		} else {
			e = RealEnv(in.Cwd, ix)
		}
		e.FormatRan = st.FormatRan
		if sh := shellOf(in.ToolInput); sh != "" {
			e.Shell = sh // Codex can run a command in bash even when the login shell is zsh
		}
		fs := guards.Check(cmd, e)
		if guards.FormatRx.MatchString(cmd) {
			st.FormatRan = true
		}
		if len(fs) == 0 {
			return nil, nil
		}
		logGuards(d, start, agent, in, cmd, fs)
		var msgs []string
		block := false
		for _, f := range fs {
			m := "[" + f.Guard + "] " + f.Message
			if f.Fixed != "" {
				m += " Suggested command: `" + f.Fixed + "`"
			}
			msgs = append(msgs, m)
			if f.Decision == guards.Block {
				block = true
			}
		}
		text := "nanaoki guard: " + strings.Join(msgs, "\n")
		if block {
			return Output{"hookSpecificOutput": map[string]any{"hookEventName": event, "permissionDecision": "deny",
				"permissionDecisionReason": text}}, nil
		}
		return context(event, text), nil
	case "PostToolUseFailure", "PostToolUse":
		cmd := commandOf(in.ToolInput)
		out := in.Error
		if event == "PostToolUse" {
			var failed bool
			out, failed = failureOf(in.ToolResponse)
			if !failed {
				return nil, nil
			}
		}
		tool := in.ToolName
		if shellTool(tool) {
			tool = "Bash"
			// Stay quiet on probes: the non-zero exit was the answer the agent asked for.
			if rules.IsProbe(cmd, out) {
				return nil, nil
			}
		}
		ix, err := d.Index()
		if err != nil {
			return nil, err
		}
		proj := rules.Project(in.Cwd)
		tr := lessons.Trigger{Kind: "error", Project: proj, Cause: rules.CauseOf(cmd, out), Sig: rules.Signature(out),
			Head: rules.Head(tool, cmd)}
		hits := st.fresh(ix.Store().Retrieve(tr, MaxFailureLessons))
		if len(hits) == 0 {
			return nil, nil
		}
		logShown(d, start, agent, in, event, hits)
		return context(event, "This error matches lessons from your earlier sessions (nanaoki):\n"+lessons.Render(hits)), nil
	}
	return nil, nil
}

func shellTool(name string) bool {
	switch name {
	case "Bash", "bash", "shell", "Shell", "exec_command", "local_shell", "":
		return true
	}
	return false
}

// shellOf returns the shell of an argv command (["bash", "-lc", "cmd"] -> "bash"), else "".
func shellOf(raw json.RawMessage) string {
	var ti struct {
		Command []string `json:"command"`
	}
	if json.Unmarshal(raw, &ti) != nil || len(ti.Command) < 3 || (ti.Command[1] != "-lc" && ti.Command[1] != "-c") {
		return ""
	}
	return ti.Command[0]
}

// commandOf reads tool_input.command: a string, or an argv list (["bash", "-lc", "cmd"] -> "cmd").
func commandOf(raw json.RawMessage) string {
	var ti struct {
		Command json.RawMessage `json:"command"`
		Cmd     json.RawMessage `json:"cmd"`
	}
	if json.Unmarshal(raw, &ti) != nil {
		return ""
	}
	c := ti.Command
	if len(c) == 0 {
		c = ti.Cmd
	}
	var s string
	if json.Unmarshal(c, &s) == nil {
		return s
	}
	var argv []string
	if json.Unmarshal(c, &argv) == nil && len(argv) > 0 {
		if len(argv) >= 3 && (argv[1] == "-lc" || argv[1] == "-c") {
			return argv[2]
		}
		return strings.Join(argv, " ")
	}
	return ""
}

var exitTextRx = regexp.MustCompile(`(?i)(?:exit(?:ed)? (?:code|status)|process exited with code)[:= ]+(-?\d+)`)

// failureOf finds a failure in a tool response with no failure event (Codex PostToolUse): a non-zero exit
// code field, or an "Exit code: N" / "Process exited with code N" line. It returns the output text.
func failureOf(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return "", false
	}
	code, hasCode := -1, false
	var texts []string
	var walk func(x any, depth int)
	walk = func(x any, depth int) {
		if depth > 4 {
			return
		}
		switch t := x.(type) {
		case map[string]any:
			for k, val := range t {
				lk := strings.ToLower(k)
				if lk == "exit_code" || lk == "exitcode" || lk == "returncode" {
					if f, ok := val.(float64); ok {
						code, hasCode = int(f), true
					}
				}
				walk(val, depth+1)
			}
		case []any:
			for _, val := range t {
				walk(val, depth+1)
			}
		case string:
			texts = append(texts, t)
		}
	}
	walk(v, 0)
	text := strings.Join(texts, "\n")
	if !hasCode {
		if m := exitTextRx.FindStringSubmatch(text); m != nil {
			code, _ = strconv.Atoi(m[1])
			hasCode = true
		}
	}
	return text, hasCode && code != 0
}

func msSince(d Deps, start time.Time) float64 {
	return float64(d.Now().Sub(start).Microseconds()) / 1000
}

func logShown(d Deps, start time.Time, agent string, in Input, event string, hits []lessons.Hit) {
	var es []lessons.ShownEntry
	now := d.Now().UTC().Format(time.RFC3339)
	ms := msSince(d, start)
	for _, h := range hits {
		es = append(es, lessons.ShownEntry{Ts: now, Lesson: h.Lesson.ID, Event: event, Agent: agent, Session: in.SessionID,
			Project: rules.Project(in.Cwd), Ms: ms})
	}
	_ = lessons.AppendShown(d.ShownLog, es)
}

func logGuards(d Deps, start time.Time, agent string, in Input, cmd string, fs []guards.Finding) {
	if d.GuardLog == "" {
		return
	}
	now := d.Now().UTC().Format(time.RFC3339)
	ms := msSince(d, start)
	c := activity.Cmd(cmd)
	es := make([]activity.GuardEntry, 0, len(fs))
	for _, f := range fs {
		e := activity.GuardEntry{Ts: now, Agent: agent, Session: in.SessionID, Project: rules.Project(in.Cwd), Guard: f.Guard,
			Cause: f.Cause, Decision: f.Decision, Cmd: c, Ms: ms}
		if f.Fixed != "" {
			e.Fixed = activity.Cmd(f.Fixed)
		}
		es = append(es, e)
	}
	_ = activity.Append(d.GuardLog, es...)
}

type state struct {
	path      string
	Started   bool            `json:"started"`
	Prompts   int             `json:"prompts"`
	FormatRan bool            `json:"format_ran"`
	Shown     map[string]bool `json:"shown"`
	dirty     bool
}

var safeID = regexp.MustCompile(`[^A-Za-z0-9_.-]`)

func loadState(dir, agent, sid string) *state {
	st := &state{Shown: map[string]bool{}}
	if dir == "" || sid == "" {
		return st
	}
	st.path = filepath.Join(dir, agent+"-"+safeID.ReplaceAllString(sid, "_")+".json")
	if b, err := os.ReadFile(st.path); err == nil {
		_ = json.Unmarshal(b, st)
		if st.Shown == nil {
			st.Shown = map[string]bool{}
		}
	}
	return st
}

// fresh drops lessons already shown in this session (one lesson is shown once per session).
func (st *state) fresh(hits []lessons.Hit) []lessons.Hit {
	var out []lessons.Hit
	for _, h := range hits {
		if !st.Shown[h.Lesson.ID] {
			st.Shown[h.Lesson.ID] = true
			out = append(out, h)
		}
	}
	return out
}

func (st *state) save() {
	if st.path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(st.path), 0o755); err != nil {
		return
	}
	b, _ := json.Marshal(st)
	_ = os.WriteFile(st.path, b, 0o644)
}

func cachedNodeMajor() (int, bool) {
	cache := filepath.Join(lessons.DefaultDir(), "node-version.json")
	p := os.Getenv("PATH")
	var c struct {
		V     int    `json:"v"`
		Path  string `json:"path"`
		Major int    `json:"major"`
		OK    bool   `json:"ok"`
	}
	if b, err := os.ReadFile(cache); err == nil && json.Unmarshal(b, &c) == nil && c.V == nodeCacheV && c.Path == p {
		return c.Major, c.OK
	}
	n, ok := guards.NodeMajorFromPath(lookPath)
	c.V, c.Path, c.Major, c.OK = nodeCacheV, p, n, ok
	if b, err := json.Marshal(c); err == nil {
		_ = os.WriteFile(cache, b, 0o644)
	}
	return n, ok
}

// nodeCacheV changes when NodeMajorFromPath changes, so that an old cached answer is not used.
const nodeCacheV = 2

var lookPath = exec.LookPath

func LogError(event string, err any) {
	p := filepath.Join(lessons.DefaultDir(), "hook.log")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	f, e := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if e != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s %v\n", time.Now().UTC().Format(time.RFC3339), event, err)
}
