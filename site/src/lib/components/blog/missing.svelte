<script lang="ts">
	type Props = { name: string; dev?: boolean };

	let { name, dev = false }: Props = $props();
</script>

<!-- Dev only. A production build fails on an unknown tag before it gets here. -->
{#if dev}
	<div class="missing">Unknown tag: <code>{name}</code>. Add it to the blog schema.</div>
{/if}

<style>
	.missing {
		margin: 2em 0;
		padding: 18px;
		border: 1px dashed var(--accent);
		border-radius: 10px;
		font: 400 14px/1.5 var(--mono);
		color: var(--accent-ink);
	}
</style>
