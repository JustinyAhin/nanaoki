<script lang="ts">
	import { whenSeen } from '#lib/blog/in-view.ts';

	import type { ViewState } from '#lib/blog/in-view.ts';
	import type { Snippet } from 'svelte';

	let { n, children }: { n: number; children?: Snippet } = $props();

	let view = $state<ViewState>('idle');
</script>

<li class="step {view}" {@attach whenSeen({ onChange: (next) => (view = next), threshold: 0.6 })}>
	<span class="n" aria-hidden="true">{String(n).padStart(2, '0')}</span>
	<div class="body">{@render children?.()}</div>
</li>

<style>
	.step {
		position: relative;
		display: grid;
		grid-template-columns: 3.2rem minmax(0, 1fr);
		gap: 0 4px;
		padding-bottom: 1.4em;
	}

	/* The thread between the numbers. */
	.step:not(:last-child)::after {
		content: '';
		position: absolute;
		left: 0.95rem;
		top: 2.3rem;
		bottom: 0.3rem;
		width: 1px;
		background: var(--rule);
		transform-origin: top;
		transition: transform 0.7s ease 0.15s;
	}

	.n {
		font: 400 1.6rem/1.1 var(--serif);
		color: var(--ink-3);
		font-variant-numeric: lining-nums;
		transition: color 0.4s;
	}

	.seen .n {
		color: var(--ink);
	}

	.body {
		padding-top: 0.15em;
		transition:
			opacity 0.6s ease,
			transform 0.6s ease;
	}

	.body :global(p:last-child) {
		margin-bottom: 0;
	}

	.armed .body {
		opacity: 0;
		transform: translateY(12px);
	}

	.armed::after {
		transform: scaleY(0);
	}

	@media (prefers-reduced-motion: reduce) {
		.step::after,
		.n,
		.body {
			transition: none;
		}
	}
</style>
