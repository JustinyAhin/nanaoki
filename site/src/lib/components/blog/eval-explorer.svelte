<script lang="ts">
	type Slice = {
		id: string;
		label: string;
		judge: string;
		n: number;
		values: (number | null)[];
		ci?: ([number, number] | null)[];
		extra?: { label: string; value: number };
		caption?: string;
	};

	type Props = { setups: string[]; tokens: number[]; slices: Slice[] };

	let { setups, tokens, slices }: Props = $props();

	const JUDGES: Record<string, string> = { claude: 'Claude', gpt: 'GPT' };

	// The slice buttons, in the order of the data, one for each id.
	const groups = $derived(
		slices
			.filter((slice, index) => slices.findIndex((other) => other.id === slice.id) === index)
			.map((slice) => ({ id: slice.id, label: slice.label, n: slice.n }))
	);
	const judges = $derived([...new Set(slices.map((slice) => slice.judge))]);

	let group = $state('');
	let judge = $state('');

	const groupId = $derived(group || groups[0]?.id);
	const judgeId = $derived.by(() => {
		const wanted = judge || judges[0];
		const has = slices.some((slice) => slice.id === groupId && slice.judge === wanted);
		return has ? wanted : (slices.find((slice) => slice.id === groupId)?.judge ?? wanted);
	});
	const slice = $derived(
		slices.find((item) => item.id === groupId && item.judge === judgeId) ?? slices[0]
	);
	const caption = $derived(
		slice?.caption ?? slices.find((item) => item.id === groupId && item.caption)?.caption
	);

	// The last setup is the product as shipped, so it is drawn darker.
	const rows = $derived(
		setups.map((name, index) => ({
			name,
			value: slice?.values[index] ?? null,
			ci: slice?.ci?.[index] ?? null,
			kind: index === setups.length - 1 ? 'main' : 'base'
		}))
	);
	const tokenMax = $derived(Math.max(...tokens, 1));
	const format = (value: number) => value.toLocaleString('en-US');

	// The table for readers without JavaScript: the first slice for each judge, and the tokens.
	const tableSlices = $derived(slices.filter((item) => item.id === groups[0]?.id));
</script>

