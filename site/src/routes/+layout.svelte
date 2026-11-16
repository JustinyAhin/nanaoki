<script lang="ts">
	import '@fontsource-variable/newsreader/opsz.css';
	import '@fontsource-variable/ibm-plex-sans';
	import '@fontsource/ibm-plex-mono/400.css';
	import '@fontsource/ibm-plex-mono/500.css';

	import '#lib/styles.css';
	import { copyStatus } from '#lib/copy.svelte.ts';

	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

{@render children()}

<p class="sr" role="status">{copyStatus.text}</p>
