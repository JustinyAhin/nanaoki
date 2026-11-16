<script lang="ts">
	import { CASE_TYPES, inlineCode } from '#lib/cases.ts';
	import Terminal from '../terminal.svelte';

	import type { Case } from '#lib/cases.ts';

	type Props = { item: Case; number: number; hidden: boolean };

	let { item, number, hidden }: Props = $props();
</script>

<article class="case" id={item.id} {hidden}>
	<div>
		<div class="case-top">
			<span class="n">{String(number).padStart(2, '0')}</span>
			<span class="tag">{CASE_TYPES[item.type]}</span>
			<a class="anchor" href="#{item.id}" aria-label="Link to this case">#{item.id}</a>
		</div>
		<h2>{item.title}</h2>

		<!-- The error block keeps whitespace. Svelte turns a plain line break into a space. -->
		<!-- prettier-ignore -->
		<div class="err"><span class="d">{item.err.lead}</span> {item.err.text}{#if item.err.error}{'\n'}<span class="r">{item.err.error}</span>{/if}</div>

		<dl class="facts">
			<dt>Before</dt>
			<dd>{item.before[0]}<b>{item.before[1]}</b>{item.before[2]}</dd>
			<dt>This time</dt>
			<dd>{item.now}</dd>
			<dt>AGENTS.md alone</dt>
			<dd>{item.agentsMd}</dd>
		</dl>

		<div class="story">
			<p>
				{#each inlineCode(item.story) as { part, code }, index (index)}
					{#if code}<code>{part}</code>{:else}{part}{/if}
				{/each}
			</p>
		</div>

		<span class="badge {item.badge.kind}">
			<svg
				viewBox="0 0 20 20"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				aria-hidden="true"
			>
				{#if item.badge.kind === 'guard'}
					<path d="M10 2.5l6 2.5v4.5c0 4-2.6 6.7-6 8-3.4-1.3-6-4-6-8V5z" stroke-linejoin="round" />
					<path d="M7.2 10.2l2 2 3.8-4.2" stroke-linecap="round" stroke-linejoin="round" />
				{:else if item.badge.kind === 'lesson'}
					<path d="M4 10.5l4 4 8-9" stroke-linecap="round" stroke-linejoin="round" />
				{:else}
					<path d="M5.5 5.5l9 9M14.5 5.5l-9 9" stroke-linecap="round" />
				{/if}
			</svg>
			{item.badge.label}
		</span>
		{#if item.why}
			<span class="why">{item.why}</span>
		{/if}
	</div>

	<Terminal title={item.termTitle} label="What the agent saw">{@render item.term()}</Terminal>
</article>

<style>
	.case {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.12fr);
		gap: 28px 48px;
		padding: 48px 0;
		border-bottom: 1px solid var(--rule);
		scroll-margin-top: 16px;
		position: relative;
	}

	.case[hidden] {
		display: none;
	}

	.case:target::before {
		content: '';
		position: absolute;
		left: calc(-1 * min(var(--gutter), 20px));
		top: 48px;
		bottom: 48px;
		width: 3px;
		border-radius: 2px;
		background: var(--accent);
	}

	/* top line */
	.case-top {
		display: flex;
		align-items: center;
		gap: 12px;
		font: 500 12.5px/1 var(--mono);
		color: var(--ink-3);
		letter-spacing: 0.04em;
		margin-bottom: 14px;
	}

	.n {
		color: var(--accent-ink);
	}

	.tag {
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.anchor {
		margin-left: auto;
		text-decoration: none;
		color: var(--ink-3);
		padding: 4px 6px;
		border-radius: 6px;
	}

	.anchor:hover {
		color: var(--ink);
		background: var(--bg-2);
	}

	h2 {
		font-size: clamp(25px, 2.2vw + 10px, 31px);
		line-height: 1.16;
		max-width: 24ch;
	}

	/* error */
	.err {
		margin: 18px 0 0;
		background: var(--term-bg);
		color: var(--term-ink);
		border: 1px solid var(--term-rule);
		border-radius: 8px;
		padding: 11px 14px;
		font: 400 13px/1.55 var(--mono);
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.err .d {
		color: var(--term-dim);
	}

	.err .r {
		color: var(--term-red);
	}

	/* facts */
	.facts {
		margin: 20px 0 0;
		display: grid;
		grid-template-columns: max-content minmax(0, 1fr);
		gap: 8px 16px;
		font-size: 15px;
	}

	.facts dt {
		font: 500 12px/1.9 var(--mono);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--ink-3);
	}

	.facts dd {
		margin: 0;
		color: var(--ink-2);
	}

	.facts dd b {
		color: var(--ink);
		font-weight: 500;
	}

	.story {
		color: var(--ink-2);
		font-size: 16px;
		margin-top: 18px;
		max-width: 56ch;
	}

	.story p:last-child {
		margin-bottom: 0;
	}

	/* result */
	.badge {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		margin-top: 22px;
		font: 500 14px/1.2 var(--sans);
		padding: 8px 13px 8px 11px;
		border-radius: 8px;
		border: 1px solid;
	}

	.badge svg {
		width: 16px;
		height: 16px;
		flex: none;
	}

	.badge.guard,
	.badge.lesson {
		color: var(--ok);
		border-color: color-mix(in srgb, var(--ok) 35%, transparent);
		background: var(--ok-soft);
	}

	.badge.miss {
		color: var(--ink-2);
		border-color: var(--rule);
		border-style: dashed;
		background: transparent;
	}

	.why {
		display: block;
		color: var(--ink-3);
		font-size: 14.5px;
		margin-top: 8px;
		max-width: 54ch;
	}

	@media (max-width: 980px) {
		.case {
			grid-template-columns: minmax(0, 1fr);
			gap: 24px;
		}
	}

	@media (max-width: 720px) {
		.case {
			padding: 36px 0;
		}

		.case:target::before {
			left: -10px;
			top: 36px;
			bottom: 36px;
		}

		.err {
			font-size: 12.5px;
		}
	}

	@media (max-width: 520px) {
		.facts {
			grid-template-columns: minmax(0, 1fr);
			gap: 2px;
		}

		.facts dd {
			margin-bottom: 8px;
		}
	}
</style>
