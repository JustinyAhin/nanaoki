<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		type?: 'note' | 'warn' | 'quote';
		title?: string;
		cite?: string;
		children?: Snippet;
	};

	let { type = 'note', title, cite, children }: Props = $props();
</script>

{#if type === 'quote'}
	<figure class="callout quote">
		<blockquote>{@render children?.()}</blockquote>
		{#if cite}<figcaption>{cite}</figcaption>{/if}
	</figure>
{:else}
	<aside class="callout {type}" aria-label={title ?? (type === 'warn' ? 'Warning' : 'Note')}>
		<p class="label">{title ?? (type === 'warn' ? 'Careful' : 'Note')}</p>
		{@render children?.()}
	</aside>
{/if}

<style>
	.callout {
		margin: 2em 0;
	}

	.note,
	.warn {
		padding: 18px 22px;
		background: var(--card);
		border: 1px solid var(--rule);
		border-left: 3px solid var(--ink-3);
		border-radius: 4px 10px 10px 4px;
		font: 400 16.5px/1.6 var(--sans);
		color: var(--ink-2);
	}

	.warn {
		border-left-color: var(--accent);
	}

	.label {
		font: 500 12.5px/1 var(--mono);
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--ink-3);
		margin: 0 0 10px;
	}

	.warn .label {
		color: var(--accent-ink);
	}

	.callout :global(p:last-child) {
		margin-bottom: 0;
	}

	.quote {
		padding: 4px 0 4px 26px;
		border-left: 1px solid var(--ink);
	}

	blockquote {
		margin: 0;
		font: 400 clamp(22px, 1.6vw + 14px, 27px) / 1.35 var(--serif);
		color: var(--ink);
		text-wrap: pretty;
	}

	figcaption {
		margin-top: 12px;
		font: 400 14px/1.4 var(--sans);
		color: var(--ink-3);
	}

	figcaption::before {
		content: '— ';
	}
</style>
