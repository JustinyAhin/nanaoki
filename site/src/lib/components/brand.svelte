<script lang="ts">
	type Props = { href: string; label?: string };

	let { href, label }: Props = $props();
</script>

<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Callers pass a resolved path or a fragment. -->
<a class="brand" {href} aria-label={label}>
	<svg class="seal" viewBox="0 0 32 32" aria-hidden="true">
		<rect x="2" y="2" width="28" height="28" rx="6" fill="var(--accent)" />
		<path
			d="M9 23V12.5c0-.6.7-.9 1.1-.5L21 23.5c.4.4 1.1.1 1.1-.5V9"
			fill="none"
			stroke="#fff"
			stroke-width="3"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
	</svg>
	nanaoki
</a>

<style>
	.brand {
		display: flex;
		align-items: center;
		gap: 10px;
		text-decoration: none;
		font: 500 17px/1 var(--mono);
		letter-spacing: -0.01em;
	}

	.seal {
		width: 26px;
		height: 26px;
		flex: none;
	}
</style>
