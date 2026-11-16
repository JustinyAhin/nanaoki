import { eighth } from './eighth.ts';
import { otherRepo } from './other-repo.ts';
import { sevenFalls } from './seven-falls.ts';

type ArtColors = {
	ink: string;
	gold1: string;
	gold2: string;
	gold3: string;
	dark: boolean;
};

type ArtFrame = {
	ctx: CanvasRenderingContext2D;
	width: number;
	height: number;
	// Seconds of visible time. The clock stops while the piece is off-screen.
	time: number;
	seed: number;
	colors: ArtColors;
};

type ArtPiece = {
	// Alt text for the canvas.
	label: string;
	// The time of the still frame for reduced motion.
	still: number;
	draw: (frame: ArtFrame) => void;
};

const ART: Record<string, ArtPiece> = {
	eighth,
	'other-repo': otherRepo,
	'seven-falls': sevenFalls
};

export { ART };
export type { ArtColors, ArtFrame, ArtPiece };
