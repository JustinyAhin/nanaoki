<script lang="ts">
	import { copyStatus, copyText } from '#lib/copy.svelte.ts';

	type Props = { text: () => string; class?: string };

	let { text, class: className }: Props = $props();

	let done = $state(false);
	let timer: ReturnType<typeof setTimeout> | undefined;

	const copy = async () => {
		if (!(await copyText(text()))) return;

		done = true;
		copyStatus.text = 'Copied to clipboard';
		clearTimeout(timer);
		timer = setTimeout(() => {
			done = false;
			copyStatus.text = '';
		}, 1600);
	};
</script>

<button class={['copy', className, { done }]} type="button" onclick={copy}>
	{done ? 'Copied' : 'Copy'}
</button>

<style>
	.copy {
		flex: none;
		border: 0;
		border-left: 1px solid var(--term-rule);
		background: var(--term-bar);
		color: var(--term-dim);
		font: 500 12.5px/1 var(--mono);
		padding: 0 14px;
		cursor: pointer;
		min-width: 74px;
	}

	.copy:hover {
		color: var(--term-ink);
	}

	.copy.done {
		color: var(--term-green);
	}

	/* The copy needs JavaScript. */
	:global(html:not(.js)) .copy {
		display: none;
	}

	@media (max-width: 520px) {
		.copy {
			min-width: 62px;
			padding: 0 10px;
		}
	}
</style>
