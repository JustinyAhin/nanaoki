<script lang="ts">
	import { whenSeen } from '#lib/blog/in-view.ts';

	import type { ViewState } from '#lib/blog/in-view.ts';
	import type { ChartDatum } from '#lib/blog/types.ts';

	type Props = {
		type: 'bar' | 'line' | 'stacked';
		data: ChartDatum[];
		series?: string[];
		title?: string;
		caption?: string;
		unit?: string;
		source?: string;
		highlight?: string;
		width?: 'text' | 'wide' | 'full';
	};

	let {
		type,
		data,
		series,
		title,
		caption,
		unit = '',
		source,
		highlight,
		width = 'text'
	}: Props = $props();

	let view = $state<ViewState>('idle');
	let hover = $state<number | undefined>();
	let tip = $state<{ row: number; part: number } | undefined>();

	const names = $derived(series ?? [title ?? 'Value']);
	const multi = $derived(names.length > 1);
	const parts = $derived(names.map((_, index) => index));
	const rows = $derived(
		data.map((datum) => ({ label: datum.label, values: datum.values ?? [datum.value ?? 0] }))
	);

	const format = (value: number) => value.toLocaleString('en-US', { maximumFractionDigits: 1 });
	const formatAxis = (value: number) =>
		value >= 10000
			? value.toLocaleString('en-US', { notation: 'compact', maximumFractionDigits: 1 })
			: format(value);

	// Round the axis up to 1, 2, 2.5 or 5 times a power of ten, with about four steps.
	const niceTicks = (max: number) => {
		if (max <= 0) return [0, 1];
		const raw = max / 4;
		const power = 10 ** Math.floor(Math.log10(raw));
		const step = [1, 2, 2.5, 5, 10].map((k) => k * power).find((k) => k >= raw) ?? raw;
		const top = Math.ceil(max / step) * step;
		return Array.from({ length: Math.round(top / step) + 1 }, (_, index) => index * step);
	};

	const totals = $derived(rows.map((row) => row.values.reduce((sum, value) => sum + value, 0)));
	const barMax = $derived(
		Math.max(...(type === 'stacked' ? totals : rows.flatMap((row) => row.values)), 0) || 1
	);
	// Room at the end of each bar for its value label, in ch.
	const room = $derived(
		Math.max(
			...rows.flatMap((row, index) =>
				(type === 'stacked' ? [totals[index]] : row.values).map(
					(value, k) => format(value).length + (index === 0 && k === 0 ? unit.length : 0)
				)
			)
		) + 1
	);

	const ticks = $derived(niceTicks(Math.max(...rows.flatMap((row) => row.values), 0)));
	const top = $derived(ticks[ticks.length - 1] || 1);
	const x = (index: number) => (rows.length === 1 ? 50 : (index / (rows.length - 1)) * 100);
	const y = (value: number) => 100 - (value / top) * 100;
	const linePath = (part: number) =>
		rows.map((row, index) => `${index ? 'L' : 'M'}${x(index)} ${y(row.values[part])}`).join(' ');
	const areaPath = $derived(`${linePath(0)} L${x(rows.length - 1)} 100 L${x(0)} 100 Z`);
	const xLabelEvery = $derived(rows.length <= 8 ? 1 : Math.ceil(rows.length / 6));

	// End labels are dropped when they would overlap; the legend, tooltip and table still carry them.
	const endLabels = $derived.by(() => {
		const last = rows.length - 1;
		const kept: { part: number; top: number }[] = [];
		names
			.map((_, part) => ({ part, top: y(rows[last]?.values[part] ?? 0) }))
			.sort((a, b) => a.top - b.top)
			.forEach((label) => {
				if (kept.every((other) => Math.abs(other.top - label.top) > 9)) kept.push(label);
			});
		return kept;
	});

	const color = (part: number, label?: string) => {
		if (multi) return `var(--s${part + 1})`;
		return highlight && label === highlight ? 'var(--accent)' : 'var(--solo)';
	};

	const onPlotMove = (event: PointerEvent) => {
		const box = (event.currentTarget as HTMLElement).getBoundingClientRect();
		const ratio = (event.clientX - box.left) / box.width;
		hover = Math.min(rows.length - 1, Math.max(0, Math.round(ratio * (rows.length - 1))));
	};

	const label = $derived(
		`${title ?? caption ?? 'Chart'}. ${type === 'line' ? 'Line' : type === 'stacked' ? 'Stacked bar' : 'Bar'} chart; the numbers are in the table below.`
	);
