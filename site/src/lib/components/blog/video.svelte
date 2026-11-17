<script lang="ts">
	type Props = {
		src: string;
		label: string;
		poster?: string;
		caption?: string;
		width?: 'text' | 'wide' | 'full';
	};

	let { src, label, poster, caption, width = 'text' }: Props = $props();
</script>

<figure class="video w-{width}">
	<!-- No autoplay, because the film has sound. No captions track: it has no speech, and all text is on screen. -->
	<!-- svelte-ignore a11y_media_has_caption -->
	<video {src} {poster} aria-label={label} controls playsinline preload="metadata"></video>
	{#if caption}<figcaption>{caption}</figcaption>{/if}
</figure>

<style>
	.video {
		margin: 2.2em 0;
	}

	video {
		display: block;
		width: 100%;
		aspect-ratio: 16 / 9;
		border: 1px solid var(--rule);
		border-radius: 10px;
		background: var(--card);
	}

	.w-full video {
		border-radius: 0;
		border-inline: 0;
	}

	figcaption {
		margin-top: 12px;
		font: 400 14px/1.5 var(--sans);
		color: var(--ink-3);
		max-width: 62ch;
		text-wrap: pretty;
	}

	.w-full figcaption {
		width: min(var(--text), 100% - 2 * var(--gutter));
		margin-inline: auto;
	}
</style>
