<section id="privacy">
	<div class="wrap privacy">
		<div>
			<p class="eyebrow">Privacy</p>
			<h2>Your sessions stay on your machine.</h2>
			<p class="lead">
				Session logs have your code in them, and your clients' code. I didn't want mine sent
				anywhere, so nanaoki sends nothing.
			</p>
		</div>
		<ul class="checks">
			{#snippet check()}
				<svg
					viewBox="0 0 20 20"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					aria-hidden="true"
				>
					<path d="M4 10.5l4 4 8-9" stroke-linecap="round" stroke-linejoin="round" />
				</svg>
			{/snippet}
			<li>
				{@render check()}
				<div>
					<b>Local only.</b>
					<span>There is no account, no telemetry and no server.</span>
				</div>
			</li>
			<li>
				{@render check()}
				<div>
					<b>No LLM.</b>
					<span>Every count comes from regex rules you can read.</span>
				</div>
			</li>
			<li>
				{@render check()}
				<div>
					<b>No network.</b>
					<span>
						It makes no network calls. The one network step is <code>nanaoki setup claude</code>,
						which asks the
						<code>claude</code>
						CLI to fetch the plugin from GitHub.
					</span>
				</div>
			</li>
			<li>
				{@render check()}
				<div>
					<b>Read-only.</b>
					<span>
						It reads <code>~/.claude</code>
						and
						<code>~/.codex</code>
						and never changes them.
					</span>
				</div>
			</li>
			<li>
				{@render check()}
				<div>
					<b>One folder.</b>
					<span>
						It writes only to <code>~/.local/share/nanaoki/</code>. Delete that folder to remove
						everything.
					</span>
				</div>
			</li>
		</ul>
	</div>
</section>

<style>
	.privacy {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.2fr);
		gap: 56px;
		align-items: start;
	}

	.checks {
		list-style: none;
		margin: 0;
		padding: 0;
		border-top: 1px solid var(--ink);
	}

	.checks li {
		display: grid;
		grid-template-columns: 28px minmax(0, 1fr);
		gap: 10px;
		padding: 16px 0;
		border-bottom: 1px solid var(--rule);
	}

	.checks svg {
		width: 18px;
		height: 18px;
		margin-top: 4px;
		color: var(--ok);
	}

	.checks b {
		font-weight: 600;
	}

	.checks span {
		color: var(--ink-2);
	}

	@media (max-width: 980px) {
		.privacy {
			grid-template-columns: minmax(0, 1fr);
			gap: 32px;
		}
	}
</style>
