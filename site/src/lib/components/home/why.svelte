<script lang="ts">
	import { resolve } from '$app/paths';

	const setups = [
		{ name: 'Nothing', note: '', prevented: '0%', bar: 0, context: '0 tokens', win: false },
		{
			name: 'An AGENTS.md written from my earlier history',
			note: 'In context on every turn',
			prevented: '~31%',
			bar: 31,
			context: '~1,700 tokens, every turn',
			win: false
		},
		{
			name: 'nanaoki lessons, shown when the error happens',
			note: 'Only the ones that match the error',
			prevented: '53-59%',
			bar: 56,
			context: '~36 tokens',
			win: true
		}
	];
</script>

<section id="why">
	<div class="wrap">
		<p class="eyebrow">Why not just CLAUDE.md</p>
		<h2>I was sure a better CLAUDE.md would be enough.</h2>
		<p class="lead">
			So I tested that before I built anything else. I took 108 real cases from my history and
			replayed them. Two AI judges (Claude and GPT, 93% agreement) scored whether each setup would
			have prevented what came next.
		</p>

		<div class="compare" role="table" aria-label="Repeats prevented and context cost per setup">
			<div class="row head" role="row">
				<div role="columnheader">Setup</div>
				<div role="columnheader">Repeats prevented</div>
				<div role="columnheader">Context per trigger</div>
			</div>
			{#each setups as setup (setup.name)}
				<div class="row" class:win={setup.win} role="row">
					<div class="name" role="cell">
						{setup.name}{#if setup.note}<small>{setup.note}</small>{/if}
					</div>
					<div role="cell">
						<span class="lbl">Prevented</span>
						<span class="pct">{setup.prevented}</span>
						<div class="bar"><i style:width="{setup.bar}%"></i></div>
					</div>
					<div class="tok" role="cell">
						<span class="lbl">Context</span>
						{setup.context}
					</div>
				</div>
			{/each}
		</div>

		<div class="caveat">
			<p>
				The file does win on stable decisions like "commit only when I say so". Those belong in a
				file that is always loaded, so <code>nanaoki rules suggest</code>
				proposes lines for yours. It loses on the long tail of one-off errors, and on fixes that live
				in another repo.
			</p>
			<p>
				Read these numbers with care. This is my own data, from one laptop, and a replay scored by
				models. I haven't measured live use yet. Run it on your history and see what your number is.
			</p>
		</div>
		<p class="small">
			<a href={resolve('/cases')}>See real cases from the replay</a>: ten repeats from my history,
			what nanaoki showed the agent, and the two it missed.
		</p>
	</div>
</section>

<style>
	.compare {
		margin-top: 44px;
		border: 1px solid var(--rule);
		border-radius: 12px;
		overflow: hidden;
		background: var(--card);
	}

	/* rows */
	.row {
		display: grid;
		grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr) minmax(0, 1fr);
		align-items: center;
	}

	.row > div {
		padding: 18px 22px;
	}

	.row + .row {
		border-top: 1px solid var(--rule);
	}

	.row.head {
		background: var(--bg-2);
		font: 500 12.5px/1.3 var(--mono);
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--ink-3);
	}

	.name {
		font-weight: 500;
	}

	.name small {
		display: block;
		font-weight: 400;
		color: var(--ink-3);
		font-size: 14px;
	}

	.pct {
		font: 400 30px/1 var(--serif);
	}

	.tok {
		font: 400 15px/1.35 var(--mono);
		color: var(--ink-2);
	}

	.lbl {
		display: none;
	}

	.bar {
		height: 4px;
		border-radius: 2px;
		background: var(--rule);
		margin-top: 10px;
		max-width: 160px;
		overflow: hidden;
	}

	.bar i {
		display: block;
		height: 100%;
		background: var(--ink-3);
	}

	.row.win {
		background: color-mix(in srgb, var(--accent-soft) 55%, var(--card));
	}

	.row.win .pct {
		color: var(--accent-ink);
	}

	.row.win .bar i {
		background: var(--accent);
	}

	/* caveat */
	.caveat {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		gap: 32px;
		margin-top: 32px;
		color: var(--ink-2);
		font-size: 16px;
	}

	.caveat p {
		margin: 0;
	}

	@media (max-width: 720px) {
		.caveat {
			grid-template-columns: minmax(0, 1fr);
			gap: 16px;
		}

		.row.head {
			display: none;
		}

		.row {
			grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		}

		.row > .name {
			grid-column: 1 / -1;
			padding-bottom: 4px;
		}

		.row > div:not(.name) {
			padding-top: 6px;
		}

		.lbl {
			display: block;
			font: 500 11px/1 var(--mono);
			text-transform: uppercase;
			letter-spacing: 0.06em;
			color: var(--ink-3);
			margin-bottom: 8px;
		}
	}
</style>
