package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/activity"
	"github.com/JustinyAhin/nanaoki/internal/hook"
	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// runHook answers one hook call: `nanaoki hook [--agent claude|codex] EVENT` with the hook JSON on stdin.
// It always exits 0. On any error or panic it prints nothing, so the agent goes on as without the hook (it
// never prints "allow": that would skip the user's own permission prompt), and logs to hook.log.
func runHook(args []string, stdin io.Reader, stdout io.Writer) (code int) {
	start := time.Now()
	agent, event := "claude", ""
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--agent" && i+1 < len(args):
			agent = args[i+1]
			i++
		case len(a) > 8 && a[:8] == "--agent=":
			agent = a[8:]
		default:
			event = a
		}
	}
	defer func() {
		if r := recover(); r != nil {
			hook.LogError(event, fmt.Sprintf("panic: %v", r))
			code = 0
		}
	}()
	b, err := io.ReadAll(io.LimitReader(stdin, 8<<20))
	if err != nil {
		hook.LogError(event, err)
		return 0
	}
	var in hook.Input
	call := activity.CallEntry{Agent: agent, Event: event}
	defer func() {
		call.Ts = start.UTC().Format(time.RFC3339)
		call.Session, call.Project = in.SessionID, rules.Project(in.Cwd)
		call.Ms = float64(time.Since(start).Microseconds()) / 1000
		_ = activity.Append(activity.CallLogPath(), call)
	}()
	if err := json.Unmarshal(b, &in); err != nil {
		call.Err = true
		hook.LogError(event, fmt.Errorf("bad hook JSON: %w", err))
		return 0
	}
	name := hook.EventName(event)
	if event != "" && name == "" {
		call.Err = true
		hook.LogError(event, "unknown event")
		return 0
	}
	if name == "SessionStart" {
		maybeStartRefresh(start)
	}
	out, err := hook.Handle(agent, name, in, hook.DefaultDeps())
	if err != nil {
		call.Err = true
		hook.LogError(event, err)
		return 0
	}
	if out != nil {
		call.Out = true
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(out); err != nil {
			hook.LogError(event, err)
		}
	}
	return 0
}