{#snippet bar({
	name,
	value,
	ci,
	kind
}: {
	name: string;
	value: number | null;
	ci: [number, number] | null;
	kind: string;
})}
	<div class="row {kind}">
		<p class="name">{name}</p>
		<div class="track">
			{#if value === null}
				<span class="none">No number for this slice</span>
			{:else}
				<span class="bar" style:--v={value / 100}></span>
				{#if ci}
					<span
						class="ci"
						style:--lo={ci[0] / 100}
						style:--hi={ci[1] / 100}
						title="95% interval {ci[0]}–{ci[1]}%"
					></span>
				{/if}
				<span class="value" style:--v={Math.max(value, ci?.[1] ?? 0) / 100}>{value}%</span>
			{/if}
		</div>
	</div>
{/snippet}

{#snippet table()}
	<table>
		<thead>
			<tr>
				<th scope="col">Setup</th>
				{#each tableSlices as item (item.judge)}
					<th scope="col">{JUDGES[item.judge] ?? item.judge} judge</th>
				{/each}
				<th scope="col">Tokens per trigger</th>
			</tr>
		</thead>
		<tbody>
			{#each setups as name, index (index)}
				<tr>
					<th scope="row">{name}</th>
					{#each tableSlices as item (item.judge)}
						<td>{item.values[index] === null ? '–' : `${item.values[index]}%`}</td>
					{/each}
					<td>{tokens[index] === undefined ? '–' : format(tokens[index])}</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/snippet}

<figure class="explorer w-wide" aria-label="Repeats prevented in the replay">
	<div class="controls">
		<div class="slices" role="group" aria-label="Slice of the cases">
			{#each groups as option (option.id)}
				<button
					type="button"
					aria-pressed={option.id === groupId}
					onclick={() => (group = option.id)}
				>
					{option.label}<span>{option.n}</span>
				</button>
			{/each}
		</div>
		{#if judges.length > 1}
			<div class="judges" role="group" aria-label="Judge">
				{#each judges as option (option)}
					{@const available = slices.some((item) => item.id === groupId && item.judge === option)}
					<button
						type="button"
						aria-pressed={option === judgeId}
						disabled={!available}
						onclick={() => (judge = option)}
					>
						{JUDGES[option] ?? option}
					</button>
				{/each}
			</div>
		{/if}
	</div>

	<p class="title">
		Repeats prevented in the replay
		<span>{slice?.label}, {slice?.n} cases, {JUDGES[judgeId] ?? judgeId} judge</span>
	</p>

	<div class="bars" aria-live="polite">
		{#each rows as row (row.name)}{@render bar(row)}{/each}
		{#if slice?.extra}
			{@render bar({ name: slice.extra.label, value: slice.extra.value, ci: null, kind: 'extra' })}
		{/if}
	</div>

	{#if caption}<p class="caption">{caption}</p>{/if}

	<div class="tokens">
		<p class="title">Context per trigger <span>tokens the agent reads</span></p>
		{#each setups as name, index (index)}
			{#if tokens[index] !== undefined}
				<div class="token-row">
					<p class="name">{name}</p>
					<div class="track">
						<span class="bar thin" style:--v={tokens[index] / tokenMax}></span>
						<span class="value" style:--v={tokens[index] / tokenMax}>{format(tokens[index])}</span>
					</div>
				</div>
			{/if}
		{/each}
	</div>

	<div class="nojs">{@render table()}</div>
	<details class="numbers">
		<summary>Show the numbers</summary>
		<div class="scroll">{@render table()}</div>
	</details>
</figure>

<style>
	.explorer {
		margin: 2.6em 0;
		font: 400 14.5px/1.4 var(--sans);
		color: var(--ink-2);
	}

	.controls {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 10px 16px;
		margin-bottom: 20px;
	}

	.slices,
	.judges {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	button {
		font: 500 13.5px/1 var(--sans);
		color: var(--ink-2);
		background: var(--card);
		border: 1px solid var(--rule);
		border-radius: 999px;
		padding: 8px 12px;
		cursor: pointer;
	}

	button span {
		margin-left: 6px;
		font: 400 12.5px var(--mono);
		opacity: 0.7;
	}

	button:hover:not(:disabled) {
		color: var(--ink);
		border-color: var(--ink-3);
	}

	button[aria-pressed='true'] {
		background: var(--ink);
		border-color: var(--ink);
		color: var(--bg);
	}

	button:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	/* The controls need JavaScript. Without it the chart shows the first slice and the table. */
	:global(html:not(.js)) .controls,
	:global(html:not(.js)) .numbers {
		display: none;
	}

	.nojs {
		display: none;
		margin-top: 18px;
		overflow-x: auto;
	}

	:global(html:not(.js)) .nojs {
		display: block;
	}

	.title {
		margin: 0 0 14px;
		font: 500 15px/1.4 var(--sans);
		color: var(--ink);
	}

	.title span {
		display: block;
		font-weight: 400;
		font-size: 13.5px;
		color: var(--ink-3);
	}

	.bars {
		display: grid;
		gap: 12px;
		padding-bottom: 6px;
	}

	.name {
		margin: 0 0 5px;
		color: var(--ink);
		text-wrap: pretty;
	}

	.track {
		position: relative;
		height: 16px;
		margin-right: 3.6em;
	}

	.bar {
		position: absolute;
		left: 0;
		top: 1px;
		height: 14px;
		width: max(calc(var(--v) * 100%), 2px);
		background: var(--ink-3);
		border-radius: 0 4px 4px 0;
		transition: width 0.3s ease;
	}

	.main .bar {
		background: var(--ink);
	}

	.extra .bar {
		background: var(--ok);
	}

	.bar.thin {
		height: 8px;
		top: 4px;
	}

	.ci {
		position: absolute;
		top: 7px;
		left: calc(var(--lo) * 100%);
		width: calc((var(--hi) - var(--lo)) * 100%);
		height: 1px;
		background: var(--ink-2);
		transition:
			left 0.3s ease,
			width 0.3s ease;
	}

	.ci::before,
	.ci::after {
		content: '';
		position: absolute;
		top: -4px;
		width: 1px;
		height: 9px;
		background: var(--ink-2);
	}

	.ci::before {
		left: 0;
	}

	.ci::after {
		right: 0;
	}

	.value {
		position: absolute;
		top: -2px;
		left: calc(var(--v) * 100% + 8px);
		color: var(--ink);
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
		transition: left 0.3s ease;
	}

	.none {
		position: absolute;
		top: -2px;
		font-size: 13.5px;
		color: var(--ink-3);
	}

	.caption {
		margin: 16px 0 0;
		color: var(--ink-2);
		text-wrap: pretty;
	}

	.tokens {
		display: grid;
		gap: 8px;
		margin-top: 26px;
		padding-top: 18px;
		border-top: 1px solid var(--rule);
	}

	.tokens .title {
		margin-bottom: 4px;
	}

	.token-row .name {
		font-size: 13.5px;
		color: var(--ink-2);
		margin-bottom: 2px;
	}

	.numbers {
		margin-top: 16px;
	}

	.numbers summary {
		cursor: pointer;
		width: max-content;
		font: 400 13px/1.4 var(--mono);
		color: var(--ink-3);
		padding: 0;
		border: 0;
		background: none;
	}

	.scroll {
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
		padding: 6px 16px 6px 0;
		border-bottom: 1px solid var(--rule);
		text-align: left;
		font-weight: 400;
		vertical-align: top;
	}

	thead th {
		color: var(--ink);
		font-weight: 500;
	}

	td {
		text-align: right;
		white-space: nowrap;
	}

	@media (prefers-reduced-motion: reduce) {
		.bar,
		.ci,
		.value {
			transition: none;
		}
	}
</style>
