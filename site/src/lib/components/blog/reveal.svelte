<script lang="ts">
	import { whenSeen } from '#lib/blog/in-view.ts';

	import type { ViewState } from '#lib/blog/in-view.ts';
	import type { Snippet } from 'svelte';

	let { children }: { children?: Snippet } = $props();

	let view = $state<ViewState>('idle');
</script>

<div class="reveal {view}" {@attach whenSeen({ onChange: (next) => (view = next) })}>
	{@render children?.()}
</div>

<style>
	.reveal {
		transition:
			opacity 0.7s ease,
			transform 0.7s cubic-bezier(0.2, 0.7, 0.2, 1);
	}

	.armed {
		opacity: 0;
		transform: translateY(18px);
	}

	.reveal > :global(:first-child) {
		margin-top: 0;
	}

	.reveal > :global(:last-child) {
		margin-bottom: 0;
	}

	@media (prefers-reduced-motion: reduce) {
		.reveal {
			transition: none;
		}
	}
</style>
