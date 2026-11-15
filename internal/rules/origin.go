package rules

import (
	"regexp"
	"strings"
)

// Origins of a user message. Only OriginTyped counts as a restated rule or a pasted error.
const (
	OriginTyped    = "typed"    // the human wrote it
	OriginCommand  = "command"  // the human typed a slash command or skill call (/name, $name, [$name](...))
	OriginInjected = "injected" // a harness, app, hook or automation wrote it (system blocks, command expansion)
	OriginTemplate = "template" // the same long text in many sessions (prompt file, routine); set by the caller
	OriginScripted = "scripted" // a side session (subagent, codex exec, guardian): a script or a parent agent wrote it; set by the caller
)

// injectedRx matches the start of messages that a tool writes into the user role. These are tags and
// headers of Claude Code, Codex and wrapper apps, not of pasted content (a pasted HTML or XML snippet is typed).
var injectedRx = regexp.MustCompile(`^\s*(?:<(?:system|system-reminder|command-name|command-message|command-args|` +
	`local-command-stdout|local-command-stderr|local-command-caveat|environment_context|user_instructions|INSTRUCTIONS|` +
	`heartbeat|automation|task-notification|subagent|skill|turn_aborted|user_shell_command)\b|` +
	`# AGENTS\.md instructions|Base directory for this skill:|Caveat: The messages below were generated)`)

// commandRx matches a message that starts with a command or skill call: "/humanizer ...", "$humanizer ...",
// "[$humanizer](/path/SKILL.md) ...". A path such as "/tmp/x" is not a command.
var commandRx = regexp.MustCompile(`^\s*(?:/[a-z][\w:-]*(?:\s|$|[,.;!?])|\$[a-z][\w-]*(?:\s|$|[,.;!?])|\[\$[\w:-]+\]\()`)

// MessageOrigin classifies one user message from its text alone. It returns OriginInjected, OriginCommand
// or OriginTyped. Templates and side sessions need more context (see the analyze package).
func MessageOrigin(text string) string {
	t := Strip(text)
	if injectedRx.MatchString(t) || autoRx.MatchString(Trunc(t, 400)) {
		return OriginInjected
	}
	if commandRx.MatchString(PromptBody(t)) {
		return OriginCommand
	}
	return OriginTyped
}

// TemplateKey is the text that identifies a template: the prompt body without pastes, lower case, with
// whitespace collapsed, cut to 2,000 characters. It is "" for bodies under TemplateMinLen characters
// (short messages such as "/humanizer" or "commit" are typed again by hand, not templates).
func TemplateKey(text string) string {
	b := strings.ToLower(strings.TrimSpace(wsRx.ReplaceAllLiteralString(PromptBody(text), " ")))
	if RuneLen(b) < TemplateMinLen {
		return ""
	}
	return Trunc(b, 2000)
}

// A body of TemplateMinLen+ characters that occurs in TemplateMinSessions+ sessions is a template.
const (
	TemplateMinLen      = 200
	TemplateMinSessions = 3
)
