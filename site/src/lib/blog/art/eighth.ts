import { bend, bundleGap, cut, easeOut, goldGradient, strokeGold, trace } from './lines.ts';
import { rgb } from './random.ts';

import type { ArtFrame, ArtPiece } from './index.ts';
import type { Point } from './lines.ts';

// The clock starts when the first pixel is on screen. Wait until more of the piece shows.
const START = 0.5;
const RISE_TIME = 1.6;
const END = START + RISE_TIME;

// The eighth line alone. Seven faint strokes under its start are the trace of the bundle.
const draw = ({ ctx, width: w, height: h, time, colors }: ArtFrame) => {
	const ink = rgb(colors.ink);
	const gap = bundleGap(w);
	// The traces sit near the lower edge, so a tall box on a phone has no dead band under them.
	const y0 = Math.max(h * 0.6, h * 0.9 - 7 * gap);
	const thin = w < 480;
	const traceEnd = w * 0.15;

	const faint = ctx.createLinearGradient(0, 0, traceEnd, 0);
	const alpha = colors.dark ? 0.1 : 0.12;
	faint.addColorStop(0, `rgba(${ink},${alpha})`);
	faint.addColorStop(0.5, `rgba(${ink},${alpha})`);
	faint.addColorStop(1, `rgba(${ink},0)`);
	ctx.strokeStyle = faint;
	ctx.lineCap = 'butt';
	ctx.lineWidth = thin ? 1.2 : 1.5;
	for (let i = 1; i <= 7; i++) {
		const y = y0 + i * gap;
		trace({
			ctx,
			points: [
				[-10, y],
				[traceEnd, y]
			]
		});
	}

	const xr = w * 0.4;
	const rise: Point[] = [
		[-10, y0],
		[xr, y0],
		...bend({ from: [xr, y0], width: w + 20 - xr, height: h * 0.1 - y0, steps: 160 })
	];
	strokeGold({
		ctx,
		points: cut({ points: rise, part: easeOut((time - START) / RISE_TIME) }),
		colors,
		stroke: goldGradient({ ctx, colors, from: 0, to: w }),
		lineWidth: thin ? 2.2 : 2.8,
		glow: thin ? 20 : 30,
		idle: time - END
	});
};

const eighth: ArtPiece = {
	label: 'One gold line runs level, then rises.',
	still: END,
	draw
};

export { eighth };
