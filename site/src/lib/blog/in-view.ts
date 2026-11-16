type ViewState = 'idle' | 'armed' | 'seen';

const inViewport = (node: Element) => {
	const box = node.getBoundingClientRect();
	return box.top < window.innerHeight && box.bottom > 0;
};

// Calls back 'armed' for an element below the fold and 'seen' when it scrolls into view.
// An element that is already on screen at hydration goes straight to 'seen', so nothing flashes.
// Without JavaScript the state stays 'idle', and the CSS must show the content.
const whenSeen =
	({ onChange, threshold = 0.2 }: { onChange: (state: ViewState) => void; threshold?: number }) =>
	(node: Element) => {
		const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		if (reduce || !('IntersectionObserver' in window) || inViewport(node)) {
			onChange('seen');
			return;
		}

		onChange('armed');
		const observer = new IntersectionObserver(
			(entries) => {
				if (!entries.some((entry) => entry.isIntersecting)) return;
				onChange('seen');
				observer.disconnect();
			},
			{ threshold, rootMargin: '0px 0px -8% 0px' }
		);

		observer.observe(node);
		return () => observer.disconnect();
	};

export { whenSeen };
export type { ViewState };
