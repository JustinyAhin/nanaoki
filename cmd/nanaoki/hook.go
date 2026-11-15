package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/JustinyAhin/nanaoki/internal/hook"
)

// runHook answers one hook call: `nanaoki hook [--agent claude|codex] EVENT` with the hook JSON on stdin.
// It always exits 0. On any error or panic it prints nothing, so the agent goes on as without the hook (it
// never prints "allow": that would skip the user's own permission prompt), and logs to hook.log.
func runHook(args []string, stdin io.Reader, stdout io.Writer) (code int) {
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
	if err := json.Unmarshal(b, &in); err != nil {
		hook.LogError(event, fmt.Errorf("bad hook JSON: %w", err))
		return 0
	}
	name := hook.EventName(event)
	if event != "" && name == "" {
		hook.LogError(event, "unknown event")
		return 0
	}
	out, err := hook.Handle(agent, name, in, hook.DefaultDeps())
	if err != nil {
		hook.LogError(event, err)
		return 0
	}
	if out != nil {
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(out); err != nil {
			hook.LogError(event, err)
		}
	}
	return 0
}
