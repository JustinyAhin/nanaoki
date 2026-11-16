<script lang="ts">
	import type { Snippet } from 'svelte';

	let { label, children }: { label: string; children?: Snippet } = $props();
</script>

<div class="pane">
	<p class="pane-label">{label}</p>
	{@render children?.()}
</div>

<style>
	.pane {
		min-width: 0;
	}

	.pane-label {
		font: 500 12.5px/1 var(--mono);
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--ink-3);
		margin: 1.2em 0 0.6em;
	}

	.pane > :global(:first-child:not(.pane-label)),
	.pane > :global(.pane-label + *) {
		margin-top: 0;
	}

	.pane > :global(:last-child) {
		margin-bottom: 0;
	}
</style>
