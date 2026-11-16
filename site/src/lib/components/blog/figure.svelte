<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		src?: string;
		alt?: string;
		caption?: string;
		width?: 'text' | 'wide' | 'full';
		children?: Snippet;
	};

	let { src, alt = '', caption, width = 'text', children }: Props = $props();
</script>

<figure class="figure w-{width}">
	{#if src}
		<img {src} {alt} loading="lazy" decoding="async" />
	{:else}
		{@render children?.()}
	{/if}
	{#if caption}<figcaption>{caption}</figcaption>{/if}
</figure>

<style>
	.figure {
		margin: 2.2em 0;
	}

	img {
		width: 100%;
		border: 1px solid var(--rule);
		border-radius: 10px;
		background: var(--card);
	}

	.w-full img {
		border-radius: 0;
		border-inline: 0;
	}

	figcaption {
		margin-top: 12px;
		font: 400 14px/1.5 var(--sans);
		color: var(--ink-3);
		max-width: 62ch;
		text-wrap: pretty;
	}

	.w-full figcaption {
		width: min(var(--text), 100% - 2 * var(--gutter));
		margin-inline: auto;
	}
</style>
