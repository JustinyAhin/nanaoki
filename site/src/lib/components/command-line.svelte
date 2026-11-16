<script lang="ts">
	import CopyButton from './copy-button.svelte';

	type Props = { command: string };

	let { command }: Props = $props();
</script>

<div class="cmd">
	<code>{command}</code>
	<CopyButton text={() => command} />
</div>

<style>
	.cmd {
		display: flex;
		align-items: stretch;
		width: 100%;
		max-width: 620px;
		background: var(--term-bg);
		color: var(--term-ink);
		border: 1px solid var(--term-rule);
		border-radius: 10px;
		overflow: hidden;
		font: 400 14px/1.5 var(--mono);
	}

	code {
		background: none;
		border: 0;
		border-radius: 0;
		font-size: inherit;
		color: var(--term-ink);
		overflow-wrap: normal;
		flex: 1;
		min-width: 0;
		padding: 13px 16px;
		overflow-x: auto;
		white-space: nowrap;
		scrollbar-width: thin;
	}

	code::before {
		content: '$ ';
		color: var(--term-dim);
	}

	@media (max-width: 520px) {
		.cmd {
			font-size: 13px;
		}
	}
</style>
