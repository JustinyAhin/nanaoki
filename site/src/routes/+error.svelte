<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import Seo from '#lib/components/seo.svelte';
	import SiteFooter from '#lib/components/site-footer.svelte';
	import SiteHeader from '#lib/components/site-header.svelte';
</script>

<Seo title="Page not found · nanaoki" description="This page does not exist on nanaoki.com." />

<SiteHeader />

<main id="top" class="wrap missing">
	<p class="eyebrow">{page.status}</p>
	<h1>{page.status === 404 ? 'This page does not exist.' : 'Something went wrong.'}</h1>
	<p class="lead">
		<a href={resolve('/')}>Go to the home page</a>
		or
		<a href={resolve('/cases')}>see the cases</a>.
	</p>
</main>

<SiteFooter />

<style>
	.missing {
		padding-block: 96px 120px;
	}

	h1 {
		font-size: clamp(34px, 3.2vw + 14px, 58px);
		line-height: 1.06;
	}
</style>
