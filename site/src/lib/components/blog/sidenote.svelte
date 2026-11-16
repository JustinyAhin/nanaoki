<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = { n: number; children?: Snippet };

	let { n, children }: Props = $props();

	const id = $props.id();
</script>

<!-- Inline elements only: the note sits inside a paragraph. On phones the number toggles the note
     with a checkbox, so it works without JavaScript. On wide screens the note floats into the margin. -->
<!-- prettier-ignore -->
<span class="sn"><label class="ref" for="sn-{id}"><span class="sr">Note </span>{n}</label><input class="toggle" type="checkbox" id="sn-{id}" /><span class="note" role="note"><span class="n" aria-hidden="true">{n}</span>{' '}{@render children?.()}</span></span>

<style>
	.ref {
		font: 500 0.62em/1 var(--mono);
		vertical-align: super;
		color: var(--accent-ink);
		padding: 0 0.15em 0 0.1em;
		cursor: pointer;
	}

	.toggle {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
		pointer-events: none;
	}

	.sn:has(.toggle:focus-visible) .ref {
		outline: 2px solid var(--accent);
		outline-offset: 2px;
		border-radius: 3px;
	}

	.note {
		display: none;
		font: 400 14.5px/1.5 var(--sans);
		color: var(--ink-2);
		text-wrap: pretty;
	}

	.toggle:checked + .note {
		display: block;
		margin: 10px 0 12px;
		padding: 10px 14px;
		background: var(--card);
		border-left: 2px solid var(--rule);
		border-radius: 2px 6px 6px 2px;
	}

	.n {
		font: 500 12px/1 var(--mono);
		color: var(--accent-ink);
	}

	@container post (min-width: 1240px) {
		.ref {
			cursor: default;
		}

		.toggle {
			display: none;
		}

		.note,
		.toggle:checked + .note {
			display: block;
			float: right;
			clear: right;
			width: var(--note-w);
			margin: 0.35em calc(-1 * (var(--note-w) + var(--note-gap))) 1em 0;
			padding: 0;
			background: none;
			border: 0;
			font-size: 13.5px;
			color: var(--ink-3);
		}
	}
</style>
