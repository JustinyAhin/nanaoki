<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		title: string;
		label: string;
		// The home page shows one example as an image. Each case is a figure.
		as?: 'image' | 'figure';
		children: Snippet;
	};

	let { title, label, as = 'figure', children }: Props = $props();
</script>

{#snippet body()}
	<div class="term-bar">
		<i></i>
		<i></i>
		<i></i>
		<span>{title}</span>
	</div>
	<div class="term-body">{@render children()}</div>
{/snippet}

{#if as === 'image'}
	<div class="term" role="img" aria-label={label}>{@render body()}</div>
{:else}
	<figure class="term" aria-label={label}>{@render body()}</figure>
{/if}

<style>
	.term {
		background: var(--term-bg);
		color: var(--term-ink);
		border: 1px solid var(--term-rule);
		border-radius: 12px;
		overflow: hidden;
		font: 400 13.5px/1.6 var(--mono);
		margin: 0;
		align-self: start;
	}

	.term-bar {
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 11px 14px;
		background: var(--term-bar);
		border-bottom: 1px solid var(--term-rule);
	}

	.term-bar i {
		width: 11px;
		height: 11px;
		border-radius: 50%;
		background: #3a3f47;
	}

	.term-bar span {
		margin-left: 10px;
		color: var(--term-dim);
		font-size: 12.5px;
		font-family: var(--sans);
	}

	/* Callers write the body with these classes. */
	.term-body {
		padding: 18px 20px 20px;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.term-body :global(.h) {
		color: var(--term-ink);
		font-weight: 500;
	}

	.term-body :global(.d) {
		color: var(--term-dim);
	}

	.term-body :global(.c) {
		color: var(--term-cyan);
	}

	.term-body :global(.g) {
		color: var(--term-green);
	}

	.term-body :global(.r) {
		color: var(--term-red);
	}

	.term-body :global(.y) {
		color: var(--term-yellow);
	}

	.term-body :global(.blk) {
		display: block;
		margin-top: 14px;
	}

	.term-body :global(.blk:first-child) {
		margin-top: 0;
	}

	.term-body :global(.gap) {
		margin-top: 22px;
	}

	.term-body :global(.ind) {
		display: block;
		padding-left: 2.2ch;
		text-indent: -2.2ch;
	}

	@media (max-width: 720px) {
		.term {
			font-size: 12.5px;
		}

		.term-body {
			padding: 14px 14px 16px;
		}
	}
</style>
