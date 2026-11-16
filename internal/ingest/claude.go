package ingest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

var exitRe = regexp.MustCompile(`^Exit code (\d+)`)

type clLine struct {
	Type        any `json:"type"`
	SessionID   any `json:"sessionId"`
	Cwd         any `json:"cwd"`
	Timestamp   any `json:"timestamp"`
	IsSidechain any `json:"isSidechain"`
	IsMeta      any `json:"isMeta"`
	Origin      any `json:"origin"`
	Message     any `json:"message"`
	Attachment  any `json:"attachment"`
}

type pendingCall struct {
	b    base
	name any
	cmd  string
	ok   bool // cmd is not None
}

var (
	userTag = []byte(`"type":"user"`)
	asstTag = []byte(`"type":"assistant"`)
	hookTag = []byte(`{"type":"hook_`)
)

// parseClaude: rel is the path shown in pointers (~/.claude/projects/...).
func parseClaude(path, rel string) ([]Row, error) {
	sideFile := strings.Contains(path, "/subagents/")
	pending := map[string]pendingCall{}
	var order []string // insertion order of pending ids (Python dict order)
	var rows []Row
	err := eachLine(path, nil, func(ln int, line []byte) {
		// Fast skip: a user/assistant line always contains its compact type tag.
		isHook := bytes.Contains(line, hookTag)
		if !isHook && !bytes.Contains(line, userTag) && !bytes.Contains(line, asstTag) {
			return
		}
		var o clLine
		if json.Unmarshal(line, &o) != nil {
			return
		}
		t := str(o.Type)
		if t != "user" && t != "assistant" && t != "attachment" {
			return
		}
		b := base{Source: "claude_transcript", Agent: "claude", SessionID: str(o.SessionID), Cwd: str(o.Cwd),
			Ts: str(o.Timestamp), RawFile: rel, RawLine: nint(int64(ln))}
		if truthy(o.IsSidechain) || sideFile {
			b.Side = 1
		}
		if t == "attachment" {
			if r, ok := hookFailure(b, o.Attachment); ok {
				rows = append(rows, r)
			}
			return
		}
		c := get(o.Message, "content")
		if t == "assistant" {
			list, ok := c.([]any)
			if !ok {
				return
			}
			for _, bl := range list {
				m, ok := bl.(map[string]any)
				if !ok {
					continue
				}
				switch str(m["type"]) {
				case "text":
					if txt := str(m["text"]); rules.Strip(txt) != "" {
						rows = append(rows, asstMsg(b, txt))
					}
				case "tool_use":
					inp := or(m["input"], map[string]any{})
					txt := or(get(inp, "command"), get(inp, "file_path"), get(inp, "query"), get(inp, "url"), inp)
					cmd, ok := tr(txt, maxCmd)
					id := key(m["id"])
					if _, seen := pending[id]; !seen {
						order = append(order, id)
					}
					pending[id] = pendingCall{b: b, name: m["name"], cmd: cmd, ok: ok}
				}
			}
			return
		}
		if list, ok := c.([]any); ok && hasToolResult(list) {
			for _, bl := range list {
				m, ok := bl.(map[string]any)
				if !ok || str(m["type"]) != "tool_result" {
					continue
				}
				id := key(m["tool_use_id"])
				pc, found := pending[id]
				if found {
					delete(pending, id)
				} else {
					pc = pendingCall{b: b}
				}
				out := blockText(m["content"])
				isErr := truthy(m["is_error"])
				name := str(pc.name)
				if !isErr && name == "Bash" && rules.HiddenFailure(pc.cmd, out) {
					isErr = true
				}
				var ec NInt
				if mm := exitRe.FindStringSubmatch(out); mm != nil {
					n, _ := strconv.ParseInt(mm[1], 10, 64)
					ec = nint(n)
				} else if name == "Bash" && !isErr {
					ec = nint(0)
				}
				cmdForClass := ""
				if name == "Bash" {
					cmdForClass = pc.cmd
				}
				r := pc.b.row(rules.EventTypeForCall(cmdForClass, isErr))
				r.Tool = name
				r = withText(r, pc.cmd, pc.ok)
				lim := okOut
				if isErr {
					lim = errOut
				}
				r = withOutput(r, trs(out, lim), true)
				r.ExitCode = ec
				r.IsError = boolInt(isErr)
				rows = append(rows, r.finish())
			}
			return
		}
		human := str(get(o.Origin, "kind")) == "human" && !truthy(o.IsMeta)
		if !human {
			return
		}
		txt := blockText(c)
		if rules.Strip(txt) != "" {
			rows = append(rows, userMsg(b, txt)...)
		}
	})
	for _, id := range order {
		pc, ok := pending[id]
		if !ok {
			continue
		}
		delete(pending, id)
		r := pc.b.row("tool_call")
		r.Tool = str(pc.name)
		rows = append(rows, withText(r, pc.cmd, pc.ok).finish())
	}
	return rows, err
}

// hookFailure turns a failed hook attachment into a hook_error row. Claude Code writes a timed-out hook as
// {"type":"hook_cancelled","timedOut":true,...}. is_error stays NULL: a hook that fails is a setup problem,
// not a mistake of the agent, so the report and the lessons do not count it.
func hookFailure(b base, a any) (Row, bool) {
	typ := str(get(a, "type"))
	var msg string
	switch {
	case typ == "hook_cancelled" && truthy(get(a, "timedOut")):
		msg = fmt.Sprintf("timed out after %s (ran %s)", secs(get(a, "timeoutMs")), secs(get(a, "durationMs")))
	case typ == "hook_cancelled":
		msg = "cancelled"
	case strings.HasPrefix(typ, "hook_") && strings.Contains(typ, "error"):
		msg = strings.TrimSpace(str(or(get(a, "stderr"), get(a, "content"), get(a, "error"))))
		if msg == "" {
			msg = strings.TrimPrefix(typ, "hook_")
		}
	default:
		return Row{}, false
	}
	r := b.row("hook_error")
	r.Tool = str(or(get(a, "hookName"), get(a, "hookEvent")))
	r = withText(r, trs(str(get(a, "command")), maxCmd), true)
	r = withOutput(r, trs(msg, errOut), true)
	return r.finish(), true
}

func secs(v any) string {
	ms, ok := v.(float64)
	if !ok {
		return "?"
	}
	return strconv.FormatFloat(ms/1000, 'f', -1, 64) + "s"
}

func hasToolResult(list []any) bool {
	for _, bl := range list {
		if str(get(bl, "type")) == "tool_result" {
			return true
		}
	}
	return false
}

// key turns a JSON id into a map key (Python dict keys can be None or any scalar).
func key(v any) string {
	if s, ok := v.(string); ok {
		return "s" + s
	}
	return "x" + pyStr(v)
}
