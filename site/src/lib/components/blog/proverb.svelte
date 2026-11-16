<script lang="ts">
	import { whenSeen } from '#lib/blog/in-view.ts';
	import Seal from '../seal.svelte';

	import type { ViewState } from '#lib/blog/in-view.ts';

	const TEXT = '七転び八起き';
	const chars = [...TEXT];

	let view = $state<ViewState>('idle');
</script>

<figure
	class="proverb {view}"
	aria-label="The proverb nana korobi ya oki, written top to bottom in Japanese"
	{@attach whenSeen({ onChange: (next) => (view = next), threshold: 0.35 })}
>
	<p class="ja" lang="ja">
		<span class="sr">{TEXT}</span>
		<!-- prettier-ignore -->
		<span aria-hidden="true">{#each chars as char, index (index)}<span class="char" style:--i={index}>{char}</span>{/each}</span>
	</p>
	<figcaption style:--i={chars.length}>
		<p class="reading">nana korobi ya oki</p>
		<p class="gloss">Seven falls, eight getting up.</p>
		<div class="seal"><Seal /></div>
	</figcaption>
</figure>

<style>
	.proverb {
		--step: 0.24s;
		display: flex;
		justify-content: center;
		align-items: stretch;
		gap: 40px;
		margin: 2.6em 0;
	}

	.ja {
		writing-mode: vertical-rl;
		font: 500 68px/1 var(--mincho);
		letter-spacing: 0.06em;
		font-feature-settings: 'vkna';
		color: var(--ink);
		margin: 0;
		flex: none;
	}

	.char {
		display: inline-block;
	}

	/* The ink soaks in: each character starts faint and soft, one after the other.
	   Only the seen state has transitions, so the arming at hydration is instant. */
	.seen .char {
		transition:
			opacity 0.9s ease,
			filter 1.1s ease,
			transform 1.1s cubic-bezier(0.2, 0.7, 0.2, 1);
		transition-delay: calc(var(--i) * var(--step));
	}

	.armed .char {
		opacity: 0;
		filter: blur(7px);
		transform: translateY(-0.12em);
	}

	figcaption {
		display: flex;
		flex-direction: column;
		justify-content: flex-end;
		align-items: flex-start;
		padding: 0 0 6px 40px;
		border-left: 1px solid var(--rule);
	}

	.seen figcaption {
		transition: opacity 0.9s ease;
		transition-delay: calc(var(--i) * var(--step));
	}

	.armed figcaption {
		opacity: 0;
	}

	.reading {
		margin: 0;
		font: 400 13px/1.4 var(--mono);
		letter-spacing: 0.24em;
		text-transform: uppercase;
		color: var(--ink-3);
	}

	.gloss {
		margin: 10px 0 0;
		font: 400 24px/1.25 var(--serif);
		color: var(--ink);
		max-width: 9em;
		text-wrap: balance;
	}

	.seal {
		--seal-size: 46px;
		margin-top: 28px;
	}

	.seen .seal {
		transition:
			opacity 0.35s ease,
			transform 0.45s cubic-bezier(0.2, 0.9, 0.3, 1);
		transition-delay: calc((var(--i) + 3) * var(--step));
	}

	.armed .seal {
		opacity: 0;
		transform: scale(1.18);
	}

	@media (max-width: 560px) {
		.proverb {
			flex-direction: column;
			align-items: center;
			gap: 22px;
		}

		.ja {
			font-size: 54px;
		}

		figcaption {
			align-items: center;
			padding: 22px 0 0;
			border-left: 0;
			border-top: 1px solid var(--rule);
			text-align: center;
			width: min(100%, 260px);
		}

		.gloss {
			font-size: 21px;
			max-width: none;
		}

		.seal {
			--seal-size: 40px;
			margin-top: 20px;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.seen .char,
		.seen figcaption,
		.seen .seal {
			transition: none;
		}
	}
</style>
