package rules

import "testing"

func TestNorm(t *testing.T) {
	cases := map[string]string{
		"error: cannot open /Users/alice/x/y.ts at line 12": "error: cannot open <path> at line N",
		"fetch https://example.com/a?b=1 failed":            "fetch <url> failed",
		"commit 1a2b3c4d5e not found":                       "commit <hex> not found",
		`Error: Cannot find module 'left-pad'`:              "Error: Cannot find module <q>",
		`say ""x" twice`:                                    "say <q> twice", // '.' may consume the quote itself
		"\x1b[31mred\x1b[0m   text":                         "red text",
		"mail bob@example.com now":                          "mail <email> now",
		"unclosed 'quote":                                   "unclosed 'quote",
	}
	for in, want := range cases {
		if got := Norm(in); got != want {
			t.Errorf("Norm(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSignature(t *testing.T) {
	if got := Signature("Exit code 2\n\nsome output\nError: boom 42"); got != "Error: boom N" {
		t.Errorf("got %q", got)
	}
	if got := Signature("Exit code 1\nplain line"); got != "plain line" {
		t.Errorf("got %q", got)
	}
	if got := Signature(""); got != "<empty>" {
		t.Errorf("got %q", got)
	}
}

func TestProject(t *testing.T) {
	cases := map[string]string{
		"/Users/alice/work/projects/demo-app/src": "demo-app",
		"file:///Users/alice/work/consulting/x":   "x",
		"/Users/alice":                            "home",
		"/private/var/folders/tmp":                "scratch",
		"/Users/alice/code/thing":                 "thing",
		"":                                        "?",
	}
	for in, want := range cases {
		if got := Project(in); got != want {
			t.Errorf("Project(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCommandClassAndHead(t *testing.T) {
	if CommandClass("git -C repo commit -m x") != "commit" || CommandClass("bun run build") != "build_result" ||
		CommandClass("npx vitest run") != "test_result" || CommandClass("wrangler deploy") != "deploy" || CommandClass("ls") != "" {
		t.Error("command classes")
	}
	cases := map[[2]string]string{
		{"shell", "cd app && FOO=1 bun run check"}: "bun check",
		{"Bash", "/usr/bin/git status"}:            "git status",
		{"mcp:docs:search", "x"}:                   "mcp:docs:search",
		{"shell", "ls -la"}:                        "ls",
	}
	for in, want := range cases {
		if got := Head(in[0], in[1]); got != want {
			t.Errorf("Head(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestCauseOf(t *testing.T) {
	cases := []struct{ cmd, out, want string }{
		{"ls src/[x]", "zsh:1: no matches found: src/[x]", "zsh_glob_nomatch"},
		{"foo", "zsh:1: command not found: foo", "command_not_found"},
		{"bun run check", "svelte-check found 3 errors", "svelte_check_errors"},
		{"x", "all good", ""},
	}
	for _, c := range cases {
		if got := CauseOf(c.cmd, c.out); got != c.want {
			t.Errorf("CauseOf(%q, %q) = %q, want %q", c.cmd, c.out, got, c.want)
		}
	}
	if CauseOfEvent("user_error_paste", "zsh: command not found: foo", "") != "command_not_found" {
		t.Error("pasted log: the text is the output")
	}
}

func TestInformativeAndPaste(t *testing.T) {
	if Informative("<empty>") || Informative("N errors") || !Informative("Error: Cannot find module <q>") {
		t.Error("informative")
	}
	if _, _, ok := PasteError("short"); ok {
		t.Error("short text is not a paste")
	}
	txt := "context line one\nError: build failed at step N\nmore lines here\nand more lines here for length"
	if _, sig, ok := PasteError(txt); !ok || sig != "Error: build failed at step N" {
		t.Errorf("paste = %v %q", ok, sig)
	}
}

func TestPromptRules(t *testing.T) {
	if _, _, ok := PromptKeep("/clear"); ok {
		t.Error("slash commands are dropped")
	}
	if _, _, ok := PromptKeep("<system-reminder>x</system-reminder>"); ok {
		t.Error("tagged machine text is dropped")
	}
	body := PromptBody("please don't push yet\n<pasted_content>\nno fluff\n</pasted_content>")
	var hits []string
	for _, r := range PromptRules {
		if r.Rx.MatchString(body) {
			hits = append(hits, r.Key)
		}
	}
	if len(hits) != 1 || hits[0] != "commit_push_only_when_told" {
		t.Errorf("hits = %v (pasted content must be ignored)", hits)
	}
}

func TestPyToRE2(t *testing.T) {
	if !py(`^\w+$`, "").MatchString("café") {
		t.Error(`\w must match Unicode letters like Python`)
	}
	if !py(`[\w.]+`, "").MatchString("é.x") {
		t.Error(`\w inside a class`)
	}
	if !py(`can\'t`, "").MatchString("can't") {
		t.Error(`escaped quote`)
	}
}
