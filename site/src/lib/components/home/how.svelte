<script lang="ts">
	import hooks from '#lib/assets/screens/hooks.png';
	import reportRaf from '#lib/assets/screens/report-raf.png';
	import reportTop from '#lib/assets/screens/report-top.png';
	import search from '#lib/assets/screens/search.png';
</script>

<section id="how">
	<div class="wrap">
		<p class="eyebrow">How it works</p>
		<h2>It reads your old sessions and finds the fix that worked.</h2>
		<p class="lead">
			One Go binary with no model in it. It is regexes and counting, over logs that are already on
			your disk.
		</p>

		<ol class="steps">
			<li>
				<b>Read</b>
				<span>
					It reads the Claude Code and Codex sessions in <code>~/.claude</code>
					and
					<code>~/.codex</code>.
				</span>
			</li>
			<li>
				<b>Detect</b>
				<span>It finds the errors that keep coming back, and the later command that worked.</span>
			</li>
			<li>
				<b>Lesson</b>
				<span>
					Each fix becomes a sentence or two: what failed and what fixed it, with the project it
					came from.
				</span>
			</li>
			<li>
				<b>Guard</b>
				<span>
					Before a command that's known to fail, the agent gets a warning and the fixed command.
				</span>
			</li>
			<li>
				<b>Search</b>
				<span>Full-text search over your prompts, the agent's replies and the tool output.</span>
			</li>
		</ol>

		<div class="shots">
			<figure class="shot wide">
				<figcaption>
					<b>nanaoki --split auto</b>
					<h3>Your number, in one command</h3>
					<p>
						How many errors in the last third of your history had a cause that an earlier session
						already fixed, and in which project. (My 43% uses a fixed Feb-Aug / Sep-Nov split.
						<code>auto</code>
						picks its own thirds, so this screenshot says 42%.)
					</p>
				</figcaption>
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Links to the full-size image file. -->
				<a href={reportRaf}>
					<img
						src={reportRaf}
						width="1898"
						height="948"
						alt="nanaoki report, repeat-after-fix section: 2,454 of 5,849 error episodes hit a cause that was already fixed, with a table of the top repeated causes."
						loading="lazy"
						decoding="async"
					/>
				</a>
			</figure>
			<figure class="shot">
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Links to the full-size image file. -->
				<a href={hooks}>
					<img
						src={hooks}
						width="1898"
						height="788"
						alt="Guard and lesson messages that the agent sees before and after a command."
						loading="lazy"
						decoding="async"
					/>
				</a>
				<figcaption>
					<b>Plugin hooks</b>
					<h3>Warn before, remind after</h3>
					<p>
						Before a command that's known to fail, the agent gets the fixed one. After a failure, it
						gets the lesson from the session that fixed it, even if that was in another repo.
					</p>
				</figcaption>
			</figure>
			<figure class="shot">
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Links to the full-size image file. -->
				<a href={search}>
					<img
						src={search}
						width="1898"
						height="828"
						alt="nanaoki search results, grouped by session with highlighted matches."
						loading="lazy"
						decoding="async"
					/>
				</a>
				<figcaption>
					<b>nanaoki search</b>
					<h3>"Did we fix this before?"</h3>
					<p>
						Search your old sessions. Real answers rank above tool calls that only grep for the
						words. The past-sessions skill tells your agent to look here when you ask.
					</p>
				</figcaption>
			</figure>
			<figure class="shot wide">
				<figcaption>
					<b>nanaoki</b>
					<h3>The full report</h3>
					<p>
						Coverage, repeated errors, time lost, the rules you keep typing, a monthly trend, and a
						suggested fix for each top item. Treat the counts as estimates. <code>
							--format json
						</code>
						gives the file and line behind each number.
					</p>
				</figcaption>
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Links to the full-size image file. -->
				<a href={reportTop}>
					<img
						src={reportTop}
						width="1898"
						height="1748"
						alt="Top of the nanaoki report: coverage and the most repeated errors."
						loading="lazy"
						decoding="async"
					/>
				</a>
			</figure>
		</div>
		<p class="anon-note">
			Screenshots use <code>--anon</code>, which hides project names, session ids and home paths.
			Use it if you share your report.
		</p>
	</div>
</section>

<style>
	/* steps */
	.steps {
		list-style: none;
		padding: 0;
		margin: 48px 0 0;
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 0;
		counter-reset: s;
		border-top: 1px solid var(--ink);
	}

	.steps li {
		padding: 20px 18px 0 0;
		counter-increment: s;
	}

	.steps li + li {
		padding-left: 18px;
		border-left: 1px solid var(--rule);
	}

	.steps li::before {
		content: '0' counter(s);
		font: 500 12.5px/1 var(--mono);
		color: var(--accent-ink);
		display: block;
		margin-bottom: 12px;
	}

	.steps b {
		display: block;
		font: 500 15px/1.3 var(--mono);
		margin-bottom: 6px;
	}

	.steps span {
		color: var(--ink-2);
		font-size: 15px;
		display: block;
	}

	/* screenshots */
	.shots {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		gap: 56px 40px;
		margin-top: 72px;
	}

	.shot {
		margin: 0;
	}

	.shot.wide {
		grid-column: 1 / -1;
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 2fr);
		gap: 40px;
		align-items: center;
	}

	.shot img {
		border-radius: 12px;
		box-shadow: 0 24px 48px -28px rgba(20, 16, 10, 0.45);
		background: var(--term-bg);
	}

	.shot figcaption {
		margin-top: 18px;
	}

	.shot.wide figcaption {
		margin-top: 0;
		order: -1;
	}

	.shot figcaption b {
		font: 500 13px/1 var(--mono);
		color: var(--accent-ink);
		display: block;
		margin-bottom: 10px;
		letter-spacing: 0.02em;
	}

	.shot figcaption h3 {
		margin-bottom: 8px;
	}

	.shot figcaption p {
		color: var(--ink-2);
		font-size: 16px;
		margin: 0;
	}

	.anon-note {
		color: var(--ink-3);
		font-size: 14px;
		margin-top: 40px;
	}

	@media (max-width: 980px) {
		.steps {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			row-gap: 28px;
		}

		.steps li:nth-child(odd) {
			padding-left: 0;
			border-left: 0;
		}

		.shot.wide {
			grid-template-columns: minmax(0, 1fr);
			gap: 0;
		}

		.shot.wide figcaption {
			order: 0;
			margin-top: 18px;
		}
	}

	@media (max-width: 720px) {
		.shots {
			grid-template-columns: minmax(0, 1fr);
			gap: 48px;
			margin-top: 48px;
		}
	}

	@media (max-width: 520px) {
		.steps {
			grid-template-columns: minmax(0, 1fr);
		}

		.steps li + li {
			padding-left: 0;
			border-left: 0;
			border-top: 1px solid var(--rule);
		}
	}
</style>
