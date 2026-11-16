<script lang="ts">
	import CopyButton from '../copy-button.svelte';

	type Props = { code: string; language?: string; html: string };

	let { code, language = '', html }: Props = $props();
</script>

<figure class="code" aria-label={language ? `${language} code` : 'Code'}>
	<div class="bar">
		<span>{language || 'text'}</span>
		<CopyButton text={() => code} />
	</div>
	<!-- eslint-disable-next-line svelte/no-at-html-tags -- Shiki output, built at prerender from the post source. -->
	{@html html}
</figure>

<style>
	.code {
		margin: 1.8em 0;
		background: var(--term-bg);
		border: 1px solid var(--term-rule);
		border-radius: 10px;
		overflow: hidden;
	}

	.bar {
		display: flex;
		justify-content: space-between;
		align-items: stretch;
		min-height: 34px;
		background: var(--term-bar);
		border-bottom: 1px solid var(--term-rule);
	}

	.bar span {
		align-self: center;
		padding: 0 14px;
		font: 400 12.5px/1 var(--mono);
		color: var(--term-dim);
	}

	.code :global(pre) {
		margin: 0;
		padding: 16px 18px 18px;
		overflow-x: auto;
		font: 400 13.5px/1.6 var(--mono);
		background: transparent !important;
		scrollbar-width: thin;
		tab-size: 2;
	}

	.code :global(pre:focus-visible) {
		outline-offset: -2px;
	}

	@media (max-width: 720px) {
		.code :global(pre) {
			font-size: 13px;
			padding: 14px;
		}
	}
</style>
