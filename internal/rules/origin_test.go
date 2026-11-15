package rules

import (
	"strings"
	"testing"
)

// All texts are synthetic.
func TestMessageOrigin(t *testing.T) {
	cases := []struct{ text, want string }{
		{"please run the humanizer on this copy", OriginTyped},
		{"did you run /humanizer ?", OriginTyped},
		{"did you [$humanizer](/home/u/.codex/skills/humanizer/SKILL.md) on it", OriginTyped},
		{"/humanizer", OriginCommand},
		{"/humanizer and copy to my clipboard", OriginCommand},
		{"  $humanizer, then shorten", OriginCommand},
		{"[$humanizer](/home/u/.codex/skills/humanizer/SKILL.md) the post", OriginCommand},
		{"/tmp/build.log has the error", OriginTyped}, // a path, not a command
		{"<system>\n## Plain language\nUse plain English.\n</system>", OriginInjected},
		{"<command-name>/humanizer</command-name>\n<command-message>humanizer</command-message>", OriginInjected},
		{"<heartbeat>\n  <automation id=\"x\">nightly</automation>\n</heartbeat>", OriginInjected},
		{"# AGENTS.md instructions for /home/u/app\n\n<INSTRUCTIONS>be brief</INSTRUCTIONS>", OriginInjected},
		{"Base directory for this skill: /home/u/.claude/skills/humanizer\n\n# Humanizer", OriginInjected},
		{"[Role Reminder: you are the verifier]", OriginInjected},
		{"<h3>Review Summary</h3>\nError: build failed", OriginTyped}, // pasted HTML is typed
		{"<pasted_content>\n/humanizer\n</pasted_content>\nfix this", OriginTyped},
	}
	for _, c := range cases {
		if got := MessageOrigin(c.text); got != c.want {
			t.Errorf("MessageOrigin(%q) = %s, want %s", c.text, got, c.want)
		}
	}
}

func TestTemplateKey(t *testing.T) {
	if TemplateKey("/humanizer") != "" {
		t.Error("short text must not be a template")
	}
	long := strings.Repeat("Review the diff and list every risk. ", 8)
	a := TemplateKey(long)
	b := TemplateKey("  " + strings.ToUpper(long) + "\n\n")
	if a == "" || a != b {
		t.Errorf("case and whitespace must not change the key: %q vs %q", a, b)
	}
	// pasted content is not part of the key: the typed part decides
	if TemplateKey("short note <pasted_content>"+long+"</pasted_content>") != "" {
		t.Error("a paste must not make a template")
	}
}
