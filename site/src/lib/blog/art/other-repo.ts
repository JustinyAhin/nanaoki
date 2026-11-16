import { bundleGap, cut, easeInOut, goldGradient, strokeGold, trace } from './lines.ts';
import { rgb, rng } from './random.ts';

import type { ArtFrame, ArtPiece } from './index.ts';
import type { Point } from './lines.ts';

const LINES = 5;
const START = 0.5;
const DRAW_TIME = 1.5;
const DIP_TIME = 2;
const STAGGER = 0.25;
const END = START + STAGGER + DIP_TIME;
// The spec fixes seed 30 for the spacing. The default seed of the tag is 1.
const SEED_BASE = 29;

// A dip that falls like the lines of the social card and comes back level. Smooth at both ends.
const dip = ({ x, y, width, depth }: { x: number; y: number; width: number; depth: number }) =>
	Array.from({ length: 60 }, (_, k): Point => {
		const u = (k + 1) / 60;
		return [x + width * u, y + depth * Math.pow(Math.sin(Math.PI * Math.pow(u, 1.8)), 2)];
	});

// Two repos. Above, one line dips once and is gold from there. Below, the same line dips again and again.
const draw = ({ ctx, width: w, height: h, time, seed, colors }: ArtFrame) => {
	const random = rng(SEED_BASE + seed);
	const narrow = w < 480;
	const ink = rgb(colors.ink);
	const soft = colors.dark ? '150,147,139' : '111,107,98';
	const gap = bundleGap(w);
	const jitter = Math.min(4, gap * 0.5);
	const dipW = w * (narrow ? 0.17 : 0.088);
	const depth = Math.min(h * 0.12, dipW * 0.5);
	const lineWidth = narrow ? 1.2 : 1.5;

	const band = ({ centre, dips, fixed }: { centre: number; dips: number[]; fixed: boolean }) => {
		for (let i = 0; i < LINES; i++) {
			const y = centre + (i - (LINES - 1) / 2) * gap + (random() - 0.5) * jitter;
			const delay = START + random() * STAGGER;
			ctx.lineCap = 'butt';
			ctx.lineJoin = 'round';
			ctx.lineWidth = lineWidth;
			ctx.strokeStyle = `rgba(${soft},${0.55 + 0.1 * i})`;

			// The lowest line of each band is the one that dips, into the empty paper.
			if (i < LINES - 1) {
				const part = easeInOut((time - delay) / DRAW_TIME);
				trace({
					ctx,
					points: [
						[-10, y],
						[-10 + (w + 20) * part, y]
					]
				});
				continue;
			}

			const points: Point[] = [[-10, y]];
			for (const at of dips) {
				points.push([w * at, y], ...dip({ x: w * at, y, width: dipW, depth }));
			}
			points.push([w + 10, y]);
			const drawn = cut({ points, part: easeInOut((time - delay) / DIP_TIME) });

			if (fixed) {
				const x = w * dips[0];
				ctx.strokeStyle = `rgb(${soft})`;
				trace({ ctx, points: drawn.slice(0, 2) });
				strokeGold({
					ctx,
					points: drawn.slice(1),
					colors,
					stroke: goldGradient({
						ctx,
						colors,
						from: x + dipW * 0.5,
						to: w,
						lead: [[x, `rgb(${soft})`]]
					}),
					lineWidth: narrow ? 1.8 : 2.2,
					glow: narrow ? 14 : 20,
					idle: time - END
				});
				continue;
			}

			const stroke = ctx.createLinearGradient(0, 0, w, 0);
			stroke.addColorStop(0, `rgb(${soft})`);
			for (const at of dips) {
				const from = at;
				const to = Math.min(1, at + dipW / w);
				stroke.addColorStop(from, `rgb(${soft})`);
				stroke.addColorStop(from + (to - from) * 0.2, `rgb(${ink})`);
				stroke.addColorStop(to - (to - from) * 0.15, `rgb(${ink})`);
				stroke.addColorStop(to, `rgb(${soft})`);
			}
			ctx.strokeStyle = stroke;
			ctx.lineWidth = lineWidth + 0.3;
			trace({ ctx, points: drawn });
		}
	};

	band({ centre: h * 0.25, dips: [0.3], fixed: true });
	band({ centre: h * 0.66, dips: narrow ? [0.48, 0.72] : [0.55, 0.65, 0.75], fixed: false });
};

const otherRepo: ArtPiece = {
	label:
		'Two bands of lines. In the upper band one line dips once and turns gold. In the lower band the same line dips three times.',
	still: END,
	draw
};

export { otherRepo };
