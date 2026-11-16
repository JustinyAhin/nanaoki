<script lang="ts">
	import { resolve } from '$app/paths';

	import { formatDate } from '#lib/blog/format.ts';
	import Seo from '#lib/components/seo.svelte';
	import SiteFooter from '#lib/components/site-footer.svelte';
	import SiteHeader from '#lib/components/site-header.svelte';
	import { absoluteUrl } from '#lib/seo.ts';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<link
		rel="alternate"
		type="application/rss+xml"
		title="nanaoki blog"
		href={absoluteUrl({ path: '/blog/rss.xml' })}
	/>
</svelte:head>

<Seo
	title="Blog · nanaoki"
	description="Notes from building nanaoki: what my own Claude Code and Codex history shows about agents that repeat a fix, and what changed in the tool because of it."
	pageType="CollectionPage"
/>

<SiteHeader current="blog" />

<main id="top">
	<div class="wrap intro">
		<p class="eyebrow">Blog</p>
		<h1>Notes from building nanaoki.</h1>
	</div>

	<div class="wrap">
		{#if data.posts.length}
			<ol class="posts">
				{#each data.posts as post (post.slug)}
					<li>
						<article class="post">
							<p class="meta">
								<time datetime={post.date}>{formatDate(post.date)}</time>
								<span>{post.minutes} min read</span>
								{#if post.draft}<span class="draft">Draft</span>{/if}
							</p>
							<h2>
								<a href={resolve('/blog/[slug]', { slug: post.slug })}>{post.title}</a>
							</h2>
							<p class="description">{post.description}</p>
						</article>
					</li>
				{/each}
			</ol>
		{:else}
			<p class="empty">No posts yet.</p>
		{/if}

		<p class="feed">
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- A feed file, not a page. -->
			<a href="/blog/rss.xml" data-sveltekit-reload>RSS feed</a>
		</p>
	</div>
</main>

<SiteFooter current="blog" />

<style>
	.intro {
		padding-block: 56px 40px;
	}

	h1 {
		font-size: clamp(34px, 3.2vw + 14px, 58px);
		line-height: 1.06;
		letter-spacing: -0.015em;
		max-width: 18ch;
	}

	.posts {
		list-style: none;
		margin: 0;
		padding: 0;
		border-top: 1px solid var(--ink);
	}

	.posts li {
		border-bottom: 1px solid var(--rule);
	}

	.post {
		display: grid;
		grid-template-columns: 200px minmax(0, 1fr);
		grid-template-areas:
			'meta title'
			'meta text';
		gap: 0 40px;
		padding: 32px 0 34px;
	}

	.meta {
		grid-area: meta;
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin: 6px 0 0;
		font: 400 13.5px/1.3 var(--mono);
		color: var(--ink-3);
	}

	.draft {
		color: var(--accent-ink);
	}

	h2 {
		grid-area: title;
		font-size: clamp(26px, 1.6vw + 18px, 34px);
		line-height: 1.15;
		max-width: 26ch;
	}

	h2 a {
		text-decoration: none;
	}

	h2 a:hover {
		text-decoration: underline;
		text-decoration-thickness: 1px;
		text-underline-offset: 5px;
	}

	.description {
		grid-area: text;
		margin: 12px 0 0;
		max-width: 62ch;
		color: var(--ink-2);
	}

	.empty {
		padding: 32px 0;
		border-top: 1px solid var(--ink);
		color: var(--ink-3);
	}

	.feed {
		margin: 28px 0 96px;
		font: 400 14px/1 var(--mono);
	}

	.feed a {
		color: var(--ink-3);
	}

	@media (max-width: 720px) {
		.intro {
			padding-block: 24px 28px;
		}

		.post {
			grid-template-columns: minmax(0, 1fr);
			grid-template-areas: 'meta' 'title' 'text';
			padding: 24px 0 26px;
		}

		.meta {
			flex-direction: row;
			gap: 14px;
			margin: 0 0 10px;
		}

		.feed {
			margin-bottom: 64px;
		}
	}
</style>
