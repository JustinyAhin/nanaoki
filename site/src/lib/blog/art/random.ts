// Mulberry32, the same generator as the social card. The same seed gives the same piece.
const rng = (seed: number) => {
	let a = seed >>> 0;
	return () => {
		a = (a + 0x6d2b79f5) | 0;
		let t = Math.imul(a ^ (a >>> 15), 1 | a);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
};

// Canvas gradients need numeric channels, and the colour tokens are hex.
const rgb = (hex: string) => {
	const match = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex.trim());
	if (!match) return '27,26,23';
	return match
		.slice(1)
		.map((part) => parseInt(part, 16))
		.join(',');
};

export { rgb, rng };
