<script lang="ts">
	import { resolve } from '$app/paths';

	import { CASES } from '#lib/cases.ts';
	import Terminal from '../terminal.svelte';

	import type { Case } from '#lib/cases.ts';

	type Props = { cases: string[] };

	let { cases }: Props = $props();

	const TYPE_MS = 400;

	const items = $derived(
		cases.map((id) => CASES.find((item) => item.id === id)).filter((item) => item !== undefined)
	);

	let selected = $state(0);
	let on = $state(false);
	// 'static' is the server render: no JavaScript or reduced motion shows both states stacked.
	let mode = $state<'static' | 'live'>('static');
	let typed = $state(Infinity);
	let ran = $state<'off' | 'on' | undefined>();
	let runs = $state(0);
	let timer = 0;

	const item = $derived(items[selected] as Case | undefined);
	const command = $derived(item?.err.text ?? '');

	const start = () => {
		const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
		const update = () => (mode = motion.matches ? 'static' : 'live');
		update();
		motion.addEventListener('change', update);
		return () => {
			motion.removeEventListener('change', update);
			cancelAnimationFrame(timer);
		};
	};

	const reset = () => {
		cancelAnimationFrame(timer);
		typed = Infinity;
		ran = undefined;
	};

	const pick = (index: number) => {
		selected = index;
		reset();
	};

	const toggle = (next: boolean) => {
		on = next;
		reset();
	};

	const run = () => {
		cancelAnimationFrame(timer);
		ran = undefined;
		typed = 0;
		const begin = performance.now();
		const length = command.length;
		const step = (now: number) => {
			typed = Math.min(length, Math.round(((now - begin) / TYPE_MS) * length));
			if (typed < length) {
				timer = requestAnimationFrame(step);
				return;
			}
			ran = on ? 'on' : 'off';
			runs += 1;
		};
		timer = requestAnimationFrame(step);
	};

	// Arrow keys move between the case tabs, as in the WAI-ARIA tabs pattern.
	const onKeydown = (event: KeyboardEvent) => {
		const next = { ArrowRight: selected + 1, ArrowLeft: selected - 1 }[event.key];
		if (next === undefined) return;
		event.preventDefault();
		pick((next + items.length) % items.length);
		(event.currentTarget as HTMLElement).parentElement
			?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
			[selected]?.focus();
	};
</script>

<!-- The off state is the error alone. The on state is the case's terminal without its first
     line, because the demo prints the command itself. -->
