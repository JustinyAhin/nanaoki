<script lang="ts">
	import { resolve } from '$app/paths';

	import { SITE } from '#lib/site.ts';
	import Brand from './brand.svelte';

	type Props = { current?: 'home' | 'cases' };

	let { current }: Props = $props();

	let inView = $state(false);

	// Draw the gold line once, when the footer comes into view.
	const revealOnScroll = (node: HTMLElement) => {
		if (!('IntersectionObserver' in window)) {
			inView = true;
			return;
		}

		const observer = new IntersectionObserver(
			(entries) => {
				if (!entries[0].isIntersecting) return;
				inView = true;
				observer.disconnect();
			},
			{ threshold: 0.35 }
		);

		observer.observe(node);
		return () => observer.disconnect();
	};

	// Seven pale lines fall away. Each line starts 9 units higher and 78 units further right.
	const falls = Array.from({ length: 7 }, (_, index) => ({
		y: 435 - index * 9,
		x: 150 + index * 78,
		opacity: [0.14, 0.19, 0.23, 0.28, 0.33, 0.37, 0.42][index]
	}));
</script>

<footer>
	<div class="colophon" class:in={inView} {@attach revealOnScroll}>
		<svg class="art art-fall" viewBox="0 0 1000 600" preserveAspectRatio="none" aria-hidden="true">
			<defs>
				<linearGradient id="fall" gradientUnits="userSpaceOnUse" x1="0" y1="381" x2="0" y2="600">
					<stop offset="0" style="stop-color:var(--ink)" />
					<stop offset=".25" style="stop-color:var(--ink);stop-opacity:.66" />
					<stop offset="1" style="stop-color:var(--ink);stop-opacity:0" />
				</linearGradient>
				<linearGradient id="gold" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="1000" y2="0">
					<stop offset="0" style="stop-color:var(--gold-1)" />
					<stop offset=".45" style="stop-color:var(--gold-2)" />
					<stop offset=".7" style="stop-color:var(--gold-3)" />
					<stop offset=".85" style="stop-color:var(--gold-2)" />
					<stop offset="1" style="stop-color:var(--gold-3)" />
				</linearGradient>
			</defs>
			<g>
				{#each falls as fall (fall.y)}
					<path stroke-opacity={fall.opacity} d="M-10 {fall.y}H{fall.x}c90 0 150 110 200 260" />
				{/each}
			</g>
		</svg>
		<svg class="art art-rise" viewBox="0 0 1000 600" preserveAspectRatio="none" aria-hidden="true">
			<path
				d="M-10 372H696c139 0 232-162 309-382"
				stroke="var(--gold-2)"
				stroke-opacity=".16"
				stroke-width="7"
				stroke-linecap="round"
			/>
			<path
				d="M-10 372H696c139 0 232-162 309-382"
				stroke="url(#gold)"
				stroke-width="2.4"
				stroke-linecap="round"
			/>
		</svg>

		<div class="wrap colophon-in">
			<div>
				<Brand href={current === 'home' ? '#top' : resolve('/')} />
				<p class="motto">Fall seven times,<br />stand up eight.</p>
			</div>
			<div class="proverb">
				<p class="ja" lang="ja">七転び八起き</p>
				<div class="side">
					<p class="romaji">Nana korobi ya oki</p>
					<!-- The seal reads top to bottom, right column first: 七転 then 八起. -->
					<svg class="hanko" viewBox="0 0 46 46" aria-hidden="true">
						<rect x="1" y="1" width="44" height="44" rx="4" fill="var(--accent)" />
						<text x="32" y="20">七</text>
						<text x="32" y="39">転</text>
						<text x="14" y="20">八</text>
						<text x="14" y="39">起</text>
					</svg>
				</div>
			</div>
		</div>
	</div>

	<div class="wrap foot">
		<nav aria-label="Footer">
			<a href={resolve('/cases')}>Cases</a>
			<!-- eslint-disable svelte/no-navigation-without-resolve -- External URLs. -->
			<a href={SITE.repo}>GitHub</a>
			<a href={SITE.releases}>Releases</a>
			<a href={SITE.license}>MIT license</a>
		</nav>
		<p class="by">
			Made by <a href={SITE.author.url}>{SITE.author.name}</a>
		</p>
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
	</div>
</footer>

<style>
	footer {
		--gold-1: #8a6516;
		--gold-2: #b8892a;
		--gold-3: #d2a84a;
		border-top: 1px solid var(--rule);
		background: var(--bg-2);
		color: var(--ink-3);
		font-size: 15px;
	}

	@media (prefers-color-scheme: dark) {
		footer {
			--gold-1: #8d6a22;
			--gold-2: #cfa54a;
			--gold-3: #f2dc9c;
		}
	}

	/* art */
	.colophon {
		position: relative;
		overflow: hidden;
	}

	.art {
		position: absolute;
		top: 0;
		bottom: 0;
		left: 0;
		height: 100%;
		right: calc(max((100% - var(--max)) / 2, 0px) + var(--gutter) + 176px);
		width: calc(100% - max((100% - var(--max)) / 2, 0px) - var(--gutter) - 176px);
		pointer-events: none;
	}

	.art path {
		fill: none;
		vector-effect: non-scaling-stroke;
	}

	.art-fall path {
		stroke: url(#fall);
		stroke-width: 1.25;
	}

	.art-rise {
		transition: clip-path 2.6s cubic-bezier(0.3, 0.1, 0.2, 1);
		clip-path: inset(0 0 0 0);
	}

	/* Without JavaScript, the line shows at once. */
	:global(html.js) .colophon:not(.in) .art-rise {
		clip-path: inset(0 100% 0 0);
	}

	/* colophon */
	.colophon-in {
		position: relative;
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 32px;
		min-height: 600px;
		padding-top: 88px;
		padding-bottom: 72px;
	}

	.motto {
		font: italic 400 26px/1.3 var(--serif);
		color: var(--ink-2);
		margin: 26px 0 0;
	}

	.proverb {
		display: flex;
		flex-direction: row-reverse;
		align-items: stretch;
		gap: 18px;
		flex: none;
	}

	.ja {
		writing-mode: vertical-rl;
		font: 500 84px/1 var(--mincho);
		letter-spacing: 0.06em;
		color: var(--ink);
		margin: 0;
		font-feature-settings: 'vkna';
	}

	.side {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		align-items: center;
		padding: 6px 0 4px;
	}

	.romaji {
		writing-mode: vertical-rl;
		font: 400 12px/1 var(--mono);
		letter-spacing: 0.32em;
		text-transform: uppercase;
		color: var(--ink-3);
		margin: 0;
		white-space: nowrap;
	}

	.hanko {
		width: 46px;
		height: 46px;
		display: block;
		transform: rotate(-2deg);
	}

	.hanko text {
		font: 500 17px var(--mincho);
		fill: #fff;
		text-anchor: middle;
	}

	/* links */
	.foot {
		display: flex;
		justify-content: space-between;
		gap: 16px 24px;
		flex-wrap: wrap;
		align-items: center;
		position: relative;
		padding-top: 28px;
		padding-bottom: 36px;
	}

	.foot::before {
		content: '';
		position: absolute;
		left: var(--gutter);
		right: var(--gutter);
		top: 0;
		border-top: 1px solid var(--rule);
	}

	.foot nav {
		display: flex;
		gap: 6px 22px;
		flex-wrap: wrap;
	}

	.foot nav a {
		text-decoration: none;
		color: var(--ink-2);
		padding: 6px 0;
	}

	.foot nav a:hover {
		color: var(--ink);
	}

	.by {
		margin: 0;
	}

	@media (max-width: 720px) {
		.art {
			right: calc(var(--gutter) + 124px);
			width: calc(100% - var(--gutter) - 124px);
		}

		.colophon-in {
			min-height: 0;
			padding-top: 56px;
			padding-bottom: 56px;
			gap: 16px;
		}

		.motto {
			font-size: 18px;
			margin-top: 18px;
		}

		.proverb {
			gap: 12px;
		}

		.ja {
			font-size: 58px;
		}

		.romaji {
			font-size: 10.5px;
			letter-spacing: 0.26em;
		}

		.hanko {
			width: 40px;
			height: 40px;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.art-rise {
			transition: none;
		}

		:global(html.js) .colophon:not(.in) .art-rise {
			clip-path: none;
		}
	}
</style>
