<script lang="ts">
	import Terminal from '../terminal.svelte';

	type Props = { title?: string; label?: string; text: string; width?: 'text' | 'wide' | 'full' };

	let { title = 'terminal', label, text, width = 'text' }: Props = $props();

	// Line prefixes: "$ " a command, "> " a prompt, "! " an error (the "!" is hidden), "# " a comment,
	// "● " a tool call and "└ " a tool result, as Claude Code prints them.
	const lines = $derived(
		text.split('\n').map((line) => {
			if (line.startsWith('$ ')) return { mark: '$', cls: 'c', body: line.slice(2) };
			if (line.startsWith('> ')) return { mark: '>', cls: 'h', body: line.slice(2) };
			if (line.startsWith('! ')) return { mark: '', cls: 'r', body: line.slice(2) };
			if (line.startsWith('# ')) return { mark: '', cls: 'd', body: line };
			if (line.startsWith('● ')) return { mark: '●', cls: 'g-mark', body: line.slice(2) };
			if (line.startsWith('└ ')) return { mark: '└', cls: '', body: line.slice(2) };
			return { mark: '', cls: '', body: line };
		})
	);
</script>

<div class="terminal w-{width}">
	<Terminal {title} label={label ?? title}>
		<!-- The body keeps whitespace, so it must stay on one line. -->
		<!-- prettier-ignore -->
		{#each lines as line, index (index)}{#if index}{'\n'}{/if}{#if line.mark}<span class={line.cls === 'g-mark' ? 'g' : 'd'}>{line.mark}</span>{' '}{/if}<span class={line.cls === 'g-mark' ? '' : line.cls}>{line.body}</span>{/each}
	</Terminal>
</div>

<style>
	.terminal {
		margin: 1.8em 0;
	}
</style>
