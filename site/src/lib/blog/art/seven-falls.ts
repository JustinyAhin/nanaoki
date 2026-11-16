import { bend, bundleGap, cut, easeOut, goldGradient, strokeGold, trace } from './lines.ts';
import { rgb, rng } from './random.ts';

import type { ArtFrame, ArtPiece } from './index.ts';
import type { Point } from './lines.ts';

const LINES = 8;
const FALL_START = 0.3;
const FALL_STAGGER = 0.3;
const FALL_TIME = 0.75;
const RISE_START = FALL_START + (LINES - 2) * FALL_STAGGER + FALL_TIME * 0.7;
const RISE_TIME = 1.5;
const RISE_END = RISE_START + RISE_TIME;

// The social card (og/v2-ab.html) with no type: seven lines fall away in turn, the eighth rises in gold.
// Seed 1 is the geometry of the card. Other seeds move the falls a little.
const draw = ({ ctx, width: w, height: h, time, seed, colors }: ArtFrame) => {
	const random = seed === 1 ? () => 0.5 : rng(seed);
	const ink = rgb(colors.ink);
	const gap = bundleGap(w);
	const y0 = h * 0.5 - gap;
	const xFirst = w * 0.21;
	const step = w * 0.0633;
	const fade = Math.min(w * 0.197, h * 0.36);
	const thin = w < 480;

	ctx.lineJoin = 'round';
	ctx.lineCap = 'butt';
	ctx.lineWidth = thin ? 1.2 : 1.5;

	for (let k = 0; k < LINES - 1; k++) {
		const y = y0 + (LINES - 1 - k) * gap;
		const x = xFirst + k * step + (random() - 0.5) * step * 0.3;
		const fallW = fade * (0.9 + random() * 0.2);
		const power = 2.3 + (random() - 0.5) * 0.5;
		const part = easeOut((time - FALL_START - k * FALL_STAGGER) / FALL_TIME);
		// Older falls are fainter.
		const alpha = (colors.dark ? 0.18 : 0.22) + (0.3 * k) / (LINES - 2);

		const stroke = ctx.createLinearGradient(0, y, 0, y + fade);
		stroke.addColorStop(0, `rgba(${ink},${alpha})`);
		stroke.addColorStop(0.25, `rgba(${ink},${alpha * 0.66})`);
		stroke.addColorStop(1, `rgba(${ink},0)`);
		ctx.strokeStyle = stroke;

		const points: Point[] = [
			[-10, y],
			[x, y],
			...bend({ from: [x, y], width: fallW, height: fade * 1.4, power })
		];
		trace({ ctx, points: cut({ points, part }) });
	}

	const xr = xFirst + (LINES - 1) * step;
	const rise: Point[] = [
		[-10, y0],
		[xr, y0],
		...bend({ from: [xr, y0], width: w + 20 - xr, height: h * 0.08 - y0, steps: 160 })
	];
	strokeGold({
		ctx,
		points: cut({ points: rise, part: easeOut((time - RISE_START) / RISE_TIME) }),
		colors,
		stroke: goldGradient({ ctx, colors, from: 0, to: w }),
		lineWidth: thin ? 2.2 : 2.8,
		glow: thin ? 22 : 36,
		idle: time - RISE_END
	});
};

const sevenFalls: ArtPiece = {
	label: 'Seven pale lines fall away; an eighth, in gold, rises.',
	still: RISE_END,
	draw
};

export { sevenFalls };
