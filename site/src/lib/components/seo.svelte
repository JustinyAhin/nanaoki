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
	};

	let {
		title,
		description,
		socialTitle = title,
		socialDescription = description,
		ogType = 'website',
		pageType = 'WebPage',
		mainEntity
	}: Props = $props();

	const isError = $derived(page.status >= 400);
	const url = $derived(isError ? undefined : canonicalUrl({ path: page.url.pathname }));
	const image = absoluteUrl({ path: SITE.ogImage.path });
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
				primaryImageOfPage: { '@type': 'ImageObject', url: image, caption: SITE.ogImage.alt },
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
	<meta property="og:image:alt" content={SITE.ogImage.alt} />

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={socialTitle} />
	<meta name="twitter:description" content={socialDescription} />
	<meta name="twitter:image" content={image} />
	<meta name="twitter:image:alt" content={SITE.ogImage.alt} />

	{#if !isError}
		<!-- eslint-disable-next-line svelte/no-at-html-tags -- JSON-LD is serialized and HTML-escaped before insertion. -->
		{@html `<script type="application/ld+json">${serializeJsonLd(jsonLd)}</` + 'script>'}
	{/if}
</svelte:head>
