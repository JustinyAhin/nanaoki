package ingest

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

var (
	uuidRe     = regexp.MustCompile(`([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
	procExitRe = regexp.MustCompile(`Process exited with code (-?\d+)`)
	jsonExitRe = regexp.MustCompile(`"exit_code"\s*:\s*(-?\d+)`)
	jsCmdRe    = regexp.MustCompile("\\bcmd\\s*:\\s*(\"(?:[^\"\\\\]|\\\\.)*\"|`(?:[^`\\\\]|\\\\.)*`|'(?:[^'\\\\]|\\\\.)*')")
	evKeep     = map[string]bool{"exec_command_end": true, "mcp_tool_call_end": true, "patch_apply_end": true, "user_message": true, "agent_message": true, "context_compacted": true}
	callTypes  = map[string]bool{"function_call": true, "custom_tool_call": true, "local_shell_call": true}
	outTypes   = map[string]bool{"function_call_output": true, "custom_tool_call_output": true, "local_shell_call_output": true}
	shellNames = map[string]bool{"exec_command": true, "shell": true, "shell_command": true, "local_shell_call": true, "container.exec": true}
)

func ThreadIDFromName(path string) string {
	if m := uuidRe.FindStringSubmatch(filepath.Base(path)); m != nil {
		return m[1]
	}
	return ""
}

type cxLine struct {
	Timestamp any `json:"timestamp"`
	Type      any `json:"type"`
	Payload   any `json:"payload"`
}

type bp struct {
	b base
	p map[string]any
}

// codexSkip is a fast pre-filter on the usual line layout {"timestamp":"..","type":"X","payload":{"type":"Y"...
// It returns true only for lines the reference ignores anyway. Unknown layouts are parsed in full.
func codexSkip(line []byte) bool {
	const pre = `{"timestamp":"`
	if !bytes.HasPrefix(line, []byte(pre)) {
		return false
	}
	rest := line[len(pre):]
	i := bytes.IndexByte(rest, '"')
	if i < 0 {
		return false
	}
	rest = rest[i+1:]
	const tp = `,"type":"`
	if !bytes.HasPrefix(rest, []byte(tp)) {
		return false
	}
	rest = rest[len(tp):]
	i = bytes.IndexByte(rest, '"')
	if i < 0 {
		return false
	}
	t := string(rest[:i])
	rest = rest[i+1:]
	switch t {
	case "session_meta", "turn_context", "compacted":
		return false
	case "event_msg", "response_item":
	default:
		return true
	}
	const pp = `,"payload":{"type":"`
	if !bytes.HasPrefix(rest, []byte(pp)) {
		return false
	}
	rest = rest[len(pp):]
	i = bytes.IndexByte(rest, '"')
	if i < 0 {
		return false
	}
	pt := string(rest[:i])
	if t == "event_msg" {
		return pt != "item_completed" && !evKeep[pt]
	}
	return !callTypes[pt] && !outTypes[pt]
}

// parseCodex: rel is the path shown in pointers (~/.codex/...).
func parseCodex(path, rel string, dec *zstd.Decoder) ([]Row, error) {
	sid := ThreadIDFromName(path)
	var cwd string
	cwdSet := false
	side := 0
	var ic, ev []bp // ic: (base, item) of item_completed; ev: (base, payload)
	calls := map[string]bp{}
	var callOrder []string
	outs := map[string]map[string]any{}

	err := eachLine(path, dec, func(ln int, line []byte) {
		if codexSkip(line) {
			return
		}
		var o cxLine
		if json.Unmarshal(line, &o) != nil {
			return
		}
		t := str(o.Type)
		p, _ := o.Payload.(map[string]any)
		if p == nil {
			p = map[string]any{}
		}
		switch t {
		case "session_meta":
			if !cwdSet {
				if s := str(p["id"]); s != "" {
					sid = s
				}
				if v := p["cwd"]; v != nil {
					cwd, cwdSet = str(v), true
				}
				ts := p["thread_source"]
				side = 0
				if str(p["originator"]) == "codex_exec" || (ts != nil && str(ts) != "user") || (ts != nil && !isString(ts)) || truthy(p["parent_thread_id"]) {
					side = 1
				}
			}
			return
		case "turn_context":
			if truthy(p["cwd"]) {
				cwd, cwdSet = str(p["cwd"]), true
			}
			return
		}
		b := base{Source: "codex_rollout", Agent: "codex", SessionID: sid, Cwd: cwd, Ts: str(o.Timestamp), Side: side,
			RawFile: rel, RawLine: nint(int64(ln))}
		switch t {
		case "event_msg":
			pt := str(p["type"])
			if pt == "item_completed" {
				it, _ := p["item"].(map[string]any)
				if it == nil {
					it = map[string]any{}
				}
				ic = append(ic, bp{b, it})
			} else if evKeep[pt] {
				ev = append(ev, bp{b, p})
			}
		case "response_item":
			pt := str(p["type"])
			if callTypes[pt] {
				id := key(p["call_id"])
				if _, ok := calls[id]; !ok {
					callOrder = append(callOrder, id)
				}
				calls[id] = bp{b, p}
			} else if outTypes[pt] {
				outs[key(p["call_id"])] = p
			}
		case "compacted":
			ev = append(ev, bp{b, map[string]any{"type": "context_compacted"}})
		}
	})

	var rows []Row
	icTypes, evTypes := map[string]bool{}, map[string]bool{}
	for _, x := range ic {
		icTypes[str(x.p["type"])] = true
	}
	for _, x := range ev {
		evTypes[str(x.p["type"])] = true
	}

	if icTypes["UserMessage"] || icTypes["AgentMessage"] {
		for _, x := range ic {
			it := str(x.p["type"])
			if it != "UserMessage" && it != "AgentMessage" {
				continue
			}
			var parts []string
			if cl, ok := x.p["content"].([]any); ok {
				for _, c := range cl {
					if m, ok := c.(map[string]any); ok {
						parts = append(parts, str(m["text"]))
					}
				}
			}
			txt := strings.Join(parts, "\n")
			if rules.Strip(txt) == "" {
				continue
			}
			if it == "UserMessage" {
				rows = append(rows, userMsg(x.b, txt)...)
			} else {
				rows = append(rows, asstMsg(x.b, txt))
			}
		}
	} else {
		for _, x := range ev {
			msg := str(x.p["message"])
			switch str(x.p["type"]) {
			case "user_message":
				if rules.Strip(msg) != "" {
					rows = append(rows, userMsg(x.b, msg)...)
				}
			case "agent_message":
				if rules.Strip(msg) != "" {
					rows = append(rows, asstMsg(x.b, msg))
				}
			}
		}
	}
	for _, x := range ev {
		if str(x.p["type"]) == "context_compacted" {
			rows = append(rows, withText(x.b.row("compaction"), "", true).finish())
		}
	}
	for _, x := range ic {
		if str(x.p["type"]) == "ContextCompaction" {
			rows = append(rows, withText(x.b.row("compaction"), "", true).finish())
		}
	}

	switch {
	case icTypes["CommandExecution"]:
		for _, x := range ic {
			if str(x.p["type"]) == "CommandExecution" {
				rows = append(rows, cmdRow(x.b, x.p["command"], x.p["exit_code"], str(x.p["status"]), x.p["aggregated_output"], str(x.p["cwd"])))
			}
		}
	case evTypes["exec_command_end"]:
		for _, x := range ev {
			if str(x.p["type"]) == "exec_command_end" {
				outp := x.p["aggregated_output"]
				if !truthy(outp) {
					outp = str(x.p["stdout"]) + str(x.p["stderr"])
				}
				rows = append(rows, cmdRow(x.b, x.p["command"], x.p["exit_code"], str(x.p["status"]), outp, str(x.p["cwd"])))
			}
		}
	default:
		for _, id := range callOrder {
			x := calls[id]
			p := x.p
			name := str(or(p["name"], p["type"]))
			op := outs[id]
			otext, mec := outText(get(op, "output"))
			if shellNames[name] {
				var a map[string]any
				if s, ok := p["arguments"].(string); ok && s != "" {
					var v any
					if json.Unmarshal([]byte(s), &v) == nil {
						a, _ = v.(map[string]any)
					}
				}
				cmd := or(a["cmd"], a["command"], get(p["action"], "command"))
				var ec any = mec
				if mec == nil {
					if m := procExitRe.FindStringSubmatch(otext); m != nil {
						n, _ := strconv.ParseFloat(m[1], 64)
						ec = n
					}
				}
				rows = append(rows, cmdRow(x.b, cmd, ec, "", otext, str(a["workdir"])))
			} else if name == "exec" && str(p["type"]) == "custom_tool_call" {
				cmds := jsCmds(str(p["input"]))
				if len(cmds) == 0 {
					continue
				}
				var codes []int64
				for _, m := range jsonExitRe.FindAllStringSubmatch(otext, -1) {
					n, _ := strconv.ParseInt(m[1], 10, 64)
					codes = append(codes, n)
				}
				failed := strings.HasPrefix(otext, "Script failed")
				var ec any
				for _, c := range codes {
					if c != 0 {
						ec = float64(c)
						break
					}
				}
				if ec == nil {
					if len(codes) > 0 {
						ec = float64(codes[0])
					} else if failed {
						ec = float64(1)
					}
				}
				status := ""
				if failed {
					status = "failed"
				}
				rows = append(rows, cmdRow(x.b, strings.Join(cmds, " && "), ec, status, otext, ""))
			}
		}
	}

	if icTypes["McpToolCall"] {
		for _, x := range ic {
			if str(x.p["type"]) == "McpToolCall" {
				rows = append(rows, mcpRow(x.b, x.p["server"], x.p["tool"], str(x.p["status"]), x.p["result"], x.p["error"]))
			}
		}
	} else {
		for _, x := range ev {
			if str(x.p["type"]) != "mcp_tool_call_end" {
				continue
			}
			inv := x.p["invocation"]
			res := or(x.p["result"], map[string]any{})
			okv, errv := get(res, "Ok"), get(res, "Err")
			status := "completed"
			if truthy(errv) {
				status = "failed"
			}
			rows = append(rows, mcpRow(x.b, get(inv, "server"), get(inv, "tool"), status, or(okv, map[string]any{}), errv))
		}
	}

	if icTypes["FileChange"] {
		for _, x := range ic {
			if str(x.p["type"]) == "FileChange" {
				r := x.b.row("file_change")
				r.Tool = "FileChange"
				rows = append(rows, withText(r, trs(keysOf(x.p["changes"]), 800), true).finish())
			}
		}
	} else {
		for _, x := range ev {
			if str(x.p["type"]) != "patch_apply_end" {
				continue
			}
			okv, has := x.p["success"]
			ok := !has || truthy(okv)
			et := "file_change"
			if !ok {
				et = "tool_error"
			}
			r := x.b.row(et)
			r.Tool = "apply_patch"
			r = withText(r, trs(keysOf(x.p["changes"]), 800), true)
			if !ok {
				o, set := tr(or(x.p["stderr"], x.p["stdout"]), errOut)
				r = withOutput(r, o, set)
			}
			r.IsError = boolInt(!ok)
			rows = append(rows, r.finish())
		}
	}
	return rows, err
}

func isString(v any) bool { _, ok := v.(string); return ok }

// keysOf is list(d.keys()) for a JSON object (Python keeps insertion order; encoding/json sorts).
func keysOf(v any) []string {
	m, _ := v.(map[string]any)
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func cmdRow(b base, cmd any, ecv any, status string, outp any, cwd string) Row {
	if l, ok := cmd.([]any); ok {
		if len(l) > 0 {
			cmd = l[len(l)-1]
		} else {
			cmd = ""
		}
	}
	ec, ecOK := toInt(ecv)
	isErr := (ecOK && ec != 0) || status == "failed" || status == "declined"
	if cwd != "" {
		b.Cwd = strings.ReplaceAll(cwd, "file://", "")
	}
	cmdText, cmdSet := tr(cmd, maxCmd)
	classIn := ""
	if s, ok := cmd.(string); ok {
		classIn = s
	} else if cmdSet {
		classIn = cmdText
	}
	if !isErr && rules.HiddenFailure(classIn, pyStr(or(outp, ""))) {
		isErr = true
	}
	r := b.row(rules.EventTypeForCall(classIn, isErr))
	r.Tool = "shell"
	r = withText(r, cmdText, cmdSet)
	lim := okOut
	if isErr {
		lim = errOut
	}
	if !truthy(outp) {
		outp = ""
	}
	r = withOutput(r, trs(outp, lim), true)
	if ecOK {
		r.ExitCode = nint(ec)
	}
	r.IsError = boolInt(isErr)
	return r.finish()
}

func mcpRow(b base, server, tool any, status string, res any, errField any) Row {
	var txt string
	if m, ok := res.(map[string]any); ok {
		txt = blockText(m["content"])
	} else {
		txt = pyStr(or(res, ""))
	}
	isErr := status == "failed" || truthy(errField)
	if m, ok := res.(map[string]any); ok && truthy(m["isError"]) {
		isErr = true
	}
	et := "tool_call"
	if isErr {
		et = "tool_error"
	}
	r := b.row(et)
	r.Tool = "mcp:" + pyStr(server) + ":" + pyStr(tool)
	lim := okOut
	if isErr {
		lim = errOut
	}
	o := txt
	if o == "" {
		o = pyStr(or(errField, ""))
	}
	r = withOutput(r, trs(o, lim), true)
	r.IsError = boolInt(isErr)
	return r.finish()
}

// outText reads a JSON string {"output": .., "metadata": {"exit_code": ..}} or plain text.
func outText(o any) (string, any) {
	if s, ok := o.(string); ok {
		var j any
		if json.Unmarshal([]byte(s), &j) == nil {
			if m, ok := j.(map[string]any); ok {
				if _, has := m["output"]; has {
					out := m["output"]
					if !truthy(out) {
						out = ""
					}
					o2, isStr := out.(string)
					if !isStr {
						o2 = blockText(out)
					}
					return o2, get(m["metadata"], "exit_code")
				}
			}
		}
		return s, nil
	}
	return blockText(o), nil
}

func jsCmds(src string) []string {
	var out []string
	for _, m := range jsCmdRe.FindAllStringSubmatch(src, -1) {
		s := m[1]
		if s[0] == '"' {
			var v string
			if json.Unmarshal([]byte(s), &v) == nil {
				out = append(out, v)
				continue
			}
		}
		out = append(out, s[1:len(s)-1])
	}
	return out
}