<!-- prettier-ignore -->
{#snippet offOutput(item: Case)}<span class="ind"><span class="d">└</span> <span class="r">{item.err.error}</span></span><span class="ind"><span class="d">└ The agent works it out again.</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet prompt(text: string, caret: boolean)}<span class="blk"><span class="d">$</span> <span class="c">{text}</span>{#if caret}<span class="caret" aria-hidden="true"></span>{/if}</span>{/snippet}

<div class="demo" {@attach start}>
	{#if item}
		{#if mode === 'live' || items.length > 1}
			<div class="tabs" role="tablist" aria-label="Cases">
				{#each items as option, index (option.id)}
					<button
						type="button"
						role="tab"
						aria-selected={index === selected}
						tabindex={index === selected ? 0 : -1}
						onclick={() => pick(index)}
						onkeydown={onKeydown}>{option.id}</button
					>
				{/each}
			</div>
		{/if}

		{#if mode === 'live'}
			<div class="controls">
				<div class="switch" role="group" aria-label="nanaoki">
					<button type="button" aria-pressed={!on} onclick={() => toggle(false)}>nanaoki off</button
					>
					<button type="button" aria-pressed={on} onclick={() => toggle(true)}>nanaoki on</button>
				</div>
				<button type="button" class="run" onclick={run} disabled={typed < command.length}>
					Run
				</button>
			</div>

			<div class={['live', item.badge.kind]} aria-live="polite">
				<Terminal title={item.termTitle} label="What the agent sees">
					<!-- prettier-ignore -->
					{@render prompt(command.slice(0, typed), typed < command.length || !ran)}{#key runs}{#if ran === 'off'}<span
								class="out">{@render offOutput(item)}</span
							>{:else if ran === 'on'}<span class="out on">{@render item.term()}</span>{/if}{/key}
				</Terminal>
			</div>

			<p class="result">
				{#if ran === 'on'}
					<span class="badge {item.badge.kind}">{item.badge.label}</span>
				{:else if ran === 'off'}
					<span class="hint">Now switch nanaoki on and run it again.</span>
				{:else}
					<span class="hint">Press Run.</span>
				{/if}
				<a href="{resolve('/cases')}#{item.id}">See the full case</a>
			</p>
		{:else}
			<div class="stacked">
				<p class="state">Without nanaoki</p>
				<Terminal title={item.termTitle} label="Without nanaoki">
					<!-- prettier-ignore -->
					{@render prompt(command, false)}<span class="out">{@render offOutput(item)}</span>
				</Terminal>
				<p class="state">With nanaoki</p>
				<Terminal title={item.termTitle} label="With nanaoki">
					<!-- prettier-ignore -->
					{@render prompt(command, false)}<span class="out on">{@render item.term()}</span>
				</Terminal>
				<p class="result">
					<span class="badge {item.badge.kind}">{item.badge.label}</span>
					<a href="{resolve('/cases')}#{item.id}">See the full case</a>
				</p>
			</div>
		{/if}
	{/if}
</div>

<style>
	.demo {
		margin: 2.2em 0;
		font: 400 15px/1.4 var(--sans);
	}

	.tabs {
		display: flex;
		gap: 6px;
		margin-bottom: 12px;
		overflow-x: auto;
		scrollbar-width: none;
		padding: 2px;
	}

	.tabs button,
	.switch button,
	.run {
		flex: none;
		font: 500 13.5px/1 var(--mono);
		color: var(--ink-2);
		background: var(--card);
		border: 1px solid var(--rule);
		border-radius: 999px;
		padding: 9px 13px;
		cursor: pointer;
	}

	.tabs button:hover,
	.switch button:hover {
		color: var(--ink);
		border-color: var(--ink-3);
	}

	.tabs button[aria-selected='true'] {
		background: var(--ink);
		border-color: var(--ink);
		color: var(--bg);
	}

	/* Without JavaScript the tabs do nothing, so they are hidden. */
	:global(html:not(.js)) .tabs {
		display: none;
	}

	.controls {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 10px;
		margin-bottom: 12px;
	}

	.switch {
		display: inline-flex;
		gap: 2px;
		padding: 3px;
		border: 1px solid var(--rule);
		border-radius: 999px;
		background: var(--card);
	}

	.switch button {
		border: 0;
		background: none;
		padding: 7px 12px;
	}

	.switch button[aria-pressed='true'] {
		background: var(--ink);
		color: var(--bg);
	}

	.run {
		background: var(--ink);
		border-color: var(--ink);
		color: var(--bg);
		padding: 10px 20px;
	}

	.run:disabled {
		opacity: 0.55;
		cursor: default;
	}

	.live :global(.term-body),
	.stacked :global(.term-body) {
		min-height: 9.5em;
	}

	/* The case terminal starts with its own command line. The demo prints the command, so hide it. */
	.out.on > :global(.blk:first-child) {
		display: none;
	}

	.out > :global(*) {
		animation: line-in 0.25s ease both;
	}

	.out > :global(:nth-child(2)) {
		animation-delay: 0.12s;
	}

	.out > :global(:nth-child(3)) {
		animation-delay: 0.24s;
	}

	.out > :global(:nth-child(4)) {
		animation-delay: 0.36s;
	}

	.out > :global(:nth-child(n + 5)) {
		animation-delay: 0.48s;
	}

	.live.guard .out.on :global(.r) {
		animation: flash 0.9s ease 0.15s;
		border-radius: 3px;
	}

	.stacked .out > :global(*) {
		animation: none;
	}

	.caret {
		display: inline-block;
		width: 0.55em;
		height: 1.1em;
		margin-left: 2px;
		vertical-align: text-bottom;
		background: var(--term-dim);
		animation: blink 1s steps(1) infinite;
	}

	@keyframes line-in {
		from {
			opacity: 0;
		}
	}

	@keyframes flash {
		0%,
		60% {
			background: color-mix(in srgb, var(--term-red) 35%, transparent);
		}

		100% {
			background: transparent;
		}
	}

	@keyframes blink {
		50% {
			opacity: 0;
		}
	}

	.state {
		margin: 18px 0 8px;
		font: 500 12.5px/1 var(--mono);
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--ink-3);
	}

	.state:first-child {
		margin-top: 0;
	}

	.result {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 8px 16px;
		margin: 12px 0 0;
		min-height: 30px;
		font-size: 14.5px;
		color: var(--ink-3);
	}

	.badge {
		font-weight: 500;
		color: var(--ink);
	}

	.badge.guard,
	.badge.lesson {
		color: var(--ok);
	}

	.badge.miss {
		color: var(--ink-2);
	}

	@media (prefers-reduced-motion: reduce) {
		.out > :global(*),
		.live.guard .out.on :global(.r),
		.caret {
			animation: none;
		}
	}

	@media (max-width: 520px) {
		.switch button {
			padding: 7px 10px;
			font-size: 13px;
		}

		.run {
			padding: 10px 16px;
		}
	}
</style>
