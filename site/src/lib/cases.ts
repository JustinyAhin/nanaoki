import {
	codeComments,
	dontChangeYet,
	gitNoCommits,
	missingScript,
	nodeVersion,
	notJson,
	sveltekitResolve,
	wrongLesson,
	zshEquals,
	zshGlob
} from './components/cases/case-terms.svelte';

import type { Snippet } from 'svelte';

const CASE_TYPES = {
	shell: 'Shell',
	node: 'Node/JS',
	framework: 'Framework',
	git: 'Git',
	prefs: 'Preferences'
} as const;

type CaseType = keyof typeof CASE_TYPES;

type Case = {
	id: string;
	type: CaseType;
	title: string;
	// The first line of the error block: a dim lead ("$", ">" or a note), then text.
	err: { lead: string; text: string; error?: string };
	// "Before" text with one bold part: [before, bold, after].
	before: [string, string, string];
	now: string;
	agentsMd: string;
	// Backticks mark inline code.
	story: string;
	badge: { kind: 'guard' | 'lesson' | 'miss'; label: string };
	why?: string;
	termTitle: string;
	term: Snippet;
};

// Splits "a `b` c" into text and code parts.
const inlineCode = (text: string) =>
	text.split('`').map((part, index) => ({ part, code: index % 2 === 1 }));

