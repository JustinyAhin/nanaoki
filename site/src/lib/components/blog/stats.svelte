<script lang="ts">
	import { setContext } from 'svelte';

	import type { Snippet } from 'svelte';

	type Props = { sources?: string[]; width?: 'text' | 'wide' | 'full'; children?: Snippet };

	let { sources = [], width = 'text', children }: Props = $props();

	const id = $props.id();
	setContext('stats-id', id);
</script>

<div class="stats w-{width}">
	<div class="row">{@render children?.()}</div>
	{#if sources.length}
		<ol class="sources" aria-label="Sources">
			{#each sources as source, index (index)}
				<li id="src-{id}-{index + 1}">{source}</li>
			{/each}
		</ol>
	{/if}
</div>

<style>
	.stats {
		margin: 2.4em 0;
	}

	.row {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
		border-top: 1px solid var(--ink);
	}

	.row > :global(.stat) {
		padding: 18px 18px 6px 0;
	}

	.row > :global(.stat + .stat) {
		padding-left: 18px;
		border-left: 1px solid var(--rule);
	}

	.sources {
		margin: 14px 0 0;
		padding: 12px 0 0;
		border-top: 1px solid var(--rule);
		list-style: none;
		counter-reset: src;
		font: 400 13.5px/1.5 var(--sans);
		color: var(--ink-3);
	}

	.sources li {
		counter-increment: src;
		scroll-margin-top: 30vh;
	}

	.sources li::before {
		content: counter(src) '  ';
		font: 500 12px var(--mono);
		color: var(--accent-ink);
		white-space: pre;
	}

	.sources li:target {
		color: var(--ink);
	}

	@media (max-width: 520px) {
		.row {
			grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		}

		.row > :global(.stat:nth-child(odd)) {
			padding-left: 0;
			border-left: 0;
		}

		.row > :global(.stat:nth-child(n + 3)) {
			border-top: 1px solid var(--rule);
		}
	}
</style>
