<script lang="ts">
	// CSS scroll-driven animation draws the bar where the browser supports it.
	// Elsewhere this script sets --p from the position of the article.
	const follow = (bar: HTMLElement) => {
		if (CSS.supports('animation-timeline: view()')) return;
		const article = bar.closest('article');
		if (!article) return;

		let queued = false;
		const update = () => {
			queued = false;
			const box = article.getBoundingClientRect();
			const total = box.height - window.innerHeight;
			const done = total > 0 ? -box.top / total : 1;
			bar.style.setProperty('--p', String(Math.min(1, Math.max(0, done))));
		};
		const schedule = () => {
			if (queued) return;
			queued = true;
			requestAnimationFrame(update);
		};

		update();
		window.addEventListener('scroll', schedule, { passive: true });
		window.addEventListener('resize', schedule);
		return () => {
			window.removeEventListener('scroll', schedule);
			window.removeEventListener('resize', schedule);
		};
	};
</script>

<div class="progress" aria-hidden="true" {@attach follow}></div>

<style>
	.progress {
		--gold-2: #b8892a;
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 10;
		height: 2px;
		background: var(--gold-2);
		transform-origin: left center;
		transform: scaleX(var(--p, 0));
		pointer-events: none;
	}

	@media (prefers-color-scheme: dark) {
		.progress {
			--gold-2: #cfa54a;
		}
	}

	/* The parent article sets view-timeline-name: --article. */
	@supports (animation-timeline: view()) {
		.progress {
			animation: progress linear both;
			animation-timeline: --article;
			animation-range: contain 0% contain 100%;
		}
	}

	@keyframes progress {
		from {
			transform: scaleX(0);
		}

		to {
			transform: scaleX(1);
		}
	}
</style>