const CASES: Case[] = [
	{
		id: 'zsh-glob',
		type: 'shell',
		title: 'zsh gives up on a glob that matches nothing',
		err: {
			lead: '$',
			text: 'ls -a ../web/.dev.vars*',
			error: '(eval):1: no matches found: ../web/.dev.vars*'
		},
		before: ['Fixed in ', 'project-a', ', about 8 months earlier'],
		now: 'project-b',
		agentsMd: 'Prevented',
		story:
			'This is my most common repeat: 471 sessions across 56 projects. Bash passes an unmatched glob through, but zsh stops the whole command. The agent adds quotes and moves on, then the next session does the same thing.',
		badge: { kind: 'guard', label: 'Caught by a guard before running' },
		why: 'The glob matched no file, so the guard warned and gave the quoted command.',
		termTitle: 'before the command runs · PreToolUse',
		term: zshGlob
	},
	{
		id: 'zsh-equals',
		type: 'shell',
		title: 'A row of equals signs breaks a zsh command',
		err: {
			lead: '$',
			text: 'ax "…/listing/1234" --md; echo =====; ax "…/listing/5678" --md',
			error: '(eval):1: ===== not found'
		},
		before: ['Hit and fixed in ', 'project-c', ', about 2.5 weeks earlier'],
		now: 'a scratch folder',
		agentsMd: 'Not prevented',
		story:
			'The agent wanted a separator between two outputs. In zsh, a word that starts with `=` expands to the path of a command, so `=====` goes looking for a program called `====`. My AGENTS.md had notes on zsh quoting, but nothing about `=`.',
		badge: { kind: 'guard', label: 'Blocked by a guard before running' },
		why: "The command can't work as written, so this guard blocks it instead of warning.",
		termTitle: 'before the command runs · PreToolUse',
		term: zshEquals
	},
	{
		id: 'node-version',
		type: 'node',
		title: 'The build wants Node 22. The shell has another one.',
		err: {
			lead: '$',
			text: 'npm run build',
			error: 'Toolchain check failed: expected Node 22.x for this project.'
		},
		before: ['Fixed in ', 'the same project', ', 7 hours earlier'],
		now: 'project-d',
		agentsMd: 'Not prevented',
		story:
			'The repo pins Node 22, and a newer Node was first on PATH. One session found the fix. Seven hours later, a new session started from zero. The only Node advice in my AGENTS.md was "use bun", which does nothing for a version pin.',
		badge: { kind: 'guard', label: 'Caught by a guard before running' },
		why: 'The guard compares the pin with the node on PATH. If the build runs anyway, the lesson comes after the failure.',
		termTitle: 'before and after · PreToolUse, PostToolUseFailure',
		term: nodeVersion
	},
	{
		id: 'missing-script',
		type: 'node',
		title: 'A script name carried over from another repo',
		err: { lead: '$', text: 'bun run gen', error: 'error: Script not found "gen"' },
		before: ['Fixed in ', 'project-e', ', about 4.5 months earlier'],
		now: 'project-f',
		agentsMd: 'Not prevented',
		story:
			"Most of my repos have a `gen` script. This one didn't. The fix is boring: read `package.json` before guessing. My AGENTS.md made it worse, because it told the agent to run `bun run gen`.",
		badge: { kind: 'lesson', label: 'Prevented by the lesson' },
		termTitle: 'after the failure · PostToolUseFailure',
		term: missingScript
	},
	{
		id: 'sveltekit-resolve',
		type: 'framework',
		title: 'SvelteKit lint wants resolve() on every internal link',
		err: {
			lead: '$',
			text: 'bun run lint',
			error:
				'24:25  error  Unexpected href link without resolve()  svelte/no-navigation-without-resolve'
		},
		before: ['Fixed in ', 'project-g', ', about 4 months earlier'],
		now: 'project-h',
		agentsMd: 'Not prevented',
		story:
			'A newer eslint rule. Three of my SvelteKit repos hit it between September and November, and each time the agent worked out the fix from scratch. My AGENTS.md only said that two other repos had turned the rule off, which points the wrong way.',
		badge: { kind: 'lesson', label: 'Prevented by the lesson' },
		termTitle: 'after the failure · PostToolUseFailure',
		term: sveltekitResolve
	},
	{
		id: 'git-no-commits',
		type: 'git',
		title: 'git log in a repo with no commits yet',
		err: {
			lead: '$',
			text: 'git status --short --branch && git diff --stat && git log -1 --oneline',
			error: "fatal: your current branch 'main' does not have any commits yet"
		},
		before: ['Fixed in ', 'project-i', ', about 5 weeks earlier'],
		now: 'project-j',
		agentsMd: 'Not prevented',
		story:
			'A fresh repo. The agent always opens with the same look around, and `git log` fails until the first commit. The lesson worked, but only just. It shows the command that worked and leaves out the reason.',
		badge: { kind: 'lesson', label: 'Prevented by the lesson' },
		termTitle: 'after the failure · PostToolUseFailure',
		term: gitNoCommits
	},
	{
		id: 'not-json',
		type: 'shell',
		title: "Parsing output that wasn't clean JSON",
		err: {
			lead: '$',
			text: "python3 -I -c 'import json,glob; …'",
			error: 'json.decoder.JSONDecodeError: Extra data: line 2 column 1'
		},
		before: ['Fixed in ', 'project-k', ', about 6 months earlier'],
		now: 'project-l',
		agentsMd: 'Not prevented (the second judge said yes)',
		story:
			"Six months earlier it was a CLI that printed a warning before its JSON. This time the files had more after the first JSON value. It's the same mistake: the agent parsed first and looked at the input second. The judge put this one at 50%, right on the line, because the earlier fix was a `jq` pipe and not Python.",
		badge: { kind: 'lesson', label: 'Prevented by the lesson' },
		termTitle: 'after the failure · PostToolUseFailure',
		term: notJson
	},
	{
		id: 'dont-change-yet',
		type: 'prefs',
		title: '"Don\'t make any changes yet"',
		err: { lead: '>', text: "how could this be fixed? just asking, don't make any changes yet" },
		before: ['I typed some version of it in ', '183 sessions', ''],
		now: 'project-m',
		agentsMd: 'Prevented',
		story:
			"This repeat is mine, not the agent's. I ask a question and the agent starts editing files. nanaoki shows the rule at the first prompt of a session when the words match. The AGENTS.md file got it too. A rule this stable belongs in the file, and `nanaoki rules suggest` proposes the line for you.",
		badge: { kind: 'lesson', label: 'Prevented by the lesson' },
		termTitle: 'first prompt of the session · UserPromptSubmit',
		term: dontChangeYet
	},
	{
		id: 'wrong-lesson',
		type: 'shell',
		title: 'A missing file, read as a missing tool',
		err: {
			lead: '$',
			text: 'rg -n "deploy" apps/cms/CLAUDE.md apps/web/CLAUDE.md',
			error: 'rg: apps/cms/CLAUDE.md: No such file or directory (os error 2)'
		},
		before: ['Same error in ', 'the same project', ', about 4 months earlier'],
		now: 'project-g',
		agentsMd: 'Not prevented',
		story:
			"The agent guessed a path that didn't exist. nanaoki matched the error to the wrong cause and said the tool was missing. The tool was fine, and the file just wasn't there. A lesson that points the wrong way costs more than no lesson at all.",
		badge: { kind: 'miss', label: 'Missed' },
		why: 'The lesson matched the words of the error, not its cause.',
		termTitle: 'after the failure · PostToolUseFailure',
		term: wrongLesson
	},
	{
		id: 'code-comments',
		type: 'prefs',
		title: 'Comments that only restate the code',
		err: {
			lead: 'No error.',
			text: 'The agent writes the code comments I keep asking it not to write.'
		},
		before: [
			'Asked for in ',
			'earlier sessions',
			'. The AGENTS.md written from them has the line.'
		],
		now: 'project-o',
		agentsMd: 'Prevented (the second judge said no)',
		story:
			"My rule is to comment only when the code can't explain itself. Nothing fails when the agent ignores it, so no hook fires after an error. The first prompt of the session didn't share enough words with the rule either. This is what the always-loaded file is good at.",
		badge: { kind: 'miss', label: 'Missed' },
		why: 'A style rule has no error for a hook to match.',
		termTitle: 'first prompt of the session · UserPromptSubmit',
		term: codeComments
	}
];

export { CASE_TYPES, CASES, inlineCode };
export type { Case, CaseType };
