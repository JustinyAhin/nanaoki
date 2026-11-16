<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import { SITE } from '#lib/site.ts';
	import Brand from './brand.svelte';
	import GithubMark from './github-mark.svelte';

	type Props = { current?: 'home' | 'cases' | 'blog' };

	let { current }: Props = $props();

	const home = resolve('/');
	const onHome = $derived(current === 'home');
</script>

<header class="wrap nav">
	<Brand href={onHome ? '#top' : home} label="nanaoki home" />

	<nav class="nav-links" aria-label="Main">
		<!-- eslint-disable svelte/no-navigation-without-resolve -- Fragments on the resolved home path. -->
		<a class="hide-sm" href={onHome ? '#how' : `${home}#how`}>How it works</a>
		<a href={resolve('/cases')} aria-current={current === 'cases' ? 'page' : undefined}>Cases</a>
		{#if page.data.hasBlog}
			<a href={resolve('/blog')} aria-current={current === 'blog' ? 'page' : undefined}>Blog</a>
		{/if}
		<a class="hide-sm" href={onHome ? '#install' : `${home}#install`}>Install</a>
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{#if onHome}
			<a class="hide-sm" href="#faq">FAQ</a>
		{/if}
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- External URL. -->
		<a class="gh" href={SITE.repo}>
			<GithubMark />
			GitHub
		</a>
	</nav>
</header>

<style>
	.nav {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 68px;
	}

	.nav-links {
		display: flex;
		gap: 22px;
		align-items: center;
		font-size: 15px;
	}

	.nav-links a {
		text-decoration: none;
		color: var(--ink-2);
	}

	.nav-links a:hover,
	.nav-links a[aria-current] {
		color: var(--ink);
	}

	.nav-links a[aria-current] {
		text-decoration: underline;
		text-decoration-color: var(--accent);
		text-decoration-thickness: 2px;
		text-underline-offset: 6px;
	}

	.gh {
		display: inline-flex;
		align-items: center;
		gap: 7px;
	}

	.gh :global(svg) {
		width: 17px;
		height: 17px;
	}

	@media (max-width: 720px) {
		.hide-sm {
			display: none;
		}
	}
</style>
