<script lang="ts">
	const quotes = [
		{
			text: "Making the same mistakes again, and then even explaining it with, I said to myself I shouldn't do it this way",
			source: 'r/ClaudeCode, in a thread about a MISTAKES.md file'
		},
		{
			text: 'A rule can be sitting in CLAUDE.md, written correctly, and still get ignored ... CLAUDE.md was read a hundred messages ago.',
			source: 'r/ClaudeCode, same thread'
		},
		{
			text: "Your folder layout, your naming rules, every 'no, not like that'. Gone the second the session ends.",
			source: 'X'
		},
		{
			text: "You're relying on Claude to find the file, and then find the mistake, and then follow the rule.",
			source: 'r/ClaudeCode, same thread'
		}
	];

	const stats = [
		{
			value: '43%',
			text: 'of the errors from Sep to Nov were ones my agents had already hit and fixed between Feb and Aug.'
		},
		{
			value: '69%',
			text: 'of those repeats had the fix only in a different project. The next repo started from zero.'
		},
		{
			value: '471',
			text: 'sessions across 56 projects hit the same zsh quoting error. The agent adds quotes and moves on, every time.'
		},
		{
			value: '183',
			text: 'sessions where I typed some version of "don\'t make any changes yet". I repeat myself too.'
		}
	];
</script>

<section id="pain">
	<div class="wrap">
		<p class="eyebrow">Same mistake again</p>
		<h2>Every session starts from zero.</h2>
		<p class="lead">
			The agent hits an error, works it out and moves on. The next session hits the same error and
			takes the same detour. People on Reddit and X describe it better than I can.
		</p>

		<div class="quotes">
			{#each quotes as quote (quote.text)}
				<figure class="quote">
					<p>{quote.text}</p>
					<figcaption>{quote.source}</figcaption>
				</figure>
			{/each}
		</div>
		<p class="quotes-note">Real comments from public Reddit and X posts, trimmed for length.</p>

		<div class="numbers-head">
			<h3>So I counted, on my own history.</h3>
			<p>
				9 months of Claude Code and Codex logs from my laptop. 3,033 Codex sessions, about 19K tool
				errors.
			</p>
		</div>
		<div class="stats">
			{#each stats as stat (stat.value)}
				<div class="stat">
					<b>{stat.value}</b>
					<p>{stat.text}</p>
				</div>
			{/each}
		</div>
	</div>
</section>

<style>
	/* quotes */
	.quotes {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 16px;
		margin-top: 48px;
	}

	.quote {
		margin: 0;
		background: var(--card);
		border: 1px solid var(--rule);
		border-radius: 12px;
		padding: 22px 22px 18px;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: 18px;
	}

	.quote p {
		font: 400 20px/1.35 var(--serif);
		margin: 0;
	}

	.quote p::before {
		content: '\201C';
		color: var(--accent);
		margin-right: 1px;
	}

	.quote p::after {
		content: '\201D';
		color: var(--accent);
	}

	.quote figcaption {
		font: 400 12.5px/1.4 var(--mono);
		color: var(--ink-3);
	}

	.quotes-note {
		color: var(--ink-3);
		font-size: 14px;
		margin-top: 14px;
	}

	/* numbers */
	.numbers-head {
		margin-top: 80px;
		display: flex;
		justify-content: space-between;
		align-items: end;
		gap: 24px;
		flex-wrap: wrap;
	}

	.numbers-head h3 {
		max-width: 26ch;
	}

	.numbers-head p {
		color: var(--ink-3);
		font-size: 15px;
		margin: 0;
		max-width: 44ch;
	}

	.stats {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		margin-top: 28px;
		border-top: 1px solid var(--ink);
	}

	.stat {
		padding: 22px 22px 6px 0;
	}

	.stat + .stat {
		padding-left: 22px;
		border-left: 1px solid var(--rule);
	}

	.stat b {
		display: block;
		font: 400 clamp(40px, 5vw, 58px)/1 var(--serif);
		letter-spacing: -0.02em;
	}

	.stat p {
		color: var(--ink-2);
		font-size: 15.5px;
		margin: 12px 0 0;
	}

	@media (max-width: 980px) {
		.quotes,
		.stats {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}

		.stat:nth-child(3) {
			padding-left: 0;
			border-left: 0;
		}

		.stat:nth-child(n + 3) {
			border-top: 1px solid var(--rule);
			margin-top: 18px;
		}
	}

	@media (max-width: 520px) {
		.quotes,
		.stats {
			grid-template-columns: minmax(0, 1fr);
		}

		.stat,
		.stat + .stat {
			padding-left: 0;
			border-left: 0;
		}

		.stat + .stat {
			border-top: 1px solid var(--rule);
			margin-top: 18px;
		}
	}
</style>
