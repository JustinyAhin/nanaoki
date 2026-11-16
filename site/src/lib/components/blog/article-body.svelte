<script lang="ts">
	import MarkdocNode from './markdoc-node.svelte';

	import type { RenderNode, TocItem } from '#lib/blog/types.ts';

	type Props = { nodes: RenderNode[]; toc: TocItem[] };

	let { nodes, toc }: Props = $props();

	let active = $state<string | undefined>();

	// Marks the section that is being read in the side contents list.
	const trackSections = (node: HTMLElement) => {
		const headings = toc
			.map((item) => node.querySelector<HTMLElement>(`#${CSS.escape(item.id)}`))
			.filter((heading) => heading !== null);
		if (!headings.length) return;

		const onScroll = () => {
			const line = window.innerHeight * 0.3;
			let current: string | undefined;
			for (const heading of headings) {
				if (heading.getBoundingClientRect().top <= line) current = heading.id;
			}
			active = current;
		};

		let queued = false;
		const schedule = () => {
			if (queued) return;
			queued = true;
			requestAnimationFrame(() => {
				queued = false;
				onScroll();
			});
		};

		onScroll();
		window.addEventListener('scroll', schedule, { passive: true });
		return () => window.removeEventListener('scroll', schedule);
	};
</script>

<div class="post-body">
	<div class={['post-grid', { 'has-toc': toc.length > 1 }]}>
		{#if toc.length > 1}
			<details class="toc-inline">
				<summary>Contents</summary>
				<ol>
					{#each toc as item (item.id)}
						<li class="l{item.level}"><a href="#{item.id}">{item.text}</a></li>
					{/each}
				</ol>
			</details>

			<aside class="toc-side">
				<nav aria-label="Contents">
					<p class="toc-title">Contents</p>
					<ol>
						{#each toc as item (item.id)}
							<li class="l{item.level}">
								<a href="#{item.id}" aria-current={active === item.id ? 'location' : undefined}
									>{item.text}</a
								>
							</li>
						{/each}
					</ol>
				</nav>
			</aside>
		{/if}

		<div class="prose" {@attach trackSections}>
			{#each nodes as node, index (index)}<MarkdocNode {node} />{/each}
		</div>
	</div>
</div>

<style>
	/* The text column is centred on the page. On wide screens the contents list sits in the left
	   margin and sidenotes in the right margin. Breakout widths use the container width (cqi). */
	.post-body {
		container: post / inline-size;
	}

	.post-grid {
		--text: min(640px, 100cqi - 2 * var(--gutter));
		--side: calc((100cqi - var(--text)) / 2);
		--note-gap: 44px;
		--note-w: min(270px, var(--side) - var(--gutter) - var(--note-gap));
		--wide: min(1040px, 100cqi - 2 * var(--gutter));
		position: relative;
	}

	.prose {
		width: var(--text);
		margin-inline: auto;
		font: 400 20px/1.65 var(--serif);
		font-optical-sizing: auto;
		color: var(--ink);
	}

	/* Base prose styles: just above the global element styles, below any component class. */
	:global(html :where(.prose) p) {
		margin: 0 0 1.05em;
	}

	:global(html :where(.prose) :is(h2, h3)) {
		scroll-margin-top: 24px;
		max-width: none;
	}

	:global(html :where(.prose) h2) {
		font-size: clamp(27px, 1.4vw + 19px, 34px);
		line-height: 1.18;
		margin: 2.1em 0 0.6em;
	}

	:global(html :where(.prose) h3) {
		font-size: clamp(21px, 0.6vw + 17px, 24px);
		margin: 1.7em 0 0.45em;
	}

	:global(html :where(.prose) > :first-child) {
		margin-top: 0;
	}

	:global(html :where(.prose) :is(ul, ol):not([class])) {
		margin: 0 0 1.1em;
		padding-left: 1.25em;
	}

	:global(html :where(.prose) li:not([class])) {
		margin: 0.3em 0;
		padding-left: 0.2em;
	}

	:global(html :where(.prose) li::marker) {
		color: var(--ink-3);
	}

	:global(html :where(.prose) strong) {
		font-weight: 600;
	}

	:global(html :where(.prose) a:not([class])) {
		text-decoration-color: color-mix(in srgb, var(--accent) 55%, transparent);
		text-decoration-thickness: 1px;
	}

	:global(html :where(.prose) a:not([class]):hover) {
		text-decoration-color: var(--accent);
	}

	:global(html :where(.prose) :not(pre) > code) {
		font-size: 0.8em;
	}

	:global(html :where(.prose) blockquote:not([class])) {
		margin: 1.6em 0;
		padding-left: 22px;
		border-left: 1px solid var(--ink);
		color: var(--ink-2);
	}

	:global(html :where(.prose) hr) {
		border: 0;
		margin: 2.6em auto;
		width: 64px;
		border-top: 1px solid var(--rule);
	}

	:global(html :where(.prose) img:not([class])) {
		border-radius: 10px;
		margin: 1.8em 0;
	}

	.post-grid .prose :global(:is(.w-wide, .w-full)) {
		clear: both;
	}

	.post-grid .prose :global(.w-wide) {
		width: var(--wide);
		margin-inline: calc((100% - var(--wide)) / 2);
	}

	.post-grid .prose :global(.w-full) {
		width: 100cqi;
		margin-inline: calc((100% - 100cqi) / 2);
	}

	/* With a contents list in the left margin, a wide piece grows to the right only, over the
	   sidenote column. A full-width piece passes over the sticky list on its own ground. */
	@container post (min-width: 1240px) {
		.post-grid.has-toc .prose :global(.w-wide) {
			width: calc(var(--text) + var(--note-gap) + var(--note-w));
			margin-inline: 0;
		}

		.post-grid .prose :global(.w-full) {
			position: relative;
			z-index: 1;
			background: var(--bg);
		}
	}

	/* contents */
	.toc-inline {
		width: var(--text);
		margin: 0 auto 2.2em;
		border-block: 1px solid var(--rule);
		font: 400 15px/1.45 var(--sans);
	}

	.toc-inline summary {
		padding: 12px 0;
		cursor: pointer;
		font: 500 12.5px/1.2 var(--mono);
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--ink-3);
	}

	.toc-inline ol {
		margin: 0;
		padding: 0 0 14px;
		list-style: none;
	}

	ol {
		list-style: none;
	}

	li.l3 {
		padding-left: 14px;
	}

	.toc-inline li,
	.toc-side li {
		margin: 0;
	}

	.toc-inline a {
		display: block;
		padding: 5px 0;
		color: var(--ink-2);
		text-decoration: none;
	}

	.toc-side {
		display: none;
	}

	@container post (min-width: 1240px) {
		.toc-inline {
			display: none;
		}

		.toc-side {
			display: block;
			position: absolute;
			top: 0;
			bottom: 0;
			left: calc(var(--side) - var(--note-gap) - var(--note-w));
			width: var(--note-w);
		}

		.toc-side nav {
			position: sticky;
			top: 32px;
			max-height: calc(100vh - 64px);
			overflow-y: auto;
			scrollbar-width: none;
		}

		.toc-title {
			margin: 0 0 10px;
			font: 500 12.5px/1.2 var(--mono);
			letter-spacing: 0.06em;
			text-transform: uppercase;
			color: var(--ink-3);
		}

		.toc-side ol {
			margin: 0;
			padding: 0;
			font: 400 14px/1.4 var(--sans);
		}

		.toc-side a {
			display: block;
			padding: 5px 0 5px 12px;
			border-left: 1px solid var(--rule);
			color: var(--ink-3);
			text-decoration: none;
			text-wrap: pretty;
			transition:
				color 0.2s,
				border-color 0.2s;
		}

		.toc-side a:hover {
			color: var(--ink);
		}

		.toc-side a[aria-current] {
			color: var(--ink);
			border-left-color: var(--ink);
		}
	}

	@media (max-width: 720px) {
		.prose {
			font-size: 18.5px;
			line-height: 1.6;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.toc-side a {
			transition: none;
		}
	}
</style>
