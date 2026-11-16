<script lang="ts">
	import { rng } from '#lib/blog/art/random.ts';

	// sameProject is only checked by the schema: it must equal repeats minus crossProject.
	type Props = { total: number; repeats: number; crossProject: number };

	let { total, repeats, crossProject }: Props = $props();

	const COLS = 150;
	const PITCH_X = 6;
	const PITCH_Y = 12;

	const id = $props.id();
	const rows = $derived(Math.ceil(total / COLS));
	const viewBox = $derived(`0 0 ${COLS * PITCH_X} ${rows * PITCH_Y}`);

	// A seeded shuffle (seed 7) picks which marks are repeats, so they spread over the field and
	// the page looks the same on every load. The first picks of the repeats are the cross-project ones.
	const sets = $derived.by(() => {
		const random = rng(7);
		const order = Array.from({ length: total }, (_, index) => index);
		for (let i = order.length - 1; i > 0; i--) {
			const j = Math.floor(random() * (i + 1));
			[order[i], order[j]] = [order[j], order[i]];
		}
		const kind = new Uint8Array(total);
		order.slice(0, repeats).forEach((mark, index) => (kind[mark] = index < crossProject ? 2 : 1));
		return kind;
	});

	// One path for each set, with relative moves to keep the markup small.
	const path = (wanted: number) => {
		let d = '';
		let x = 0;
		let y = 0;
		for (let mark = 0; mark < total; mark++) {
			if (sets[mark] !== wanted) continue;
			const nx = (mark % COLS) * PITCH_X + PITCH_X / 2;
			const ny = Math.floor(mark / COLS) * PITCH_Y + 3;
			d += d ? `m${nx - x} ${ny - y}v6` : `M${nx} ${ny}v6`;
			x = nx;
			y = ny + 6;
		}
		return d;
	};

	const format = (value: number) => value.toLocaleString('en-US');
	const lines = $derived([
		`${format(total)} error episodes, September 1 to November 14.`,
		`${format(repeats)} had a cause that was already fixed between February and August.`,
		`${format(crossProject)} of those had the fix only in another project.`
	]);

	// The server render shows the last step: the final state without JavaScript or motion.
	let step = $state(3);
	let sticky = $state(false);

	const follow = (node: HTMLElement) => {
		const wide = window.matchMedia('(min-width: 720px) and (min-height: 560px)');
		const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
		const figure = node.querySelector<HTMLElement>('.stick');

		const update = () => {
			if (!sticky || !figure) return;
			const box = node.getBoundingClientRect();
			const top = parseFloat(getComputedStyle(figure).top) || 0;
			const travel = box.height - figure.offsetHeight;
			const progress = travel > 0 ? Math.min(1, Math.max(0, (top - box.top) / travel)) : 1;
			step = progress < 1 / 3 ? 1 : progress < 2 / 3 ? 2 : 3;
		};

		const setMode = () => {
			sticky = wide.matches && !motion.matches;
			if (sticky) requestAnimationFrame(update);
		};

		let queued = false;
		const schedule = () => {
			if (queued) return;
			queued = true;
			requestAnimationFrame(() => {
				queued = false;
				update();
			});
		};

		setMode();
		window.addEventListener('scroll', schedule, { passive: true });
		wide.addEventListener('change', setMode);
		motion.addEventListener('change', setMode);
		return () => {
			window.removeEventListener('scroll', schedule);
			wide.removeEventListener('change', setMode);
			motion.removeEventListener('change', setMode);
		};
	};
</script>

