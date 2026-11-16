<script lang="ts">
	import { COMPONENTS } from './registry.ts';
	import Self from './markdoc-node.svelte';

	import type { RenderNode } from '#lib/blog/types.ts';

	type Props = { node: RenderNode };

	let { node }: Props = $props();

	const VOID = new Set(['br', 'hr', 'img', 'wbr', 'col', 'source']);
</script>

<!-- Keep each branch on one line: whitespace between inline nodes would render as spaces. -->
<!-- prettier-ignore -->
{#if typeof node === 'string' || typeof node === 'number'}{node}{:else if node && typeof node === 'object'}{@const Component = COMPONENTS[node.name]}{#if Component && node.children.length}<Component {...node.attributes}>{#each node.children as child, index (index)}<Self node={child} />{/each}</Component>{:else if Component}<Component {...node.attributes} />{:else if VOID.has(node.name)}<svelte:element this={node.name} {...node.attributes} />{:else}<svelte:element this={node.name} {...node.attributes}>{#each node.children as child, index (index)}<Self node={child} />{/each}</svelte:element>{/if}{/if}
