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

export { CASE_TYPES, inlineCode };
export type { Case, CaseType };
