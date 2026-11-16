<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children?: Snippet } = $props();
</script>

<!-- A wide table scrolls inside its own box, so the page never scrolls sideways. -->
<div class="table-wrap">
	{@render children?.()}
</div>

<style>
	.table-wrap {
		margin: 2em 0;
		overflow-x: auto;
		scrollbar-width: thin;
	}

	.table-wrap :global(table) {
		width: 100%;
		border-collapse: collapse;
		font: 400 15px/1.45 var(--sans);
		font-variant-numeric: tabular-nums;
	}

	.table-wrap :global(th) {
		text-align: left;
		font-weight: 500;
		color: var(--ink);
		border-bottom: 1px solid var(--ink);
		padding: 0 14px 8px 0;
		vertical-align: bottom;
	}

	.table-wrap :global(td) {
		border-bottom: 1px solid var(--rule);
		padding: 9px 14px 9px 0;
		color: var(--ink-2);
		vertical-align: top;
	}

	.table-wrap :global(:is(th, td):last-child) {
		padding-right: 0;
	}
</style>
