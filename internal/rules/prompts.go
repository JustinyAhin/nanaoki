package rules

import (
	"regexp"
	"strings"
)

// Rule is a named regex over user prompts.
type Rule struct {
	Key, Kind, Label string
	Rx               *regexp.Regexp
}

// PromptRules: preferences first (in the Python order), then the recall request, then memory phrases.
var PromptRules = []Rule{
	{"no_ai_attribution", "pref", "No AI attribution in commits/PRs", py(`co-?author|generated (by|with) (claude|codex)|claude (code )?attribution|\battribution\b|no codex in|don'?t (mention|add) (claude|codex)`, "(?i)")},
	{"commit_push_only_when_told", "pref", "Commit or push only when told", py(`don'?t (commit|push)|do not (commit|push)|not push|(commit|push)[^.\n]{0,30}(but )?don'?t push|before i tell you|don'?t do the pr yet`, "(?i)")},
	{"investigate_only_no_changes", "pref", "Investigate only, no changes yet", py(`don'?t make any (changes|change|files)|no changes yet|don'?t change anything|do not (make|change) any`, "(?i)")},
	{"short_no_fluff", "pref", "Short, no fluff", py(`no fluff|straight to the point|keep (it|the (pr )?description|the message) (short|simple|concise)|be concise`, "(?i)")},
	{"deploy_wait_verify_prod", "pref", "Deploy, wait, verify on production", py(`wait (a few|a couple of|two|three|\d+) (minutes|mins)[^.\n]{0,40}(deploy|test|verify|check)|verify (it )?(on|in) prod`, "(?i)")},
	{"use_subagents_parallel", "pref", "Use subagents / parallel sessions", py(`\b(in|use|have) (a )?sub-?agents?\b|sub-?sessions?|in parallel`, "(?i)")},
	{"copy_setup_from_sibling", "pref", "Copy setup from a sibling project", py(`(same|like) (setup|structure|agents\.md) (as|from|in)|copy (the )?(setup|structure|agents\.md)`, "(?i)")},
	{"no_tests_unless_needed", "pref", "No tests unless needed", py(`don'?t add tests|only add tests when|tests for the sake|no (unit )?tests`, "(?i)")},
	{"pr_draft", "pref", "Open PRs as draft", py(`\bdraft pr\b|open (it|the pr) as (a )?draft|as a draft`, "(?i)")},
	{"recall_request", "recall", `"Check my past sessions" request`, py(`(check|look (at|into|through)|search|read|go through|find|review)[^.\n?]{0,40}(recent|previous|past|other|earlier|last|older)[^.\n?]{0,25}(session|conversation|thread|chat|run)s?\b|(claude code|codex) (session|conversation|thread|chat)s\b|in (a|one of the|another|the other) (codex|claude)[^.\n?]{0,20}(session|conversation|thread|chat)`, "(?i)")},
	{"remember", "mem", `"remember" / "don't forget"`, py(`\bremember\b|\bdon'?t forget\b`, "(?i)")},
	{"already_did", "mem", `"we already did/fixed ..."`, py(`\b(we|you|i)('ve| have)? already (did|done|fixed|tried|added|made|built|discussed|decided|set)|isn'?t this something we'?ve`, "(?i)")},
	{"like_last_time", "mem", `"like last time"`, py(`\b(like|as) (last time|we did( before)?|i said|i told you|discussed|before)\b`, "(?i)")},
	{"i_told_you", "mem", `"I told you"`, py(`\bi (told|asked) you\b|\bi already (told|said|asked|mentioned)\b`, "(?i)")},
	{"continue_handoff", "mem", `"continue from / handoff"`, py(`\bcontinue (from|where)\b|\bpick up (where|from)\b|\bhand ?off\b`, "(?i)")},
}

var (
	autoRx   = py(`^\s*(current ws\.agent\.snapshot|coordinator\b|\[role reminder|you are a (subagent|sub-agent|worker|implementor|verifier)|automation:|# (task|automation)|<(subagent|task|automation))|\[Role Reminder:`, "(?i)")
	pastedRx = py(`<pasted_content>.*?</pasted_content>`, "(?s)")
)

// PromptKeep applies the prompt filter. It returns the stripped text and the dedupe key part.
func PromptKeep(text string) (string, string, bool) {
	t := Strip(text)
	if t == "" || (strings.HasPrefix(t, "<") && !strings.HasPrefix(t, "<pasted_content")) || strings.HasPrefix(t, "Call your image_gen") {
		return "", "", false
	}
	if strings.HasPrefix(t, "/") && RuneLen(t) < 40 {
		return "", "", false
	}
	if autoRx.MatchString(Trunc(t, 400)) {
		return "", "", false
	}
	return t, Trunc(wsRx.ReplaceAllLiteralString(t, " "), 200), true
}

// PromptBody removes pasted content before rules are matched.
func PromptBody(t string) string { return pastedRx.ReplaceAllLiteralString(t, " ") }
