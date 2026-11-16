<script lang="ts">
	import { onMount } from 'svelte';

	import { install, loadInstallMethod, METHODS, selectInstallMethod } from '#lib/install.svelte.ts';
	import { INSTALL } from '#lib/site.ts';
	import CopyButton from './copy-button.svelte';

	import type { InstallMethod } from '#lib/install.svelte.ts';

	const id = $props.id();
	const tabs: Record<string, HTMLButtonElement> = {};

	onMount(loadInstallMethod);

	// Arrow keys, Home and End move between tabs, as in the WAI-ARIA tabs pattern.
	const onKeydown = (event: KeyboardEvent, index: number) => {
		const next = {
			ArrowRight: index + 1,
			ArrowLeft: index - 1,
			Home: 0,
			End: METHODS.length - 1
		}[event.key];
		if (next === undefined) return;

		event.preventDefault();
		const method = METHODS[(next + METHODS.length) % METHODS.length];
		selectInstallMethod(method);
		tabs[method]?.focus();
	};

	const notes: Partial<Record<InstallMethod, string>> = { go: '# Go 1.26+' };
</script>

<div class="sw">
	<div class="sw-bar">
		<div class="sw-tabs" role="tablist" aria-label="Install method">
			{#each METHODS as method, index (method)}
				<button
					bind:this={tabs[method]}
					type="button"
					role="tab"
					id="{id}-tab-{method}"
					class:fallback={method === 'brew'}
					aria-controls="{id}-{method}"
					aria-selected={install.method === method}
					tabindex={install.method === method ? 0 : -1}
					onclick={() => selectInstallMethod(method)}
					onkeydown={(event) => onKeydown(event, index)}
				>
					{method}
				</button>
			{/each}
		</div>
		<CopyButton class="sw-copy" text={() => INSTALL[install.method].join(' && ')} />
	</div>

	{#each METHODS as method (method)}
		<div
			class="sw-panel"
			role="tabpanel"
			id="{id}-{method}"
			aria-labelledby="{id}-tab-{method}"
			tabindex="0"
			hidden={install.method !== method}
		>
			{#if notes[method]}
				<code class="ln cm">{notes[method]}</code>
			{/if}
			{#each INSTALL[method] as line (line)}
				<code class="ln">{line}</code>
			{/each}
		</div>
	{/each}
</div>

<style>
	.sw {
		background: var(--term-bg);
		color: var(--term-ink);
		border: 1px solid var(--term-rule);
		border-radius: 10px;
		overflow: hidden;
		font: 400 14px/1.55 var(--mono);
		max-width: 620px;
	}

	/* bar */
	.sw-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		padding: 6px;
		background: var(--term-bar);
		border-bottom: 1px solid var(--term-rule);
	}

	.sw-tabs {
		display: flex;
		gap: 2px;
	}

	.sw-tabs button {
		border: 0;
		background: none;
		color: var(--term-dim);
		cursor: pointer;
		font: 500 13.5px/1 var(--mono);
		padding: 8px 12px;
		border-radius: 6px;
	}

	.sw-tabs button:hover {
		color: var(--term-ink);
	}

	.sw-tabs button[aria-selected='true'] {
		color: var(--term-ink);
		background: rgba(255, 255, 255, 0.09);
	}

	/* Without JavaScript, only the brew tab and its panel show. */
	:global(html:not(.js)) .sw-tabs button:not(.fallback) {
		display: none;
	}

	.sw-bar :global(.copy.sw-copy) {
		border: 0;
		border-radius: 6px;
		background: none;
		padding: 8px 12px;
		min-width: 0;
		font-size: 13px;
	}

	.sw-bar :global(.copy.sw-copy:hover) {
		background: rgba(255, 255, 255, 0.06);
	}

	/* panels */
	.sw-panel {
		padding: 14px 16px 15px;
	}

	.sw-panel[hidden] {
		display: none;
	}

	.sw-panel:focus-visible {
		outline-offset: -3px;
	}

	.ln {
		background: none;
		border: 0;
		border-radius: 0;
		padding: 0 0 0 2ch;
		font-size: inherit;
		color: var(--term-ink);
		display: block;
		text-indent: -2ch;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.ln::before {
		content: '$ ';
		color: var(--term-dim);
	}

	.ln.cm {
		color: var(--term-dim);
	}

	.ln.cm::before {
		content: none;
	}

	@media (max-width: 520px) {
		.sw {
			font-size: 13px;
		}

		.sw-tabs button {
			padding: 8px 10px;
		}

		.sw-panel {
			padding: 12px 14px 13px;
		}
	}
</style>
