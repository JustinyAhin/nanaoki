<script lang="ts">
	import { ART } from '#lib/blog/art/index.ts';

	import type { ArtColors } from '#lib/blog/art/index.ts';

	type Props = {
		name: string;
		height?: number | string;
		caption?: string;
		seed?: number;
		width?: 'text' | 'wide' | 'full';
	};

	let { name, height = 320, caption, seed = 1, width = 'text' }: Props = $props();

	const piece = $derived(ART[name]);
	const px = $derived(Number(height) || 320);

	// Runs the piece only while it is on screen. Reduced motion gets one still frame.
	const run = (canvas: HTMLCanvasElement) => {
		const art = piece;
		const ctx = canvas.getContext('2d');
		if (!art || !ctx) return;

		const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
		const scheme = window.matchMedia('(prefers-color-scheme: dark)');
		let colors: ArtColors;
		let size = { width: 0, height: 0 };
		let time = 0;
		let last = 0;
		let frame = 0;
		let visible = false;

		const readColors = () => {
			const style = getComputedStyle(canvas);
			const token = (key: string) => style.getPropertyValue(key).trim();
			colors = {
				ink: token('--ink'),
				gold1: token('--gold-1'),
				gold2: token('--gold-2'),
				gold3: token('--gold-3'),
				dark: scheme.matches
			};
		};

		const paint = () => {
			if (!size.width || !size.height) return;
			const ratio = Math.min(window.devicePixelRatio || 1, 2);
			ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
			ctx.clearRect(0, 0, size.width, size.height);
			art.draw({
				ctx,
				width: size.width,
				height: size.height,
				time: motion.matches ? art.still : time,
				seed,
				colors
			});
		};

		const tick = (now: number) => {
			time += Math.min(0.05, (now - last) / 1000);
			last = now;
			paint();
			frame = requestAnimationFrame(tick);
		};

		const start = () => {
			cancelAnimationFrame(frame);
			if (!visible || motion.matches) return paint();
			last = performance.now();
			frame = requestAnimationFrame(tick);
		};

		const resize = new ResizeObserver(([entry]) => {
			const ratio = Math.min(window.devicePixelRatio || 1, 2);
			size = { width: entry.contentRect.width, height: entry.contentRect.height };
			canvas.width = Math.round(size.width * ratio);
			canvas.height = Math.round(size.height * ratio);
			paint();
		});

		const view = new IntersectionObserver(([entry]) => {
			visible = entry.isIntersecting;
			start();
		});

		const restyle = () => {
			readColors();
			paint();
		};

		readColors();
		resize.observe(canvas);
		view.observe(canvas);
		motion.addEventListener('change', start);
		scheme.addEventListener('change', restyle);

		return () => {
			cancelAnimationFrame(frame);
			resize.disconnect();
			view.disconnect();
			motion.removeEventListener('change', start);
			scheme.removeEventListener('change', restyle);
		};
	};
</script>

<figure class="art w-{width}">
	<div class="stage" style:--h="{px}px" role={piece ? 'img' : undefined} aria-label={piece?.label}>
		{#if piece}
			<canvas {@attach run} aria-hidden="true"></canvas>
		{:else}
			<p class="missing">Art piece <code>{name}</code> is not in the registry yet.</p>
		{/if}
	</div>
	{#if caption}<figcaption>{caption}</figcaption>{/if}
</figure>

<style>
	.art {
		--gold-1: #8a6516;
		--gold-2: #b8892a;
		--gold-3: #d2a84a;
		margin: 2.4em 0;
	}

	@media (prefers-color-scheme: dark) {
		.art {
			--gold-1: #8d6a22;
			--gold-2: #cfa54a;
			--gold-3: #f2dc9c;
		}
	}

	.stage {
		position: relative;
		height: min(var(--h), 75vw);
		overflow: hidden;
	}

	canvas {
		display: block;
		width: 100%;
		height: 100%;
	}

	.missing {
		display: grid;
		place-items: center;
		height: 100%;
		margin: 0;
		border: 1px dashed var(--rule);
		border-radius: 10px;
		font: 400 14px/1.4 var(--mono);
		color: var(--ink-3);
		text-align: center;
		padding: 16px;
	}

	figcaption {
		margin-top: 10px;
		font: 400 14px/1.5 var(--sans);
		color: var(--ink-3);
		text-wrap: pretty;
	}

	.w-full figcaption {
		width: min(var(--text), 100% - 2 * var(--gutter));
		margin-inline: auto;
	}
</style>
