<script lang="ts">
	import { resolve } from '$app/paths';

	import { formatDate } from '#lib/blog/format.ts';
	import ArticleBody from '#lib/components/blog/article-body.svelte';
	import ReadingProgress from '#lib/components/blog/reading-progress.svelte';
	import Seo from '#lib/components/seo.svelte';
	import SiteFooter from '#lib/components/site-footer.svelte';
	import SiteHeader from '#lib/components/site-header.svelte';
	import { absoluteUrl, canonicalUrl } from '#lib/seo.ts';
	import { SITE } from '#lib/site.ts';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const meta = $derived(data.meta);
	const image = $derived(meta.og ? { path: meta.og, alt: meta.title } : undefined);
	const posting = $derived({
		'@type': 'BlogPosting',
		headline: meta.title,
		description: meta.description,
		datePublished: meta.date,
		url: canonicalUrl({ path: `/blog/${meta.slug}/` }),
		image: absoluteUrl({ path: image?.path ?? SITE.ogImage.path }),
		keywords: meta.tags.join(', ') || undefined,
		author: { '@type': 'Person', name: SITE.author.name, url: SITE.author.url }
	});
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
	title="{meta.title} · nanaoki"
	description={meta.description}
	socialTitle={meta.title}
	ogType="article"
	published={meta.date}
	{image}
	mainEntity={posting}
/>

<SiteHeader current="blog" />

<main id="top">
	<article class="post">
		<ReadingProgress />

		<header class="head">
			<p class="meta">
				<a href={resolve('/blog')}>Blog</a>
				<time datetime={meta.date}>{formatDate(meta.date)}</time>
				<span>{meta.minutes} min read</span>
				{#if meta.draft}<span class="draft">Draft, dev only</span>{/if}
			</p>
			<h1>{meta.title}</h1>
			<p class="dek">{meta.description}</p>
		</header>

		{#if data.errors.length}
			<div class="errors" role="alert">
				<p>This post would fail the production build:</p>
				<ul>
					{#each data.errors as message (message)}<li><code>{message}</code></li>{/each}
				</ul>
			</div>
		{/if}

		<ArticleBody nodes={data.nodes} toc={data.toc} />

		<footer class="foot">
			<p>
				Written by
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- External URL. -->
				<a href={SITE.author.url}>{SITE.author.name}</a>.
			</p>
			<p class="links">
				<a href={resolve('/blog')}>All posts</a>
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- A feed file, not a page. -->
				<a href="/blog/rss.xml" data-sveltekit-reload>RSS feed</a>
			</p>
		</footer>
	</article>
</main>

<SiteFooter current="blog" />

<style>
	.post {
		view-timeline-name: --article;
		padding-bottom: 88px;
	}

	.head,
	.errors,
	.foot {
		width: min(640px, 100% - 2 * var(--gutter));
		margin-inline: auto;
	}

	.head {
		padding: 56px 0 36px;
		margin-bottom: 40px;
		border-bottom: 1px solid var(--rule);
	}

	.meta {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 16px;
		margin: 0 0 22px;
		font: 400 13.5px/1.4 var(--mono);
		color: var(--ink-3);
	}

	.meta a {
		color: var(--accent-ink);
		text-decoration: none;
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.draft {
		color: var(--accent-ink);
	}

	h1 {
		width: min(820px, 100vw - 2 * var(--gutter));
		font-size: clamp(36px, 3vw + 18px, 60px);
		line-height: 1.06;
		letter-spacing: -0.018em;
	}

	.dek {
		margin: 22px 0 0;
		font: 400 clamp(19px, 0.5vw + 16px, 22px) / 1.5 var(--serif);
		color: var(--ink-2);
	}

	.errors {
		margin-bottom: 32px;
		padding: 14px 18px;
		border: 1px dashed var(--accent);
		border-radius: 10px;
		font-size: 14px;
		color: var(--accent-ink);
	}

	.errors p {
		margin: 0 0 6px;
	}

	.errors ul {
		margin: 0;
		padding-left: 18px;
	}

	.foot {
		margin-top: 64px;
		padding-top: 20px;
		border-top: 1px solid var(--ink);
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 8px 24px;
		font-size: 15px;
		color: var(--ink-3);
	}

	.foot p {
		margin: 0;
	}

	.links {
		display: flex;
		gap: 18px;
	}

	@media (max-width: 720px) {
		.head {
			padding-top: 24px;
			margin-bottom: 28px;
		}

		.post {
			padding-bottom: 64px;
		}
	}
</style>
