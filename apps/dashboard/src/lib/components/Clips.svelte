<script lang="ts">
	import { captionStyles, clipFileUrl, clipFormats, type CaptionStyle, type Clip, type ClipFormat } from '$lib/api';
	import { captionStyleLabel, clipFormatLabel, formatBytes, formatClock } from '$lib/format';

	let {
		clips,
		busy = false,
		hasSubtitles = false,
		wordTimings = false,
		onpreview,
		onexport
	}: {
		clips: Clip[];
		/** Another job of the video is pending or running: exports must wait. */
		busy?: boolean;
		/** Line captions need the subtitles. */
		hasSubtitles?: boolean;
		/** Word-by-word captions need the transcript's word timings. */
		wordTimings?: boolean;
		onpreview?: (start: number, end: number) => void;
		onexport?: (clipId: string, format: ClipFormat, captions: boolean, style: CaptionStyle) => Promise<void>;
	} = $props();

	type Choice = CaptionStyle | 'off';
	// Captions per clip: highlighted words when possible, else lines, else none.
	let choice = $state<Record<string, Choice>>({});
	let fallback = $derived<Choice>(wordTimings ? 'highlight' : hasSubtitles ? 'lines' : 'off');
	const available = (style: CaptionStyle) => (style === 'lines' ? hasSubtitles : wordTimings);
	const captionsOf = (clip: Clip): Choice => choice[clip.id] ?? fallback;
	let starting = $state(''); // "clipId:format" being queued
	let error = $state('');

	async function exportClip(clip: Clip, format: ClipFormat) {
		if (!onexport) return;
		starting = `${clip.id}:${format}`;
		error = '';
		try {
			const c = captionsOf(clip);
			await onexport(clip.id, format, c !== 'off', c === 'off' ? 'highlight' : c);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			starting = '';
		}
	}

	const exportOf = (clip: Clip, format: ClipFormat) => clip.exports.find((e) => e.format === format);
	const running = (clip: Clip) => clip.exports.some((e) => e.status === 'pending' || e.status === 'running');
</script>

{#if hasSubtitles && !wordTimings}
	<p class="muted hint">
		Captions that follow each word need a new transcription: run <strong>Transcription</strong> again below.
	</p>
{/if}
<ol class="clips" data-testid="clips">
	{#each clips as clip (clip.id)}
		<li class="card clip">
			<div class="head">
				<span class="rank">#{clip.rank}</span>
				<h3>{clip.title}</h3>
				<span class="score" title="How likely it is to hold attention, from the model">{clip.score}/10</span>
			</div>
			<p class="muted meta">
				{formatClock(clip.start)}–{formatClock(clip.end)} · {Math.round(clip.duration)} s
			</p>
			{#if clip.reason}<p class="reason">{clip.reason}</p>{/if}

			<div class="actions">
				<button class="secondary" onclick={() => onpreview?.(clip.start, clip.end)}>▶ Preview</button>
				{#if hasSubtitles || wordTimings}
					<label class="captions">
						Captions
						<select
							value={captionsOf(clip)}
							onchange={(e) => (choice[clip.id] = e.currentTarget.value as Choice)}
							data-testid="caption-style"
						>
							{#each captionStyles as style (style)}
								<option value={style} disabled={!available(style)}>{captionStyleLabel(style)}</option>
							{/each}
							<option value="off">Off</option>
						</select>
					</label>
				{/if}
			</div>

			<div class="formats">
				{#each clipFormats as format (format)}
					{@const ex = exportOf(clip, format)}
					{@const active = ex && (ex.status === 'pending' || ex.status === 'running')}
					<div class="format" data-testid="export-{format}">
						<button
							class="secondary"
							disabled={busy || !!active || starting !== ''}
							title={busy ? 'Wait for the running job to finish' : `${ex?.ready ? 'Export again' : 'Export'} as ${clipFormatLabel(format)}`}
							onclick={() => exportClip(clip, format)}
						>
							{starting === `${clip.id}:${format}` ? 'Starting…' : clipFormatLabel(format)}
						</button>
						{#if active}
							<span class="status" data-testid="exporting">Exporting…{ex.progress > 0 ? ` ${ex.progress}%` : ''}</span>
						{:else if ex?.status === 'failed'}
							<span class="status error" title={ex.error ?? ''}>Failed</span>
						{/if}
						{#if ex?.ready && !active}
							<!-- Captions, when chosen, are burned into the picture. -->
							<!-- svelte-ignore a11y_media_has_caption -->
							<video
								class:portrait={format !== 'original'}
								src={clipFileUrl(clip.id, format, true)}
								controls
								playsinline
								preload="metadata"
							></video>
							<a href={clipFileUrl(clip.id, format)} data-testid="download">Download</a>
							<span class="status">
								{ex.sizeBytes ? formatBytes(ex.sizeBytes) : ''}{ex.captions
									? ` · ${captionStyleLabel(ex.captionStyle ?? 'lines')}`
									: ''}
							</span>
						{/if}
					</div>
				{/each}
			</div>
			{#if running(clip)}<span class="sr-only">Export running</span>{/if}
		</li>
	{/each}
</ol>
{#if error}<p class="error">{error}</p>{/if}

<style>
	.clips {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.75rem;
	}
	.clip {
		display: grid;
		gap: 0.5rem;
	}
	.head {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
	}
	h3 {
		margin: 0;
		font-size: 1rem;
		flex: 1;
	}
	.rank {
		color: var(--accent);
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}
	.score {
		background: var(--badge-neutral-bg);
		border-radius: 999px;
		padding: 0.1rem 0.5rem;
		font-size: 0.8rem;
		white-space: nowrap;
	}
	.meta,
	.reason {
		margin: 0;
		font-size: 0.9rem;
	}
	.actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}
	.captions {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.9rem;
	}
	.captions select {
		font: inherit;
		padding: 0.2rem 0.3rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: inherit;
	}
	.hint {
		margin: 0 0 0.5rem;
		font-size: 0.85rem;
	}
	.formats {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 0.5rem;
		align-items: start;
	}
	.format {
		display: grid;
		gap: 0.35rem;
		justify-items: start;
	}
	.format button {
		width: 100%;
		white-space: nowrap;
		padding-inline: 0.4rem;
	}
	.status {
		font-size: 0.8rem;
		color: var(--muted);
	}
	.status.error {
		color: var(--badge-err-fg);
	}
	video {
		width: 100%;
		border-radius: 6px;
		background: #000;
		aspect-ratio: 16 / 9;
	}
	video.portrait {
		aspect-ratio: 9 / 16;
	}
	.format a {
		font-size: 0.8rem;
		overflow-wrap: anywhere;
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
	}
</style>
