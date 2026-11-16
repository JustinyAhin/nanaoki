package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local_rules.json")
	os.WriteFile(path, []byte(`{
	  "causes": [
	    {"id": "my_cli_name", "lesson": "The CLI is 'my-cli'.", "rx": "command not found: my\\b", "cmd_rx": "\\bmy\\b",
	     "before": "git_remote_auth", "hook_fix": "Rewrite 'my ' calls."},
	    {"id": "webfetch_reddit", "scope": "env", "lesson": "Local text.", "rx": "reddit\\.com.*403"}
	  ],
	  "prompts": [{"key": "my_pref", "label": "My preference", "rx": "my pref", "before": "pr_draft"}]
	}`), 0o644)
	lr := readLocalRules(path)
	if lr == nil {
		t.Fatal("file not read")
	}
	table := mergeCauses(causeTable, lr)
	if len(table) != len(causeTable)+1 {
		t.Fatalf("len %d, want %d", len(table), len(causeTable)+1)
	}
	idx := map[string]int{}
	for i, d := range table {
		idx[d.ID] = i
	}
	if idx["my_cli_name"]+1 != idx["git_remote_auth"] {
		t.Error("new cause not put before git_remote_auth")
	}
	if d := table[idx["webfetch_reddit"]]; d.Lesson != "Local text." || idx["webfetch_reddit"] != indexOf(causeTable, "webfetch_reddit") {
		t.Error("built-in cause not replaced in place")
	}
	if table[idx["my_cli_name"]].Scope != "env" || LocalHookFix["my_cli_name"] != "Rewrite 'my ' calls." {
		t.Error("default scope or hook fix")
	}
	prompts := mergePrompts(PromptRules, lr)
	for i, r := range prompts {
		if r.Key == "my_pref" {
			if prompts[i+1].Key != "pr_draft" || r.Kind != "pref" || !r.Rx.MatchString("My Pref here") {
				t.Error("prompt rule")
			}
			return
		}
	}
	t.Error("prompt rule not added")
}

func TestLocalRulesBadFile(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"syntax.json": `{"causes": [`,
		"rx.json":     `{"causes": [{"id": "x", "rx": "a(b"}]}`,
		"noid.json":   `{"prompts": [{"rx": "x"}]}`,
	} {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(body), 0o644)
		if readLocalRules(p) != nil {
			t.Errorf("%s: bad file was used", name)
		}
	}
	if readLocalRules(filepath.Join(dir, "missing.json")) != nil || readLocalRules("") != nil {
		t.Error("missing file")
	}
}

func indexOf(t []causeDef, id string) int {
	for i, d := range t {
		if d.ID == id {
			return i
		}
	}
	return -1
}
