<script lang="ts">
	import Faq from '#lib/components/home/faq.svelte';
	import Hero from '#lib/components/home/hero.svelte';
	import How from '#lib/components/home/how.svelte';
	import InstallSection from '#lib/components/home/install-section.svelte';
	import Pain from '#lib/components/home/pain.svelte';
	import Privacy from '#lib/components/home/privacy.svelte';
	import Why from '#lib/components/home/why.svelte';
	import Seo from '#lib/components/seo.svelte';
	import SiteFooter from '#lib/components/site-footer.svelte';
	import SiteHeader from '#lib/components/site-header.svelte';
	import { SITE } from '#lib/site.ts';

	const software = {
		'@type': 'SoftwareApplication',
		name: SITE.name,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'macOS, Linux, Windows',
		license: SITE.license,
		downloadUrl: SITE.releases,
		codeRepository: SITE.repo,
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
		author: { '@type': 'Person', name: SITE.author.name, url: SITE.author.url }
	};
</script>

<Seo
	title="nanaoki: your coding agent already fixed this once"
	description="Your coding agent keeps making the same mistake. nanaoki reads the Claude Code and Codex history on your machine, finds the fix that worked last time, and shows it when the error comes back. Runs locally with no LLM and no network calls. Free, MIT."
	socialTitle="nanaoki: your agent already fixed this once"
	socialDescription="Your coding agent keeps making the same mistake. nanaoki finds the fix it already wrote, often in another repo, and shows it when the error comes back. Runs locally with no LLM and no network calls."
	mainEntity={software}
/>

<SiteHeader current="home" />

<main id="top">
	<Hero />
	<Pain />
	<Why />
	<How />
	<InstallSection />
	<Privacy />
	<Faq />
</main>

<SiteFooter current="home" />
