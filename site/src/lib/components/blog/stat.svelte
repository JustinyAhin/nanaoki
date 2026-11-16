<script lang="ts">
	import { getContext } from 'svelte';

	type Props = {
		value: string | number;
		label: string;
		source?: string;
		// Set by the stats row: it lists the sources under the row.
		standalone?: boolean;
		note?: number;
	};

	let { value, label, source, standalone = true, note }: Props = $props();

	const statsId = getContext<string | undefined>('stats-id');
	const shown = $derived(typeof value === 'number' ? value.toLocaleString('en-US') : value);
</script>

<div class={['stat', { standalone }]}>
	<p class="value">
		{shown}{#if source && note && statsId}<a class="mark" href="#src-{statsId}-{note}"
				><span class="sr">Source {note}: </span><span aria-hidden="true">{note}</span><span
					class="tip"
					aria-hidden="true">{source}</span
				></a
			>{/if}
	</p>
	<p class="label">{label}</p>
	{#if source && standalone}
		<p class="source">Source: {source}</p>
	{/if}
</div>

<style>
	.stat.standalone {
		margin: 2.2em 0;
		padding-top: 16px;
		border-top: 1px solid var(--ink);
	}

	.value {
		position: relative;
		margin: 0;
		font: 400 clamp(38px, 3vw + 20px, 54px) / 1 var(--serif);
		letter-spacing: -0.02em;
		color: var(--ink);
	}

	.label {
		margin: 10px 0 0;
		font: 400 15px/1.45 var(--sans);
		color: var(--ink-2);
		text-wrap: pretty;
	}

	.source {
		margin: 8px 0 0;
		font: 400 13.5px/1.45 var(--sans);
		color: var(--ink-3);
	}

	.mark {
		font: 500 13px/1 var(--mono);
		vertical-align: super;
		margin-left: 3px;
		color: var(--accent-ink);
		text-decoration: none;
		letter-spacing: 0;
	}

	.tip {
		position: absolute;
		left: 0;
		bottom: calc(100% + 8px);
		z-index: 2;
		width: max-content;
		max-width: min(280px, 80vw);
		padding: 8px 10px;
		background: var(--ink);
		color: var(--bg);
		border-radius: 6px;
		font: 400 13px/1.4 var(--sans);
		opacity: 0;
		visibility: hidden;
		transition:
			opacity 0.15s,
			visibility 0.15s;
		pointer-events: none;
	}

	.mark:hover .tip,
	.mark:focus-visible .tip {
		opacity: 1;
		visibility: visible;
	}

	@media (prefers-reduced-motion: reduce) {
		.tip {
			transition: none;
		}
	}
</style>
