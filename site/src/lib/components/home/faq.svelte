<section id="faq">
	<div class="wrap">
		<h2>Questions</h2>
		<div class="faq">
			<details>
				<summary>Does it send my code anywhere?</summary>
				<div class="a">
					<p>
						No. nanaoki makes no network calls and there is no account. The index, the lessons and
						the report stay in <code>~/.local/share/nanaoki/</code>. The one network step is
						<code>nanaoki setup claude</code>, which asks the
						<code>claude</code>
						CLI to fetch the plugin from GitHub.
					</p>
				</div>
			</details>
			<details>
				<summary>Which agents does it work with?</summary>
				<div class="a">
					<p>
						Claude Code and Codex. It reads both histories, so a fix from a Codex session can show
						up in Claude Code. It runs on macOS, Linux and Windows.
					</p>
				</div>
			</details>
			<details>
				<summary>Does it slow my agent down?</summary>
				<div class="a">
					<p>
						A hook call takes about 15 ms at p95 on my laptop. If a hook fails, it fails open and
						the agent carries on. The hooks stay quiet on probes like <code>test -f</code>
						or a grep with no match.
					</p>
				</div>
			</details>
			<details>
				<summary>What does it cost?</summary>
				<div class="a"><p>Nothing. It's free and open source, under the MIT license.</p></div>
			</details>
			<details>
				<summary>Do I still need CLAUDE.md or AGENTS.md?</summary>
				<div class="a">
					<p>
						Yes, for stable decisions. In my replay the file beat the lessons on those.
						<code>nanaoki rules suggest</code>
						prints lines for your file from the instructions you keep typing. The suggestions can be noisy,
						so it only prints them. It never writes the file.
					</p>
				</div>
			</details>
			<details>
				<summary>My history is short. Is it worth it?</summary>
				<div class="a">
					<p>
						Run the report and look. If you use Claude Code, check one setting today: it deletes
						transcripts after 30 days by default. Set <code>"cleanupPeriodDays": 3650</code>
						in
						<code>~/.claude/settings.json</code>
						to keep them. The report gets more useful as the history grows.
					</p>
				</div>
			</details>
		</div>
	</div>
</section>

<style>
	.faq {
		margin-top: 40px;
		border-top: 1px solid var(--ink);
		max-width: 820px;
	}

	details {
		border-bottom: 1px solid var(--rule);
	}

	summary {
		cursor: pointer;
		list-style: none;
		padding: 20px 36px 20px 0;
		position: relative;
		font: 400 21px/1.3 var(--serif);
	}

	summary::-webkit-details-marker {
		display: none;
	}

	summary::after {
		content: '';
		position: absolute;
		right: 6px;
		top: 50%;
		width: 10px;
		height: 10px;
		border-right: 1.5px solid var(--ink-3);
		border-bottom: 1.5px solid var(--ink-3);
		transform: translateY(-70%) rotate(45deg);
		transition: transform 0.15s;
	}

	details[open] summary::after {
		transform: translateY(-30%) rotate(-135deg);
	}

	.a {
		padding: 0 0 22px;
		color: var(--ink-2);
		max-width: 68ch;
	}

	.a p:last-child {
		margin-bottom: 0;
	}
</style>
