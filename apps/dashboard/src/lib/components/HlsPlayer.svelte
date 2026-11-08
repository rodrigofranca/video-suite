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

	// Qualities, when hls.js plays a stream with several (docs/adr/0028).
	type Level = { index: number; height: number; label: string };
	let levels = $state<Level[]>([]);
	let chosen = $state(-1); // -1: automatic, by the connection
	let playing = $state(''); // the quality being played
	let player: { currentLevel: number } | undefined;

	function choose(index: number) {
		chosen = index;
		if (player) player.currentLevel = index; // -1 turns automatic switching back on
	}

	// End of the range playRange is playing, if any.
	let stopAt: number | null = null;

	/** Jump to a time (seconds) and play. */
	export function seek(seconds: number) {
		stopAt = null;
		video.currentTime = seconds;
		video.play().catch(() => {}); // autoplay may be blocked; the jump still happens
	}

	/** Play from start to end (seconds), then pause. */
	export function playRange(start: number, end: number) {
		seek(start);
		stopAt = end;
	}

	function timeUpdate() {
		if (stopAt !== null && video.currentTime >= stopAt) {
			stopAt = null;
			video.pause();
		}
		ontime?.(video.currentTime);
	}

	$effect(() => {
		const url = src;
		let destroyed = false;
		let hls: { destroy(): void } | undefined;
		error = '';

		import('hls.js').then(({ default: Hls }) => {
			if (destroyed) return;
			if (Hls.isSupported()) {
				const instance = new Hls();
				instance.on(Hls.Events.ERROR, (_event, data) => {
					if (data.fatal) error = `Playback error: ${data.details}`;
				});
				instance.on(Hls.Events.MANIFEST_PARSED, (_event, data) => {
					levels = data.levels
						.map((l, index) => {
							const short = Math.min(l.width, l.height);
							return { index, height: short, label: short ? `${short}p` : `${Math.round(l.bitrate / 1000)} kbps` };
						})
						.sort((a, b) => b.height - a.height);
				});
				instance.on(Hls.Events.LEVEL_SWITCHED, (_event, data) => {
					playing = levels.find((l) => l.index === data.level)?.label ?? '';
				});
				instance.loadSource(url);
				instance.attachMedia(video);
				hls = instance;
				player = instance;
			} else if (video.canPlayType('application/vnd.apple.mpegurl')) {
				video.src = url;
			} else {
				error = 'This browser cannot play HLS video.';
			}
		});

		return () => {
			destroyed = true;
			hls?.destroy();
			player = undefined;
			levels = [];
			chosen = -1;
			playing = '';
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
	ontimeupdate={timeUpdate}
	data-testid="player"
>
	{#each subtitles as sub, i (sub.url)}
		<track kind="subtitles" src={sub.url} srclang={sub.language} label={languageName(sub.language) + (sub.translatedFrom ? ' (translated)' : '')} default={i === 0} />
	{/each}
</video>
{#if levels.length > 1}
	<label class="quality">
		Quality
		<select value={chosen} onchange={(e) => choose(Number(e.currentTarget.value))} data-testid="quality">
			<option value={-1}>Auto{chosen === -1 && playing ? ` (${playing})` : ''}</option>
			{#each levels as l (l.index)}<option value={l.index}>{l.label}</option>{/each}
		</select>
	</label>
{/if}
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
	.quality {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin-top: 0.5rem;
		font-size: 0.9rem;
	}
	.quality select {
		font: inherit;
		padding: 0.2rem 0.3rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: inherit;
	}
</style>