</script>

<figure
	class="chart w-{width} {view}"
	{@attach whenSeen({ onChange: (next) => (view = next), threshold: 0.35 })}
>
	{#if title}<p class="title">{title}</p>{/if}

	{#if multi}
		<ul class="legend" aria-hidden="true">
			{#each names as name, part (part)}
				<li>
					<i class={type === 'line' ? 'key-line' : 'key-box'} style:--c={color(part)}></i>{name}
				</li>
			{/each}
		</ul>
	{/if}

	<div class="visual" role="img" aria-label={label}>
		{#if type === 'line'}
			<div class="line-chart">
				<div class="grid" aria-hidden="true">
					{#each ticks as tick (tick)}
						<span class="tick" style:top="{y(tick)}%"><b>{formatAxis(tick)}</b></span>
					{/each}
				</div>
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div class="plot" onpointermove={onPlotMove} onpointerleave={() => (hover = undefined)}>
					<svg viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
						{#if !multi}
							<path class="area" d={areaPath} style:--c={color(0)} />
						{/if}
						{#each parts as part (part)}
							<path class="line" d={linePath(part)} style:--c={color(part)} />
						{/each}
					</svg>
					{#each parts as part (part)}
						{@const last = rows.length - 1}
						<i
							class="dot end"
							style:left="{x(last)}%"
							style:top="{y(rows[last].values[part])}%"
							style:--c={color(part)}
						></i>
					{/each}
					{#each endLabels as end (end.part)}
						<span class="end-label" style:top="{end.top}%"
							>{format(rows[rows.length - 1].values[end.part])}{unit}</span
						>
					{/each}
					{#if hover !== undefined}
						<span class="cross" style:left="{x(hover)}%"></span>
						{#each parts as part (part)}
							<i
								class="dot"
								style:left="{x(hover)}%"
								style:top="{y(rows[hover].values[part])}%"
								style:--c={color(part)}
							></i>
						{/each}
						<div class={['tooltip', { flip: x(hover) > 60 }]} style:left="{x(hover)}%">
							<p class="tip-head">{rows[hover].label}</p>
							{#each names as name, part (part)}
								<p class="tip-row">
									<i class="key-line" style:--c={color(part)}></i><b
										>{format(rows[hover].values[part])}{unit}</b
									>
									{#if multi}<span>{name}</span>{/if}
								</p>
							{/each}
						</div>
					{/if}
				</div>
				<div class="x-labels" aria-hidden="true">
					{#each rows as row, index (index)}
						{#if index % xLabelEvery === 0 || index === rows.length - 1}
							<span style:left="{x(index)}%">{row.label}</span>
						{/if}
					{/each}
				</div>
			</div>
		{:else}
			<div class="bars" style:--room="{room}ch">
				{#each rows as row, index (index)}
					<div class="bar-row" style:--i={index}>
						<p class="bar-label">{row.label}</p>
						{#if type === 'stacked'}
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<div class="track" onpointerleave={() => (tip = undefined)}>
								<span class="stack" style:--v={totals[index] / barMax}>
									{#each row.values as value, part (part)}
										{#if value > 0}
											<!-- svelte-ignore a11y_no_static_element_interactions -->
											<span
												class={['seg', { lift: tip?.row === index && tip.part === part }]}
												style:flex-grow={value}
												style:--c={color(part)}
												onpointerenter={() => (tip = { row: index, part })}
											></span>
										{/if}
									{/each}
								</span>
								<span class="value">{format(totals[index])}{index === 0 ? unit : ''}</span>
								{#if tip?.row === index}
									<div class="tooltip bar-tip">
										<p class="tip-row">
											<i class="key-box" style:--c={color(tip.part)}></i><b
												>{format(row.values[tip.part])}{unit}</b
											>
											<span>{names[tip.part]}</span>
										</p>
									</div>
								{/if}
							</div>
						{:else}
							{#each row.values as value, part (part)}
								<div class="track">
									<span class="bar" style:--v={value / barMax} style:--c={color(part, row.label)}
									></span>
									<span class="value">{format(value)}{index === 0 && part === 0 ? unit : ''}</span>
								</div>
							{/each}
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>

	{#if caption || source}
		<figcaption>
			{#if caption}{caption}{/if}
			{#if source}<span class="source">Source: {source}</span>{/if}
		</figcaption>
	{/if}

	<details class="table">
		<summary>Show the numbers</summary>
		<div class="table-scroll">
			<table>
				<thead>
					<tr>
						<th scope="col">{type === 'line' ? 'Point' : 'Item'}</th>
						{#each names as name, part (part)}<th scope="col">{name}</th>{/each}
						{#if type === 'stacked'}<th scope="col">Total</th>{/if}
					</tr>
				</thead>
				<tbody>
					{#each rows as row, index (index)}
						<tr>
							<th scope="row">{row.label}</th>
							{#each row.values as value, part (part)}<td>{format(value)}</td>{/each}
							{#if type === 'stacked'}<td>{format(totals[index])}</td>{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</details>
</figure>

<style>
	/* Series colours are checked with the dataviz palette validator for each mode. */
	.chart {
		--solo: var(--ink-2);
		--s1: #2f63a8;
		--s2: #c8402c;
		--s3: #1b8f6a;
		--s4: #c08a1e;
		margin: 2.4em 0;
		font: 400 14px/1.4 var(--sans);
		color: var(--ink-2);
	}

	@media (prefers-color-scheme: dark) {
		.chart {
			--s1: #3f80d4;
			--s2: #e0583f;
			--s3: #18a274;
			--s4: #b07e10;
		}
	}

	.title {
		margin: 0 0 12px;
		font: 500 15px/1.4 var(--sans);
		color: var(--ink);
	}

	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: 6px 18px;
		margin: 0 0 14px;
		padding: 0;
		list-style: none;
		color: var(--ink-2);
	}

	.legend li {
		display: inline-flex;
		align-items: center;
		gap: 8px;
	}

	.key-box,
	.key-line {
		display: inline-block;
		flex: none;
		background: var(--c);
	}

	.key-box {
		width: 10px;
		height: 10px;
		border-radius: 2px;
	}

	.key-line {
		width: 14px;
		height: 2px;
		border-radius: 1px;
	}

	/* bars */
	.bars {
		display: grid;
		gap: 14px;
	}

	.bar-label {
		margin: 0 0 5px;
		color: var(--ink);
		text-wrap: pretty;
	}

	.track {
		position: relative;
		display: flex;
		align-items: center;
		gap: 8px;
		min-height: 18px;
	}

	.track + .track {
		margin-top: 3px;
	}

	.bar,
	.stack {
		flex: none;
		width: calc(var(--v) * (100% - var(--room) - 8px));
		min-width: 2px;
		height: 14px;
		transform-origin: left center;
		transition: transform 0.9s cubic-bezier(0.2, 0.7, 0.2, 1) calc(var(--i) * 90ms);
	}

	.bar {
		background: var(--c);
		border-radius: 0 4px 4px 0;
	}

	.stack {
		display: flex;
		gap: 2px;
	}

	.seg {
		min-width: 2px;
		background: var(--c);
		transition: filter 0.15s;
	}

	.seg:last-child {
		border-radius: 0 4px 4px 0;
	}

	.seg.lift {
		filter: brightness(1.15) saturate(1.1);
	}

	.value {
		font-variant-numeric: tabular-nums;
		color: var(--ink-2);
		white-space: nowrap;
		transition: opacity 0.5s ease calc(var(--i) * 90ms + 0.4s);
	}

	.armed .bar,
	.armed .stack {
		transform: scaleX(0);
	}

	.armed .value {
		opacity: 0;
	}

	/* line */
	.line-chart {
		position: relative;
		padding: 8px 4.2em 26px 3em;
	}

	.grid {
		position: absolute;
		inset: 8px 4.2em 26px 3em;
		pointer-events: none;
	}

	.tick {
		position: absolute;
		left: 0;
		right: 0;
		border-top: 1px solid var(--rule);
	}

	.tick b {
		position: absolute;
		right: calc(100% + 8px);
		transform: translateY(-50%);
		font-weight: 400;
		font-size: 13px;
		color: var(--ink-3);
		font-variant-numeric: tabular-nums;
	}

	.plot {
		position: relative;
		height: clamp(180px, 30vw, 260px);
		touch-action: pan-y;
	}

	svg {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		overflow: visible;
		transition: clip-path 1.4s cubic-bezier(0.3, 0.1, 0.2, 1);
		clip-path: inset(-10px -10px -10px -10px);
	}

	.armed svg {
		clip-path: inset(-10px 100% -10px -10px);
	}

	.line {
		fill: none;
		stroke: var(--c);
		stroke-width: 2;
		stroke-linejoin: round;
		stroke-linecap: round;
		vector-effect: non-scaling-stroke;
	}

	.area {
		fill: var(--c);
		opacity: 0.07;
	}

	.dot {
		position: absolute;
		width: 12px;
		height: 12px;
		margin: -6px 0 0 -6px;
		border-radius: 50%;
		background: var(--c);
		border: 2px solid var(--bg);
		pointer-events: none;
	}

	.end {
		transition: opacity 0.4s ease 1.2s;
	}

	.armed .end,
	.armed .end-label {
		opacity: 0;
	}

	.end-label {
		position: absolute;
		left: calc(100% + 10px);
		transform: translateY(-50%);
		color: var(--ink);
		white-space: nowrap;
		font-variant-numeric: tabular-nums;
		transition: opacity 0.4s ease 1.2s;
	}

	.cross {
		position: absolute;
		top: 0;
		bottom: 0;
		width: 1px;
		background: var(--ink-3);
		pointer-events: none;
	}

	.x-labels {
		position: absolute;
		left: 3em;
		right: 4.2em;
		bottom: 0;
		height: 20px;
	}

	.x-labels span {
		position: absolute;
		transform: translateX(-50%);
		font-size: 13px;
		color: var(--ink-3);
		white-space: nowrap;
	}

	.x-labels span:first-child {
		transform: none;
	}

	.x-labels span:last-child {
		transform: translateX(-100%);
	}

	/* tooltip */
	.tooltip {
		position: absolute;
		top: 0;
		z-index: 2;
		transform: translateX(12px);
		min-width: 120px;
		padding: 8px 10px;
		background: var(--card);
		border: 1px solid var(--rule);
		border-radius: 8px;
		box-shadow: 0 4px 14px rgb(0 0 0 / 0.08);
		pointer-events: none;
	}

	.tooltip.flip {
		transform: translateX(calc(-100% - 12px));
	}

	.bar-tip {
		top: auto;
		bottom: calc(100% + 6px);
		left: 0;
		transform: none;
	}

	.tip-head {
		margin: 0 0 4px;
		color: var(--ink-3);
		font-size: 13px;
	}

	.tip-row {
		display: flex;
		align-items: center;
		gap: 8px;
		margin: 0;
		white-space: nowrap;
	}

	.tip-row b {
		font-weight: 600;
		color: var(--ink);
		font-variant-numeric: tabular-nums;
	}

	.tip-row span {
		color: var(--ink-3);
	}

	/* caption and table */
	figcaption {
		margin-top: 14px;
		color: var(--ink-3);
		text-wrap: pretty;
	}

	.source {
		display: block;
		margin-top: 4px;
		font-size: 13px;
	}

	.table {
		margin-top: 10px;
	}

	.table summary {
		cursor: pointer;
		width: max-content;
		font: 400 13px/1.4 var(--mono);
		color: var(--ink-3);
	}

	.table summary:hover {
		color: var(--ink);
	}

	.table-scroll {
		overflow-x: auto;
		margin-top: 10px;
	}

	table {
		border-collapse: collapse;
		font-size: 14px;
		font-variant-numeric: tabular-nums;
	}

	th,
	td {
		text-align: left;
		padding: 5px 16px 5px 0;
		border-bottom: 1px solid var(--rule);
		font-weight: 400;
	}

	thead th {
		color: var(--ink);
		font-weight: 500;
	}

	td {
		text-align: right;
	}

	@media (max-width: 520px) {
		.line-chart {
			padding-right: 3.4em;
		}

		.grid,
		.x-labels {
			right: 3.4em;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.bar,
		.stack,
		.value,
		svg,
		.end,
		.end-label {
			transition: none;
		}
	}
</style>
