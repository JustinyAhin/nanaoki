<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		label?: string;
		labels: string[];
		width?: 'text' | 'wide' | 'full';
		children?: Snippet;
	};

	let { label = 'Compare', labels, width = 'text', children }: Props = $props();

	const id = $props.id();
</script>

<!-- Radio inputs switch the panes with CSS only, so the toggle works without JavaScript. -->
<div class="compare w-{width}" role="group" aria-label={label}>
	<fieldset class="switch">
		<legend class="sr">{label}</legend>
		{#each labels as text, index (index)}
			<label>
				<input type="radio" name="cmp-{id}" value={index} checked={index === 0} />
				<span>{text}</span>
			</label>
		{/each}
	</fieldset>
	<div class="panes">{@render children?.()}</div>
</div>

<style>
	.compare {
		margin: 2.2em 0;
	}

	.switch {
		display: inline-flex;
		gap: 2px;
		margin: 0 0 14px;
		padding: 3px;
		border: 1px solid var(--rule);
		border-radius: 999px;
		background: var(--card);
	}

	.switch label {
		position: relative;
		cursor: pointer;
	}

	.switch input {
		position: absolute;
		inset: 0;
		opacity: 0;
		margin: 0;
		cursor: pointer;
	}

	.switch span {
		display: block;
		padding: 7px 14px;
		border-radius: 999px;
		font: 500 14px/1.2 var(--sans);
		color: var(--ink-2);
		transition:
			background-color 0.2s,
			color 0.2s;
	}

	.switch input:checked + span {
		background: var(--ink);
		color: var(--bg);
	}

	.switch input:focus-visible + span {
		outline: 2px solid var(--accent);
		outline-offset: 2px;
	}

	/* Without :has(), both panes show, each with its own label. */
	@supports selector(:has(*)) {
		.panes {
			display: grid;
		}

		.panes > :global(.pane) {
			grid-area: 1 / 1;
			transition:
				opacity 0.3s ease,
				visibility 0.3s;
		}

		.panes > :global(.pane .pane-label) {
			display: none;
		}

		.compare:has(input[value='0']:checked) .panes > :global(.pane:nth-child(2)),
		.compare:has(input[value='1']:checked) .panes > :global(.pane:nth-child(1)) {
			opacity: 0;
			visibility: hidden;
		}
	}

	@supports not selector(:has(*)) {
		.switch {
			display: none;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.switch span,
		.panes > :global(.pane) {
			transition: none;
		}
	}
</style>
