<script lang="ts">
	import { resolve } from '$app/paths';

	import { CASE_TYPES } from '#lib/cases.ts';
	import CaseCard from '#lib/components/cases/case-card.svelte';
	import CommandLine from '#lib/components/command-line.svelte';
	import InstallSwitcher from '#lib/components/install-switcher.svelte';
	import Seo from '#lib/components/seo.svelte';
	import SiteFooter from '#lib/components/site-footer.svelte';
	import SiteHeader from '#lib/components/site-header.svelte';
	import { SITE } from '#lib/site.ts';

	import type { Case, CaseType } from '#lib/cases.ts';

	const cases: Case[] = [
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
				'This is my most common repeat: 465 sessions across 54 projects. Bash passes an unmatched glob through, but zsh stops the whole command. The agent adds quotes and moves on, then the next session does the same thing.',
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
			before: ['I typed some version of it in ', '181 sessions', ''],
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

	const filters = [
		{ key: 'all' as const, label: 'All', count: cases.length },
		...(Object.keys(CASE_TYPES) as CaseType[]).map((type) => ({
			key: type,
			label: CASE_TYPES[type],
			count: cases.filter((item) => item.type === type).length
		}))
	];

	let filter = $state<CaseType | 'all'>('all');

	const isShown = (item: Case) => filter === 'all' || item.type === filter;
	const shown = $derived(cases.filter(isShown).length);

	// A link to a hidden case resets the filter, so the case can show.
	const onHashChange = () => {
		const target = cases.find((item) => `#${item.id}` === location.hash);
		if (!target || isShown(target)) return;

		filter = 'all';
		requestAnimationFrame(() => document.getElementById(target.id)?.scrollIntoView());
	};
</script>

<!-- The terminal keeps whitespace, so each body line must stay as written. -->
<!-- prettier-ignore -->
{#snippet zshGlob()}<span class="blk"><span class="g">●</span> Bash(<span class="c">ls -a ../web/.dev.vars*</span>)</span><span class="ind"><span class="d">└</span> <span class="y">nanaoki guard:</span> [zsh_glob] zsh fails with "no matches found" when an unquoted glob matches no file: `../web/.dev.vars*`. Quote words with [ ] * ? (URLs, [slug] routes, --include=*.md). Suggested command: <span class="c">ls -a '../web/.dev.vars*'</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet zshEquals()}<span class="blk"><span class="g">●</span> Bash(<span class="c">ax "https://jobs.example.com/listing/1234" --md --budget 1500; echo =====; ax "https://jobs.example.com/listing/5678" --md</span>)</span><span class="ind"><span class="d">└</span> <span class="r">Blocked by hook.</span> <span class="y">nanaoki guard:</span> [zsh_eq] In zsh an unquoted word that starts with '=' expands to a command path, so `=====` fails with "not found". Quote it. Suggested command: <span class="c">ax "https://jobs.example.com/listing/1234" --md --budget 1500; echo '====='; ax "https://jobs.example.com/listing/5678" --md</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet nodeVersion()}<span class="blk"><span class="g">●</span> Bash(<span class="c">npm run build</span>)</span><span class="ind"><span class="d">└</span> <span class="y">nanaoki guard:</span> [node_version] The repo pins Node 22 (.nvmrc) but node on PATH is 26. Switch first. Suggested command: <span class="c">nvm use 22 &amp;&amp; &lt;command&gt;</span></span><span class="blk gap"><span class="d">If it runs anyway:</span></span><span class="ind"><span class="d">└</span> <span class="r">Toolchain check failed: expected Node 22.x for this project.</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] The project pins a Node major version. Use the pinned version (fnm/nvm use) before install/build. Fix seen: <span class="c">APP_NODE=~/.nvm/versions/node/v22.22.0/bin/node npm …</span> <span class="d">(project-d, 2026-09)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet missingScript()}<span class="blk"><span class="g">●</span> Bash(<span class="c">bun run gen</span>)</span><span class="ind"><span class="d">└</span> <span class="r">error: Script not found "gen"</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] The package.json script does not exist in this workspace. Read package.json scripts first. Fix seen: <span class="c">bun run format:check</span> <span class="d">(project-e, 2026-04)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet sveltekitResolve()}<span class="blk"><span class="g">●</span> Bash(<span class="c">bun run lint</span>)</span><span class="ind"><span class="d">└</span> <span class="r">24:25  error  Unexpected href link without resolve()  svelte/no-navigation-without-resolve</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] SvelteKit lint: wrap internal hrefs/goto with resolve() from $app/paths (svelte/no-navigation-without-resolve). Fix seen: <span class="c">bunx eslint &lt;5 files&gt;</span> <span class="d">(project-g, 2026-05)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet gitNoCommits()}<span class="blk"><span class="g">●</span> Bash(<span class="c">git status --short --branch &amp;&amp; git diff --stat &amp;&amp; git log -1 --oneline</span>)</span><span class="ind"><span class="d">└</span> <span class="r">fatal: your current branch 'main' does not have any commits yet</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] Error `fatal: your current branch &lt;q&gt; does not have any commits yet` after `git status --short &amp;&amp; git log --oneline -8 &amp;&amp; if test -f web/package.json; then sed -n '1,240p' web/package.json; fi` was fixed by: <span class="c">git status --short; git diff --stat; git diff -- &lt;4 files&gt; …</span> <span class="d">(project-i, 2026-08)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet notJson()}<span class="blk"><span class="g">●</span> Bash(<span class="c">python3 -I -c 'import json,glob; [json.load(open(f)) for f in sorted(glob.glob("data/q*.json"))]'</span>)</span><span class="ind"><span class="d">└</span> <span class="r">json.decoder.JSONDecodeError: Extra data: line 2 column 1 (char 2817)</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] CLI output was not JSON (banner, warning, or error text). Check the exit code and strip non-JSON lines before parsing. Fix seen: <span class="c">seo-cli keywords "…" | jq -r …</span> <span class="d">(project-k, 2026-04)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet dontChangeYet()}<span class="blk"><span class="d">&gt;</span> <span class="h">how could this be fixed? just asking, don't make any changes yet</span></span><span class="ind"><span class="d">└</span> Lessons from your earlier sessions (nanaoki):</span><span class="ind">{"  [L1] don't make any changes yet"}</span><span class="ind">{"  [L2] but don't make any changes yet"}</span>{/snippet}

<!-- prettier-ignore -->
{#snippet wrongLesson()}<span class="blk"><span class="g">●</span> Bash(<span class="c">rg -n "deploy" apps/cms/CLAUDE.md apps/web/CLAUDE.md</span>)</span><span class="ind"><span class="d">└</span> <span class="r">rg: apps/cms/CLAUDE.md: No such file or directory (os error 2)</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] A CLI was not installed or has another name. Check with 'command -v' once, then use the right name. Fix seen: <span class="c">rg -n "…" …</span> <span class="d">(project-n, 2026-02)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet codeComments()}<span class="blk"><span class="d">&gt;</span> <span class="h">add the helper for the share card and comment the code</span></span><span class="ind"><span class="d">└ (nanaoki printed nothing)</span></span>{/snippet}

<svelte:window onhashchange={onHashChange} />

<Seo
	title="Cases: when my coding agent repeated a fix it already knew · nanaoki"
	description="Ten real cases from 9 months of my own Claude Code and Codex history: the agent hit an error an earlier session had already fixed. What nanaoki showed it, and what happened in the replay, misses included."
	socialTitle="Ten times my coding agent repeated a fix it already knew"
	socialDescription="Real cases from my own Claude Code and Codex history, replayed with nanaoki. Eight caught or prevented, two missed."
	ogType="article"
	pageType="CollectionPage"
/>

<SiteHeader current="cases" />

<main id="top">
	<div class="wrap intro">
		<p class="eyebrow">Cases</p>
		<h1>
			Ten times my coding agent repeated a fix it already knew.
			<span class="quiet">Two of them still got through.</span>
		</h1>
		<div class="intro-grid">
			<div>
				<p class="lead">
					These come from 9 months of my own Claude Code and Codex history. Each one is an error
					that an earlier session had already worked out, often in another repo, and that came back
					anyway.
				</p>
				<p class="lead">
					I replayed every case with only the history from before that moment, and checked what
					nanaoki would have shown the agent. Most of the time it had the fix. Sometimes it didn't,
					so the misses are here too.
				</p>
			</div>
			<div class="fine">
				<p>
					The message in each terminal is what nanaoki 0.1.1 prints today for the same error. I
					changed project names, paths and hosts. The error and the shape of the fix are real.
				</p>
				<p>
					The verdicts come from my replay, scored by a Claude judge, on my own data. "AGENTS.md
					alone" is a file written from the same history and loaded on every turn.
				</p>
			</div>
		</div>

		<div class="tally">
			<div class="ok">
				<b>3</b>
				<p>caught by a guard before the command ran</p>
			</div>
			<div class="ok">
				<b>5</b>
				<p>prevented by the lesson shown after the first failure</p>
			</div>
			<div>
				<b>2</b>
				<p>missed. One lesson was wrong, one rule had no error to hook onto.</p>
			</div>
			<div>
				<b>3<small>{' of 10'}</small></b>
				<p>prevented by the AGENTS.md file alone</p>
			</div>
		</div>
	</div>

	<section class="cases" aria-label="Cases">
		<div class="wrap">
			<div class="filterbar">
				<div class="filters" role="group" aria-label="Filter cases by type">
					{#each filters as option (option.key)}
						<button
							type="button"
							aria-pressed={filter === option.key}
							onclick={() => (filter = option.key)}
						>
							{option.label}<span>{option.count}</span>
						</button>
					{/each}
				</div>
				<p class="count" aria-live="polite">Showing {shown} of {cases.length}</p>
			</div>

			{#each cases as item, index (item.id)}
				<CaseCard {item} number={index + 1} hidden={!isShown(item)} />
			{/each}
		</div>
	</section>

	<section id="install">
		<div class="wrap cta-grid">
			<div>
				<p class="eyebrow">Your turn</p>
				<h2>Find the repeats in your own history.</h2>
				<p class="lead">
					My numbers come from my laptop. Yours will be different. The report takes about two
					minutes and tells you which errors your agents keep fixing twice.
				</p>
				<div class="cta-links">
					<a class="btn-secondary" href={resolve('/')}>Back to the home page</a>
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- External URL. -->
					<a href={SITE.repo}>Read the code on GitHub</a>
				</div>
			</div>
			<div class="cta-steps">
				<p class="label">1. Get the binary</p>
				<InstallSwitcher />
				<p class="label">2. See your number</p>
				<CommandLine command="nanaoki --split auto" />
				<p class="label">3. Install the hooks</p>
				<CommandLine command="nanaoki setup claude" />
			</div>
		</div>
	</section>
</main>

<SiteFooter current="cases" />

<style>
	/* intro */
	.intro {
		padding-block: 56px 72px;
	}

	h1 {
		font-size: clamp(34px, 3.2vw + 14px, 58px);
		line-height: 1.06;
		letter-spacing: -0.015em;
		max-width: 22ch;
	}

	.quiet {
		color: var(--ink-3);
	}

	.intro-grid {
		display: grid;
		grid-template-columns: minmax(0, 1.25fr) minmax(0, 1fr);
		gap: 56px;
		margin-top: 32px;
		align-items: start;
	}

	.intro .lead {
		margin: 0;
		font-size: 19px;
		max-width: 56ch;
	}

	.intro .lead + .lead {
		margin-top: 14px;
	}

	.fine {
		color: var(--ink-3);
		font-size: 14.5px;
		margin: 0;
		border-left: 2px solid var(--rule);
		padding-left: 16px;
	}

	.fine p {
		margin: 0 0 0.7em;
	}

	.fine p:last-child {
		margin: 0;
	}

	/* tally */
	.tally {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		margin-top: 56px;
		border-top: 1px solid var(--ink);
	}

	.tally div {
		padding: 20px 20px 4px 0;
	}

	.tally div + div {
		padding-left: 20px;
		border-left: 1px solid var(--rule);
	}

	.tally b {
		display: block;
		font: 400 clamp(38px, 4.6vw, 52px)/1 var(--serif);
		letter-spacing: -0.02em;
	}

	.tally b small {
		font-size: 0.45em;
		color: var(--ink-3);
		letter-spacing: 0;
	}

	.tally p {
		color: var(--ink-2);
		font-size: 15px;
		margin: 10px 0 0;
	}

	.tally .ok b {
		color: var(--ok);
	}

	/* filters */
	.cases {
		padding-top: 0;
		border-top: 0;
	}

	.filterbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px 24px;
		flex-wrap: wrap;
		padding: 18px 0;
		border-bottom: 1px solid var(--rule);
	}

	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.filters button {
		font: 500 14px/1 var(--sans);
		color: var(--ink-2);
		background: var(--card);
		border: 1px solid var(--rule);
		border-radius: 999px;
		padding: 9px 14px;
		cursor: pointer;
	}

	.filters button:hover {
		border-color: var(--ink-3);
		color: var(--ink);
	}

	.filters button[aria-pressed='true'] {
		background: var(--ink);
		border-color: var(--ink);
		color: var(--bg);
	}

	.filters button span {
		font-family: var(--mono);
		font-size: 12px;
		opacity: 0.7;
		margin-left: 6px;
	}

	.count {
		font: 400 13px/1 var(--mono);
		color: var(--ink-3);
		margin: 0;
	}

	/* The filters need JavaScript. */
	:global(html:not(.js)) .filterbar {
		display: none;
	}

	/* call to action */
	.cta-grid {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.1fr);
		gap: 56px;
		align-items: start;
	}

	.cta-steps {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.label {
		font: 500 12.5px/1 var(--mono);
		color: var(--ink-3);
		text-transform: uppercase;
		letter-spacing: 0.06em;
		margin: 8px 0 0;
	}

	.cta-links {
		display: flex;
		gap: 10px 18px;
		flex-wrap: wrap;
		align-items: center;
		margin-top: 28px;
		font-size: 15px;
	}

	@media (max-width: 980px) {
		.intro-grid,
		.cta-grid {
			grid-template-columns: minmax(0, 1fr);
			gap: 28px;
		}

		.tally {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}

		.tally div:nth-child(3) {
			padding-left: 0;
			border-left: 0;
		}

		.tally div:nth-child(n + 3) {
			border-top: 1px solid var(--rule);
			margin-top: 16px;
		}
	}

	@media (max-width: 720px) {
		.intro {
			padding-block: 24px 48px;
		}
	}

	@media (max-width: 520px) {
		.tally {
			grid-template-columns: minmax(0, 1fr);
		}

		.tally div,
		.tally div + div {
			padding-left: 0;
			border-left: 0;
		}

		.tally div + div {
			border-top: 1px solid var(--rule);
			margin-top: 16px;
		}

		.filters button {
			padding: 8px 12px;
			font-size: 13.5px;
		}
	}
</style>
