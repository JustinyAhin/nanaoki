<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children?: Snippet } = $props();
</script>

<ol class="steps">{@render children?.()}</ol>

<style>
	.steps {
		list-style: none;
		margin: 2.2em 0;
		padding: 0;
	}
</style>
