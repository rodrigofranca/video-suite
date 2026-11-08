<script lang="ts">
	import {
		captionColors,
		captionLanguages,
		captionPositions,
		captionStyles,
		clipFileUrl,
		clipFormats,
		styleColor,
		type CaptionColor,
		type CaptionOptions,
		type CaptionPosition,
		type CaptionStyle,
		type Clip,
		type ClipFormat,
		type TranscriptSegment
	} from '$lib/api';
	import { captionStyleLabel, clipFormatLabel, formatBytes, formatClock, languageName } from '$lib/format';
	import ClipEditor from './ClipEditor.svelte';

	let {
		clips,
		busy = false,
		hasSubtitles = false,
		wordTimings = false,
		tracking = false,
		translating = false,
		transcriptLanguage = '',
		translations = [],
		segments = [],
		duration = 0,
		onpreview,
		onexport,
		onedit,
		ondelete
	}: {
		clips: Clip[];
		/** Another job of the video is pending or running: exports must wait. */
		busy?: boolean;
		/** Line captions need the subtitles. */
		hasSubtitles?: boolean;
		/** Word-by-word captions need the transcript's word timings. */
		wordTimings?: boolean;
		/** A face tracking job is pending or running. */
		tracking?: boolean;
		/** A translation job is pending or running (an export waits for it). */
		translating?: boolean;
		/** The transcript's language, and the languages it has been translated into. */
		transcriptLanguage?: string;
		translations?: string[];
		/** Transcript segments, for snapping an edited clip to sentences. */
		segments?: TranscriptSegment[];
		/** The video's length in seconds, or 0 when unknown. */
		duration?: number;
		onpreview?: (start: number, end: number) => void;
		onexport?: (
			clipId: string,
			format: ClipFormat,
			captions: boolean,
			style: CaptionStyle,
			options: CaptionOptions
		) => Promise<void>;
		onedit?: (clipId: string, changes: { start?: number; end?: number; title?: string }) => Promise<void>;
		ondelete?: (clipId: string) => Promise<void>;
	} = $props();

	let editing = $state(''); // id of the clip being edited
	let confirmingDelete = $state(''); // id of the clip asking "delete?"
	let deleting = $state(false);

	async function removeClip(clip: Clip) {
		if (!ondelete) return;
		deleting = true;
		error = '';
		try {
			await ondelete(clip.id);
			confirmingDelete = '';
		} catch (e) {
			error = (e as Error).message;
		} finally {
			deleting = false;
		}
	}

	type Choice = CaptionStyle | 'off';
	// Captions per clip: highlighted words when possible, else lines, else none.
	let choice = $state<Record<string, Choice>>({});
	let fallback = $derived<Choice>(wordTimings ? 'highlight' : hasSubtitles ? 'lines' : 'off');
	const available = (style: CaptionStyle) => (style === 'lines' ? hasSubtitles : wordTimings);
	const captionsOf = (clip: Clip): Choice => choice[clip.id] ?? fallback;

	// Position, accent color and capitals per clip; each style brings its own
	// color and case until one is picked.
	let positionChoice = $state<Record<string, CaptionPosition>>({});
	let colorChoice = $state<Record<string, CaptionColor>>({});
	let capsChoice = $state<Record<string, boolean>>({});
	const positionOf = (clip: Clip) => positionChoice[clip.id] ?? 'bottom';
	const colorOf = (clip: Clip): CaptionColor | undefined => {
		const c = captionsOf(clip);
		return c === 'off' || !styleColor[c] ? undefined : (colorChoice[clip.id] ?? styleColor[c]);
	};
	const capsOf = (clip: Clip) => capsChoice[clip.id] ?? captionsOf(clip) === 'one-word';
	// The captions' language: the transcript's own ('') or a translation.
	let languageChoice = $state<Record<string, string>>({});
	const languageOf = (clip: Clip) => languageChoice[clip.id] ?? '';
	let otherLanguages = $derived(captionLanguages.filter((l) => l !== transcriptLanguage));
	const optionsOf = (clip: Clip): CaptionOptions => ({
		position: positionOf(clip),
		color: colorOf(clip),
		caps: capsOf(clip),
		language: languageOf(clip)
	});
	const capitalize = (p: string) => p[0].toUpperCase() + p.slice(1);
	let starting = $state(''); // "clipId:format" being queued
	let error = $state('');

	async function exportClip(clip: Clip, format: ClipFormat) {
		if (!onexport) return;
		starting = `${clip.id}:${format}`;
		error = '';
		try {
			const c = captionsOf(clip);
			await onexport(clip.id, format, c !== 'off', c === 'off' ? 'highlight' : c, optionsOf(clip));
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
				{#if clip.editedAt}<span class="edited" title="Changed by hand">Edited</span>{/if}
				<span class="score" title="How likely it is to hold attention, from the model">{clip.score}/10</span>
			</div>
			<p class="muted meta">
				{formatClock(clip.start)}–{formatClock(clip.end)} · {Math.round(clip.duration)} s
			</p>
			{#if clip.reason}<p class="reason">{clip.reason}</p>{/if}

			{#if confirmingDelete === clip.id}
				<div class="confirm-delete" data-testid="delete-clip-confirm">
					<p>Delete this clip{clip.exports.some((e) => e.ready) ? ' and its exported files' : ''}? This cannot be undone.</p>
					<button class="danger" disabled={deleting || busy} onclick={() => removeClip(clip)} data-testid="delete-clip-yes"
						>{deleting ? 'Deleting…' : 'Yes, delete'}</button
					>
					<button class="secondary" onclick={() => (confirmingDelete = '')}>Cancel</button>
				</div>
			{/if}
			{#if editing === clip.id && onedit}
				<ClipEditor
					{clip}
					{segments}
					{duration}
					{busy}
					{onpreview}
					onsave={async (changes) => {
						await onedit(clip.id, changes);
						editing = '';
					}}
					oncancel={() => (editing = '')}
				/>
			{/if}

			<div class="actions">
				<button class="secondary" onclick={() => onpreview?.(clip.start, clip.end)}>▶ Preview</button>
				{#if onedit && editing !== clip.id}
					<button class="secondary" onclick={() => (editing = clip.id)} data-testid="edit-clip">✎ Edit</button>
				{/if}
				{#if ondelete && confirmingDelete !== clip.id}
					<button
						class="secondary"
						disabled={busy}
						title={busy ? 'Wait for the running job to finish' : 'Delete this clip'}
						onclick={() => (confirmingDelete = clip.id)}
						data-testid="delete-clip">🗑 Delete</button
					>
				{/if}
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
			{#if (hasSubtitles || wordTimings) && captionsOf(clip) !== 'off'}
				<div class="look" data-testid="caption-look">
					{#if transcriptLanguage}
						<label>
							Language
							<select
								value={languageOf(clip)}
								onchange={(e) => (languageChoice[clip.id] = e.currentTarget.value)}
								data-testid="caption-language"
							>
								<option value="">{languageName(transcriptLanguage)} (original)</option>
								{#each otherLanguages as l (l)}<option value={l}>{languageName(l)}</option>{/each}
							</select>
						</label>
					{/if}
					<label>
						Position
						<select
							value={positionOf(clip)}
							onchange={(e) => (positionChoice[clip.id] = e.currentTarget.value as CaptionPosition)}
							data-testid="caption-position"
						>
							{#each captionPositions as p (p)}<option value={p}>{capitalize(p)}</option>{/each}
						</select>
					</label>
					{#if colorOf(clip)}
						<div class="swatches" role="radiogroup" aria-label="Color" data-testid="caption-color">
							{#each captionColors as c (c.name)}
								<button
									class="swatch"
									class:on={colorOf(clip) === c.name}
									style="--swatch: {c.hex}"
									role="radio"
									aria-checked={colorOf(clip) === c.name}
									aria-label={capitalize(c.name)}
									title={capitalize(c.name)}
									onclick={() => (colorChoice[clip.id] = c.name)}
									data-testid="color-{c.name}"
								></button>
							{/each}
						</div>
					{/if}
					<label class="caps">
						<input
							type="checkbox"
							checked={capsOf(clip)}
							onchange={(e) => (capsChoice[clip.id] = e.currentTarget.checked)}
							data-testid="caption-caps"
						/>
						CAPS
					</label>
				</div>
				{#if languageOf(clip) && !translations.includes(languageOf(clip))}
					<p class="muted hint" data-testid="translate-note">
						The first export in {languageName(languageOf(clip))} translates the transcript with the language model; the
						next ones reuse it.
					</p>
				{/if}
			{/if}

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
						{#if format === 'vertical-face' && !ex}
							<span class="status">
								{clip.faceCoverage === null
									? tracking
										? 'Finding faces…'
										: 'No face tracking yet: centred'
									: clip.faceCoverage < 0.3
										? 'Few faces found: centred'
										: clip.faceFollows === 'speaker'
											? `Follows who speaks · ${clip.facePeople} people`
											: `Face in ${Math.round(clip.faceCoverage * 100)}% of the clip`}
							</span>
						{/if}
						{#if active && ex.status === 'pending' && ex.captionLanguage && translating}
							<span class="status" data-testid="translating">Translating into {languageName(ex.captionLanguage)}…</span>
						{:else if active}
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
							{#if ex.outdated}
								<span class="status warn" data-testid="outdated">Made before the edit: export again</span>
							{/if}
							<span class="status">
								{ex.sizeBytes ? formatBytes(ex.sizeBytes) : ''}{ex.framing === 'center' ? ' · centred' : ''}{ex.captions
									? ` · ${captionStyleLabel(ex.captionStyle ?? 'lines')}` +
										(ex.captionPosition && ex.captionPosition !== 'bottom' ? `, ${ex.captionPosition}` : '') +
										(ex.captionColor && ex.captionColor !== styleColor[ex.captionStyle ?? 'lines'] ? `, ${ex.captionColor}` : '') +
										(ex.captionCaps && ex.captionStyle !== 'one-word' ? ', caps' : '') +
										(ex.translatedFrom && ex.captionLanguage ? `, ${languageName(ex.captionLanguage)}` : '')
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
	.confirm-delete {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem;
		padding: 0.6rem 0.75rem;
		border: 1px solid var(--badge-err-fg);
		border-radius: 8px;
	}
	.confirm-delete p {
		flex-basis: 100%;
		margin: 0;
		font-size: 0.9rem;
	}
	.edited {
		font-size: 0.75rem;
		color: var(--badge-busy-fg);
		background: var(--badge-busy-bg);
		border-radius: 999px;
		padding: 0.1rem 0.45rem;
	}
	.status.warn {
		color: var(--badge-busy-fg);
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
	.look {
		display: flex;
		align-items: center;
		gap: 0.6rem 1rem;
		flex-wrap: wrap;
		font-size: 0.9rem;
	}
	.look label {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.look select {
		font: inherit;
		padding: 0.2rem 0.3rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: inherit;
	}
	.swatches {
		display: flex;
		gap: 0.35rem;
	}
	.swatch {
		width: 1.6rem;
		height: 1.6rem;
		border-radius: 50%;
		border: 2px solid var(--border);
		background: var(--swatch);
		padding: 0;
		cursor: pointer;
	}
	.swatch.on {
		border-color: var(--text);
		box-shadow: 0 0 0 2px var(--surface) inset;
	}
	.caps input {
		width: 1.1rem;
		height: 1.1rem;
	}
	.hint {
		margin: 0 0 0.5rem;
		font-size: 0.85rem;
	}
	.formats {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.5rem;
		align-items: start;
	}
	@media (min-width: 560px) {
		.formats {
			grid-template-columns: repeat(4, minmax(0, 1fr));
		}
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
