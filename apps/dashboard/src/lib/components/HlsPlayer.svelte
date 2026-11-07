<script lang="ts">
	// Plays an HLS playlist: hls.js where Media Source Extensions exist,
	// native HLS otherwise (Safari on iOS). UltraMedia replaces this later
	// (docs/adr/0012).
	import type { Subtitles } from '$lib/api';
	import { languageName } from '$lib/format';

	let {
		src,
		poster,
		subtitles = [],
		ontime
	}: {
		src: string;
		poster?: string | null;
		subtitles?: Subtitles[];
		/** Called with the current time, in seconds, while the video plays. */
		ontime?: (seconds: number) => void;
	} = $props();

	let video: HTMLVideoElement;
	let error = $state('');

	/** Jump to a time (seconds) and play. */
	export function seek(seconds: number) {
		video.currentTime = seconds;
		video.play().catch(() => {}); // autoplay may be blocked; the jump still happens
	}

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

<!-- Subtitles are sidecar WebVTT files read from storage, hence crossorigin. -->
<!-- svelte-ignore a11y_media_has_caption -->
<video
	bind:this={video}
	controls
	playsinline
	preload="metadata"
	crossorigin="anonymous"
	poster={poster ?? undefined}
	ontimeupdate={() => ontime?.(video.currentTime)}
	data-testid="player"
>
	{#each subtitles as sub, i (sub.url)}
		<track kind="subtitles" src={sub.url} srclang={sub.language} label={languageName(sub.language)} default={i === 0} />
	{/each}
</video>
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
