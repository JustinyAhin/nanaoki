package ingest

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// codexSQLiteRows reads the threads in thread_history_1.sqlite that have no rollout file.
func codexSQLiteRows(p Paths, known map[string]bool, emit func(Row) error) (threads, missing int, err error) {
	path := filepath.Join(p.CodexDir, "thread_history_1.sqlite")
	if _, err := os.Stat(path); err != nil {
		return 0, 0, nil
	}
	con, err := sql.Open("sqlite", "file:"+path+"?mode=ro&immutable=1")
	if err != nil {
		return 0, 0, err
	}
	defer con.Close()
	rs, err := con.Query(`select distinct thread_id from thread_items`)
	if err != nil {
		return 0, 0, err
	}
	var miss []string
	for rs.Next() {
		var t string
		if rs.Scan(&t) == nil {
			threads++
			if !known[t] {
				miss = append(miss, t)
			}
		}
	}
	rs.Close()
	rel := p.Rel(path)
	for _, tid := range miss {
		rows, err := con.Query(`select item_id, created_at_ms, item_json, item_type from thread_items where thread_id=? order by rollout_ordinal`, tid)
		if err != nil {
			return threads, len(miss), err
		}
		cwd := ""
		type item struct {
			id, j, typ string
			ms         float64
		}
		var items []item
		for rows.Next() {
			var it item
			if rows.Scan(&it.id, &it.ms, &it.j, &it.typ) == nil {
				items = append(items, it)
			}
		}
		rows.Close()
		for _, it := range items {
			var d map[string]any
			if json.Unmarshal([]byte(it.j), &d) != nil {
				continue
			}
			b := base{Source: "codex_sqlite", Agent: "codex", SessionID: tid, Cwd: cwd, Ts: isoMs(it.ms),
				RawFile: rel + "#thread_items:" + tid + "/" + it.id}
			var out []Row
			switch it.typ {
			case "commandExecution":
				if c := str(d["cwd"]); c != "" {
					cwd = c
				}
				b.Cwd = cwd
				ec, ecOK := toInt(d["exitCode"])
				isErr := (ecOK && ec != 0) || str(d["status"]) == "failed"
				cmd := str(or(d["command"], ""))
				r := b.row(rules.EventTypeForCall(cmd, isErr))
				r.Tool = "shell"
				r = withText(r, trs(cmd, maxCmd), true)
				lim := okOut
				if isErr {
					lim = errOut
				}
				r = withOutput(r, trs(or(d["aggregatedOutput"], ""), lim), true)
				if ecOK {
					r.ExitCode = nint(ec)
				}
				r.IsError = boolInt(isErr)
				out = append(out, r.finish())
			case "userMessage":
				if txt := blockText(d["content"]); rules.Strip(txt) != "" {
					out = append(out, userMsg(b, txt)...)
				}
			case "agentMessage":
				if txt := str(d["text"]); rules.Strip(txt) != "" {
					out = append(out, asstMsg(b, txt))
				}
			case "mcpToolCall":
				isErr := str(d["status"]) == "failed" || truthy(d["error"])
				et := "tool_call"
				lim := okOut
				if isErr {
					et, lim = "tool_error", errOut
				}
				r := b.row(et)
				r.Tool = "mcp:" + pyStr(d["server"]) + ":" + pyStr(d["tool"])
				r = withOutput(r, trs(pyStr(or(d["error"], d["result"], "")), lim), true)
				r.IsError = boolInt(isErr)
				out = append(out, r.finish())
			case "contextCompaction":
				out = append(out, withText(b.row("compaction"), "", true).finish())
			}
			for _, r := range out {
				if err := emit(r); err != nil {
					return threads, len(miss), err
				}
			}
		}
	}
	return threads, len(miss), nil
}

// claudeHistoryRows reads ~/.claude/history.jsonl (prompts and pastes).
func claudeHistoryRows(p Paths, emit func(Row) error) error {
	path := filepath.Join(p.ClaudeDir, "history.jsonl")
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	rel := p.Rel(path)
	var ferr error
	err := eachLine(path, nil, func(ln int, line []byte) {
		if ferr != nil {
			return
		}
		var o map[string]any
		if json.Unmarshal(line, &o) != nil {
			return
		}
		txt := str(o["display"])
		if pc, ok := o["pastedContents"].(map[string]any); ok {
			for _, k := range numericKeys(pc) {
				if v, ok := pc[k].(map[string]any); ok && truthy(v["content"]) {
					txt += "\n<pasted_content>\n" + str(v["content"]) + "\n</pasted_content>"
				}
			}
		}
		if rules.Strip(txt) == "" {
			return
		}
		ms, _ := or(o["timestamp"], 0.0).(float64)
		b := base{Source: "claude_history", Agent: "claude", SessionID: str(o["sessionId"]), Cwd: str(o["project"]),
			Ts: isoMs(ms), RawFile: rel, RawLine: nint(int64(ln))}
		for _, r := range userMsg(b, txt) {
			if ferr = emit(r); ferr != nil {
				return
			}
		}
	})
	if ferr != nil {
		return ferr
	}
	return err
}

// codexHistoryRows reads ~/.codex/history.jsonl (prompts). sidCwd gives the cwd of a session.
func codexHistoryRows(p Paths, sidCwd map[string]string, emit func(Row) error) error {
	path := filepath.Join(p.CodexDir, "history.jsonl")
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	rel := p.Rel(path)
	var ferr error
	err := eachLine(path, nil, func(ln int, line []byte) {
		if ferr != nil {
			return
		}
		var o map[string]any
		if json.Unmarshal(line, &o) != nil {
			return
		}
		txt := str(o["text"])
		if rules.Strip(txt) == "" {
			return
		}
		sid := str(o["session_id"])
		sec, _ := or(o["ts"], 0.0).(float64)
		b := base{Source: "codex_history", Agent: "codex", SessionID: sid, Cwd: sidCwd[sid], Ts: isoMs(sec * 1000),
			RawFile: rel, RawLine: nint(int64(ln))}
		for _, r := range userMsg(b, txt) {
			if ferr = emit(r); ferr != nil {
				return
			}
		}
	})
	if ferr != nil {
		return ferr
	}
	return err
}

// numericKeys sorts keys as numbers when they are numbers (paste ids "1", "2", ...), else as strings.
func numericKeys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool {
		a, ea := strconv.Atoi(ks[i])
		b, eb := strconv.Atoi(ks[j])
		if ea == nil && eb == nil {
			return a < b
		}
		return ks[i] < ks[j]
	})
	return ks
}