<figure class="episodes w-wide" aria-label={lines.join(' ')}>
	<div class={['scroller', { sticky }]} {@attach follow}>
		<div class="stick">
			<ol class="counts">
				{#each lines as line, index (index)}
					<li class={{ now: index + 1 === step, past: index + 1 < step, next: index + 1 > step }}>
						{line}
					</li>
				{/each}
			</ol>

			<div class="field step-{step}" aria-hidden="true">
				<svg class="defs" {viewBox} width="0" height="0">
					<defs>
						<path id="ep-p-{id}" d={path(0)} vector-effect="non-scaling-stroke" />
						<path id="ep-s-{id}" d={path(1)} vector-effect="non-scaling-stroke" />
						<path id="ep-x-{id}" d={path(2)} vector-effect="non-scaling-stroke" />
					</defs>
				</svg>
				<svg class="layer pale" {viewBox} preserveAspectRatio="none">
					<use href="#ep-p-{id}" />
				</svg>
				<svg class="layer before" {viewBox} preserveAspectRatio="none">
					<use href="#ep-s-{id}" />
					<use href="#ep-x-{id}" />
				</svg>
				<!-- Repeats fall by 3 units when they go dark. The cross-project ones rise back in gold. -->
				<svg class="layer same" {viewBox} preserveAspectRatio="none">
					<use href="#ep-s-{id}" y="3" />
				</svg>
				<svg class="layer cross-ink" {viewBox} preserveAspectRatio="none">
					<use href="#ep-x-{id}" y="3" />
				</svg>
				<svg class="layer cross-gold" {viewBox} preserveAspectRatio="none">
					<use href="#ep-x-{id}" />
				</svg>
			</div>

			<div class="steps" role="group" aria-label="Steps of the figure">
				{#each lines as line, index (index)}
					<button
						type="button"
						aria-pressed={step === index + 1}
						aria-label="Step {index + 1}: {line}"
						onclick={() => (step = index + 1)}>{index + 1}</button
					>
				{/each}
			</div>
		</div>
	</div>
</figure>

<style>
	.episodes {
		--gold-2: #b8892a;
		margin: 2.6em 0;
	}

	@media (prefers-color-scheme: dark) {
		.episodes {
			--gold-2: #cfa54a;
		}
	}

	/* The figure stays in view for about 1.5 screens while the steps advance. */
	/* A sticky element only moves inside its parent's content box, so a spacer gives it room. */
	.scroller.sticky::after {
		content: '';
		display: block;
		height: 150vh;
	}

	.scroller.sticky .stick {
		position: sticky;
		top: max(24px, calc((100vh - 470px) / 2));
	}

	.counts {
		list-style: none;
		margin: 0 0 14px;
		padding: 0;
		display: grid;
		gap: 2px;
		font: 400 15px/1.45 var(--sans);
		color: var(--ink-3);
	}

	.counts li {
		transition:
			color 0.3s,
			opacity 0.3s;
	}

	.counts .now {
		color: var(--ink);
		font-weight: 500;
	}

	.counts .next {
		opacity: 0;
	}

	.field {
		position: relative;
		height: 320px;
	}

	.defs {
		position: absolute;
	}

	.layer {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		overflow: visible;
		fill: none;
		stroke-width: 2;
		transition: clip-path 0.6s linear;
	}

	.pale,
	.before {
		stroke: var(--rule);
	}

	.same,
	.cross-ink {
		stroke: var(--ink);
	}

	.cross-gold {
		stroke: var(--gold-2);
	}

	.layer:not(.pale) {
		clip-path: inset(-10px 0 -10px 0);
	}

	/* Hidden layers are clipped from the right or the left, so every change sweeps left to right. */
	.step-1 .same,
	.step-1 .cross-ink,
	.field:not(.step-3) .cross-gold {
		clip-path: inset(-10px 100% -10px 0);
	}

	.field:not(.step-1) .before,
	.step-3 .cross-ink {
		clip-path: inset(-10px 0 -10px 100%);
	}

	.steps {
		display: flex;
		gap: 6px;
		margin-top: 14px;
	}

	.steps button {
		width: 34px;
		height: 34px;
		font: 500 13.5px/1 var(--mono);
		color: var(--ink-2);
		background: var(--card);
		border: 1px solid var(--rule);
		border-radius: 50%;
		cursor: pointer;
	}

	.steps button:hover {
		color: var(--ink);
		border-color: var(--ink-3);
	}

	.steps button[aria-pressed='true'] {
		background: var(--ink);
		border-color: var(--ink);
		color: var(--bg);
	}

	:global(html:not(.js)) .steps {
		display: none;
	}

	@media (max-width: 520px) {
		.field {
			height: 300px;
		}

		.layer {
			stroke-width: 1.2;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.layer,
		.counts li {
			transition: none;
		}
	}
</style>
