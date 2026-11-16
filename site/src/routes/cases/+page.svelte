<script lang="ts">
	import { resolve } from '$app/paths';

	import { CASE_TYPES, CASES } from '#lib/cases.ts';
	import CaseCard from '#lib/components/cases/case-card.svelte';
	import CommandLine from '#lib/components/command-line.svelte';
	import InstallSwitcher from '#lib/components/install-switcher.svelte';
	import Seo from '#lib/components/seo.svelte';
	import SiteFooter from '#lib/components/site-footer.svelte';
	import SiteHeader from '#lib/components/site-header.svelte';
	import { SITE } from '#lib/site.ts';

	import type { Case, CaseType } from '#lib/cases.ts';

	const cases = CASES;

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
