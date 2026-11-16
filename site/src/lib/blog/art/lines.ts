import type { ArtColors } from './index.ts';

type Point = [number, number];

const clamp = (value: number, low = 0, high = 1) => Math.min(high, Math.max(low, value));

const easeOut = (t: number) => 1 - Math.pow(1 - clamp(t), 3);

const easeInOut = (t: number) => {
	const x = clamp(t);
	return x < 0.5 ? 2 * x * x : 1 - Math.pow(-2 * x + 2, 2) / 2;
};

// The social card has 10 px between lines at 1200 px. Narrow columns need less.
const bundleGap = (width: number) => clamp(width * 0.0118, 5, 10);

// The first part of a polyline, measured by length.
const cut = ({ points, part }: { points: Point[]; part: number }): Point[] => {
	if (part <= 0) return [];
	if (part >= 1) return points;

	let total = 0;
	for (let i = 1; i < points.length; i++) {
		total += Math.hypot(points[i][0] - points[i - 1][0], points[i][1] - points[i - 1][1]);
	}

	const stop = total * part;
	const out: Point[] = [points[0]];
	let run = 0;
	for (let i = 1; i < points.length; i++) {
		const [ax, ay] = points[i - 1];
		const [bx, by] = points[i];
		const length = Math.hypot(bx - ax, by - ay);
		if (run + length < stop) {
			out.push(points[i]);
			run += length;
			continue;
		}
		const k = (stop - run) / (length || 1);
		out.push([ax + (bx - ax) * k, ay + (by - ay) * k]);
		break;
	}
	return out;
};

const trace = ({ ctx, points }: { ctx: CanvasRenderingContext2D; points: Point[] }) => {
	if (points.length < 2) return;
	ctx.beginPath();
	ctx.moveTo(...points[0]);
	for (let i = 1; i < points.length; i++) ctx.lineTo(...points[i]);
	ctx.stroke();
};

// The curve of the social card: y = y0 + height * t^2.3. A negative height rises.
const bend = ({
	from,
	width,
	height,
	power = 2.3,
	steps = 100
}: {
	from: Point;
	width: number;
	height: number;
	power?: number;
	steps?: number;
}): Point[] =>
	Array.from({ length: steps }, (_, k) => {
		const t = (k + 1) / steps;
		return [from[0] + width * t, from[1] + height * Math.pow(t, power)];
	});

const goldStops = (colors: ArtColors): [number, string][] =>
	colors.dark
		? [
				[0, '#8d6a22'],
				[0.3, '#cfa54a'],
				[0.55, '#f2dc9c'],
				[0.7, '#c2963a'],
				[0.85, '#e9cd82'],
				[1, '#fbf0c8']
			]
		: [
				[0, colors.gold1],
				[0.3, colors.gold2],
				[0.55, colors.gold3],
				[0.7, colors.gold2],
				[0.85, colors.gold3],
				[1, colors.gold2]
			];

// The gold runs from x = from to x = to. Stops in `lead` come before it, for a line that turns gold.
const goldGradient = ({
	ctx,
	colors,
	from,
	to,
	lead = []
}: {
	ctx: CanvasRenderingContext2D;
	colors: ArtColors;
	from: number;
	to: number;
	lead?: [number, string][];
}) => {
	const span = to - from || 1;
	const gradient = ctx.createLinearGradient(lead[0]?.[0] ?? from, 0, to, 0);
	const start = lead[0]?.[0] ?? from;
	const at = (x: number) => clamp((x - start) / (to - start || 1));
	for (const [x, color] of lead) gradient.addColorStop(at(x), color);
	for (const [stop, color] of goldStops(colors))
		gradient.addColorStop(at(from + span * stop), color);
	return gradient;
};

const SHEEN_WAIT = 2.5;
const SHEEN_TIME = 3.2;
const SHEEN_PERIOD = 11;

// Strokes a gold line. Dark pages add the glow of the social card.
// `idle` is the time since the line was complete: a slow highlight then moves along it.
const strokeGold = ({
	ctx,
	points,
	colors,
	stroke,
	lineWidth,
	glow,
	idle = 0
}: {
	ctx: CanvasRenderingContext2D;
	points: Point[];
	colors: ArtColors;
	stroke: CanvasGradient;
	lineWidth: number;
	glow: number;
	idle?: number;
}) => {
	if (points.length < 2) return;
	ctx.lineCap = 'round';
	ctx.lineJoin = 'round';

	if (colors.dark) {
		ctx.save();
		ctx.shadowColor = 'rgba(226,180,84,.55)';
		ctx.shadowBlur = glow;
		ctx.strokeStyle = 'rgba(226,180,84,.25)';
		ctx.lineWidth = lineWidth + 0.2;
		trace({ ctx, points });
		ctx.restore();
	}

	ctx.strokeStyle = stroke;
	ctx.lineWidth = lineWidth;
	trace({ ctx, points });

	const cycle = (idle - SHEEN_WAIT) % SHEEN_PERIOD;
	if (idle < SHEEN_WAIT || cycle > SHEEN_TIME) return;

	const progress = cycle / SHEEN_TIME;
	const x0 = points[0][0];
	const x1 = points[points.length - 1][0];
	const band = Math.max(50, (x1 - x0) * 0.14);
	const centre = x0 - band + (x1 - x0 + 2 * band) * easeInOut(progress);
	const light = colors.dark ? '255,250,232' : '250,234,186';
	const sheen = ctx.createLinearGradient(centre - band, 0, centre + band, 0);
	sheen.addColorStop(0, `rgba(${light},0)`);
	sheen.addColorStop(0.5, `rgba(${light},${0.85 * Math.sin(Math.PI * progress)})`);
	sheen.addColorStop(1, `rgba(${light},0)`);
	ctx.strokeStyle = sheen;
	trace({ ctx, points });
};

export { bend, bundleGap, clamp, cut, easeInOut, easeOut, goldGradient, strokeGold, trace };
export type { Point };
