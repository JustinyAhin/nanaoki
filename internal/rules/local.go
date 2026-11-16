package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Local rules: personal causes and prompt rules that are not in the built-in tables, from
// ~/.local/share/nanaoki/local_rules.json (NANAOKI_LOCAL_RULES sets another path). Example:
//
//	{
//	  "causes": [
//	    {"id": "my_cli_name", "scope": "env", "lesson": "The CLI is 'my-cli', not 'my'.",
//	     "rx": "command not found: my\\b", "cmd_rx": "\\bmy\\b", "before": "git_remote_auth",
//	     "hook_fix": "PreToolUse hook on Bash: rewrite 'my ' calls."}
//	  ],
//	  "prompts": [
//	    {"key": "humanizer", "kind": "pref", "label": "Run the humanizer on copy", "rx": "humaniz",
//	     "before": "no_tests_unless_needed"}
//	  ]
//	}
//
// An entry with the id or key of a built-in rule replaces that rule in place. A new entry goes before the
// rule named by "before", or at the end. Patterns use Python `re` syntax, like the built-in tables.
// Causes match with the flags (?im), cmd_rx and prompt rules with (?i).
type localRules struct {
	Causes []struct {
		ID      string `json:"id"`
		Scope   string `json:"scope"`
		Lesson  string `json:"lesson"`
		Rx      string `json:"rx"`
		CmdRx   string `json:"cmd_rx"`
		Before  string `json:"before"`
		HookFix string `json:"hook_fix"`
	} `json:"causes"`
	Prompts []struct {
		Key    string `json:"key"`
		Kind   string `json:"kind"`
		Label  string `json:"label"`
		Rx     string `json:"rx"`
		Before string `json:"before"`
	} `json:"prompts"`
}

// LocalHookFix is the hook suggestion of each local cause that has one.
var LocalHookFix = map[string]string{}

// LocalRulesPath is the file that local rules come from ("" in tests, so that they do not read the user's file).
func LocalRulesPath() string {
	if p := os.Getenv("NANAOKI_LOCAL_RULES"); p != "" {
		return p
	}
	if testing.Testing() {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "nanaoki", "local_rules.json")
}

// readLocalRules returns nil when the file is missing. A bad file gives a warning on stderr and is ignored,
// so that a typo never stops a hook.
func readLocalRules(path string) *localRules {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lr localRules
	if err := json.Unmarshal(b, &lr); err != nil {
		fmt.Fprintf(os.Stderr, "nanaoki: %s ignored: %v\n", path, err)
		return nil
	}
	for _, c := range lr.Causes {
		if err := checkRx(c.Rx, "(?im)"); c.ID == "" || c.Rx == "" || err != nil {
			fmt.Fprintf(os.Stderr, "nanaoki: %s ignored: cause %q needs an id and a valid rx (%v)\n", path, c.ID, err)
			return nil
		}
		if err := checkRx(c.CmdRx, "(?i)"); err != nil {
			fmt.Fprintf(os.Stderr, "nanaoki: %s ignored: cause %q: bad cmd_rx (%v)\n", path, c.ID, err)
			return nil
		}
	}
	for _, p := range lr.Prompts {
		if err := checkRx(p.Rx, "(?i)"); p.Key == "" || p.Rx == "" || err != nil {
			fmt.Fprintf(os.Stderr, "nanaoki: %s ignored: prompt rule %q needs a key and a valid rx (%v)\n", path, p.Key, err)
			return nil
		}
	}
	return &lr
}

func checkRx(p, flags string) error {
	if p == "" {
		return nil
	}
	_, err := regexp.Compile(flags + pyToRE2(p))
	return err
}

// mergeCauses returns the built-in table with the local causes put in.
func mergeCauses(table []causeDef, lr *localRules) []causeDef {
	out := append([]causeDef(nil), table...)
	if lr == nil {
		return out
	}
	for _, c := range lr.Causes {
		d := causeDef{ID: c.ID, Scope: c.Scope, Lesson: c.Lesson, Rx: c.Rx, CmdRx: c.CmdRx}
		if d.Scope == "" {
			d.Scope = "env"
		}
		if c.HookFix != "" {
			LocalHookFix[c.ID] = c.HookFix
		}
		out = insertAt(out, d, func(x causeDef) string { return x.ID }, c.ID, c.Before)
	}
	return out
}

// mergePrompts returns the built-in prompt rules with the local rules put in.
func mergePrompts(table []Rule, lr *localRules) []Rule {
	out := append([]Rule(nil), table...)
	if lr == nil {
		return out
	}
	for _, p := range lr.Prompts {
		r := Rule{Key: p.Key, Kind: p.Kind, Label: p.Label, Rx: py(p.Rx, "(?i)")}
		if r.Kind == "" {
			r.Kind = "pref"
		}
		if r.Label == "" {
			r.Label = p.Key
		}
		out = insertAt(out, r, func(x Rule) string { return x.Key }, p.Key, p.Before)
	}
	return out
}

// insertAt replaces the item with the same id, else puts it before the item named before, else at the end.
func insertAt[T any](s []T, v T, id func(T) string, key, before string) []T {
	for i := range s {
		if id(s[i]) == key {
			s[i] = v
			return s
		}
	}
	for i := range s {
		if before != "" && id(s[i]) == before {
			return append(s[:i], append([]T{v}, s[i:]...)...)
		}
	}
	return append(s, v)
}
