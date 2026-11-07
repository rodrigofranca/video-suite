<script lang="ts">
	// Plays an HLS playlist: hls.js where Media Source Extensions exist,
	// native HLS otherwise (Safari on iOS). UltraMedia replaces this later
	// (docs/adr/0012).
	let { src }: { src: string } = $props();

	let video: HTMLVideoElement;
	let error = $state('');

	$effect(() => {
		const url = src;
		let destroyed = false;
		let hls: { destroy(): void } | undefined;
		error = '';

		import('hls.js').then(({ default: Hls }) => {
			if (destroyed) return;
			if (Hls.isSupported()) {
				const player = new Hls();
				player.on(Hls.Events.ERROR, (_event, data) => {
					if (data.fatal) error = `Playback error: ${data.details}`;
				});
				player.loadSource(url);
				player.attachMedia(video);
				hls = player;
			} else if (video.canPlayType('application/vnd.apple.mpegurl')) {
				video.src = url;
			} else {
				error = 'This browser cannot play HLS video.';
			}
		});

		return () => {
			destroyed = true;
			hls?.destroy();
		};
	});
</script>

<!-- svelte-ignore a11y_media_has_caption -->
<video bind:this={video} controls playsinline preload="metadata" data-testid="player"></video>
{#if error}
	<p class="error">{error}</p>
{/if}

<style>
	video {
		width: 100%;
		aspect-ratio: 16 / 9;
		background: #000;
		border-radius: 8px;
		display: block;
	}
	.error {
		color: var(--badge-err-fg);
		margin-top: 0.5rem;
	}
</style>
