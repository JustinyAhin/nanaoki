<script lang="ts">
	import { SITE } from '#lib/site.ts';
	import GithubMark from '../github-mark.svelte';
	import InstallSwitcher from '../install-switcher.svelte';
	import Terminal from '../terminal.svelte';
</script>

<!-- The terminal keeps whitespace, so the body line must stay as written. -->
<!-- prettier-ignore -->
{#snippet example()}<span class="h">Before the command runs</span> <span class="d">(PreToolUse hook)</span><span class="blk"><span class="g">●</span> Bash(<span class="c">ls src/routes/blog/[slug]</span>)</span><span class="ind"><span class="d">└</span> <span class="y">nanaoki guard:</span> [zsh_glob] zsh fails with "no matches found" when an unquoted glob matches no file. Suggested command: <span class="c">ls 'src/routes/blog/[slug]'</span></span><span class="blk gap"><span class="h">After a failure</span> <span class="d">(PostToolUse hook)</span></span><span class="blk"><span class="g">●</span> Bash(<span class="c">pnpm lint</span>)</span><span class="ind"><span class="d">└</span> <span class="r">error  Unexpected href link without resolve()</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] SvelteKit lint: wrap internal hrefs/goto with resolve() from $app/paths. Fix seen: <span class="c">bunx eslint &lt;5 files&gt;</span> <span class="d">(project-a, 2026-05-26)</span></span>{/snippet}

<div class="wrap hero">
	<h1>
		Your coding agent keeps making the same mistake.
		<span class="quiet">It fixed it last month,</span>
		<em>in another repo.</em>
	</h1>

	<div class="hero-grid">
		<div>
			<p class="lead">
				nanaoki reads the Claude Code and Codex history already on your machine and finds the errors
				your agents keep hitting. When one comes back, the agent gets the fix that worked last time.
			</p>
			<div class="cta">
				<InstallSwitcher />
				<div class="cta-row">
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- External URL. -->
					<a class="btn-secondary" href={SITE.repo}>
						<GithubMark />
						View on GitHub
					</a>
					<span>Runs locally with no LLM and no network calls. Free, MIT.</span>
				</div>
			</div>
		</div>

		<Terminal
			as="image"
			title="what the agent sees"
			label="Example of the messages the agent sees: a guard warning before an unquoted zsh glob, and a lesson after a lint failure that was fixed in another project."
		>
			{@render example()}
		</Terminal>
	</div>
</div>

<style>
	.hero {
		padding-block: 64px 104px;
	}

	h1 {
		font-size: clamp(34px, 3.2vw + 14px, 60px);
		line-height: 1.06;
		letter-spacing: -0.015em;
		max-width: 27ch;
	}

	.quiet {
		color: var(--ink-3);
	}

	em {
		font-style: normal;
		color: var(--accent-ink);
	}

	.hero-grid {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.1fr);
		gap: 56px;
		align-items: start;
		margin-top: 40px;
	}

	.lead {
		font-size: 19px;
		margin-top: 0;
		max-width: 46ch;
	}

	.cta {
		margin-top: 28px;
		display: flex;
		flex-direction: column;
		gap: 18px;
		align-items: stretch;
	}

	.cta-row {
		display: flex;
		gap: 8px 18px;
		align-items: center;
		flex-wrap: wrap;
		font-size: 15px;
		color: var(--ink-3);
	}

	.cta-row :global(svg) {
		width: 17px;
		height: 17px;
	}

	@media (max-width: 980px) {
		.hero-grid {
			grid-template-columns: minmax(0, 1fr);
			gap: 44px;
		}
	}

	@media (max-width: 720px) {
		.hero {
			padding-block: 24px 64px;
		}
	}
</style>
