<script lang="ts">
	import { page } from '$app/state';

	import { absoluteUrl, canonicalUrl, serializeJsonLd } from '#lib/seo.ts';
	import { SITE } from '#lib/site.ts';

	import type { StructuredData } from '#lib/seo.ts';

	type Props = {
		title: string;
		description: string;
		// Social cards can use a shorter title and description than the search result.
		socialTitle?: string;
		socialDescription?: string;
		ogType?: 'website' | 'article';
		pageType?: 'WebPage' | 'CollectionPage';
		mainEntity?: StructuredData;
		// A post can bring its own card. It must be 1200x630 like the site card.
		image?: { path: string; alt: string };
		published?: string;
	};

	let {
		title,
		description,
		socialTitle = title,
		socialDescription = description,
		ogType = 'website',
		pageType = 'WebPage',
		mainEntity,
		image: card = { path: SITE.ogImage.path, alt: SITE.ogImage.alt },
		published
	}: Props = $props();

	const isError = $derived(page.status >= 400);
	const url = $derived(isError ? undefined : canonicalUrl({ path: page.url.pathname }));
	const image = $derived(absoluteUrl({ path: card.path }));
	const siteUrl = canonicalUrl({ path: '/' });

	const website = {
		'@type': 'WebSite',
		'@id': `${siteUrl}#website`,
		url: siteUrl,
		name: SITE.name,
		inLanguage: 'en'
	};

	const jsonLd = $derived({
		'@context': 'https://schema.org',
		'@graph': [
			website,
			{
				'@type': pageType,
				'@id': `${url}#webpage`,
				url,
				name: title,
				description,
				inLanguage: 'en',
				isPartOf: { '@id': website['@id'] },
				primaryImageOfPage: { '@type': 'ImageObject', url: image, caption: card.alt },
				mainEntity
			}
		]
	});
</script>

<svelte:head>
	<title>{title}</title>
	<meta name="description" content={description} />
	<meta name="robots" content={isError ? 'noindex, nofollow' : SITE.robots} />

	{#if url}
		<link rel="canonical" href={url} />
		<link rel="describedby" href={absoluteUrl({ path: '/llms.txt' })} />
		<meta property="og:url" content={url} />
	{/if}

	<meta property="og:type" content={ogType} />
	<meta property="og:site_name" content={SITE.name} />
	<meta property="og:locale" content="en_US" />
	<meta property="og:title" content={socialTitle} />
	<meta property="og:description" content={socialDescription} />
	<meta property="og:image" content={image} />
	<meta property="og:image:width" content={String(SITE.ogImage.width)} />
	<meta property="og:image:height" content={String(SITE.ogImage.height)} />
	<meta property="og:image:alt" content={card.alt} />
	{#if published}
		<meta property="article:published_time" content={published} />
	{/if}

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={socialTitle} />
	<meta name="twitter:description" content={socialDescription} />
	<meta name="twitter:image" content={image} />
	<meta name="twitter:image:alt" content={card.alt} />

	{#if !isError}
		<!-- eslint-disable-next-line svelte/no-at-html-tags -- JSON-LD is serialized and HTML-escaped before insertion. -->
		{@html `<script type="application/ld+json">${serializeJsonLd(jsonLd)}</` + 'script>'}
	{/if}
</svelte:head>
