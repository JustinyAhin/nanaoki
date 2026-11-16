<script lang="ts">
	import { SITE } from '#lib/site.ts';
	import CommandLine from '../command-line.svelte';
	import InstallSwitcher from '../install-switcher.svelte';
</script>

<section id="install">
	<div class="wrap install">
		<div class="intro">
			<p class="eyebrow">Install</p>
			<h2>Get your own number in two minutes.</h2>
		</div>

		<div class="install-steps">
			<div class="step-block">
				<h3>1. Get the binary</h3>
				<p class="sub">
					macOS or Linux. On Windows, download the zip from the
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- External URL. -->
					<a href={SITE.releases}>releases page</a>.
				</p>
				<InstallSwitcher />
			</div>
			<div class="step-block">
				<h3>2. See your numbers</h3>
				<p class="sub">
					How many errors in the last third of your history were already fixed in the first two
					thirds.
				</p>
				<div class="cmds">
					<CommandLine command="nanaoki --split auto" />
				</div>
			</div>
			<div class="step-block">
				<h3>3. Stop the repeats</h3>
				<p class="sub">Install the hooks. They refresh the lessons once a day in the background.</p>
				<div class="cmds">
					<CommandLine command="nanaoki setup claude" />
					<CommandLine command="nanaoki setup codex" />
				</div>
				<p class="small">
					For Codex, trust the hooks in <code>/hooks</code>
					after setup.
				</p>
			</div>
		</div>
	</div>
</section>

<style>
	.install {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.3fr);
		gap: 56px;
		align-items: start;
	}

	.intro {
		position: sticky;
		top: 32px;
	}

	/* steps */
	.install-steps {
		border-top: 1px solid var(--ink);
	}

	.step-block {
		padding: 24px 0 32px;
	}

	.step-block + .step-block {
		border-top: 1px solid var(--rule);
	}

	.step-block:last-child {
		padding-bottom: 0;
	}

	.step-block h3 {
		font-size: 22px;
		margin-bottom: 6px;
	}

	.sub {
		color: var(--ink-2);
		font-size: 15.5px;
		margin-bottom: 16px;
		max-width: 56ch;
	}

	.cmds {
		display: flex;
		flex-direction: column;
		gap: 10px;
		max-width: 620px;
	}

	@media (max-width: 980px) {
		.install {
			grid-template-columns: minmax(0, 1fr);
			gap: 32px;
		}

		.intro {
			position: static;
		}
	}
</style>
