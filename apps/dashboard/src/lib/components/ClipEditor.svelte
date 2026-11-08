<script lang="ts">
	import { clipLimits, type Clip, type TranscriptSegment } from '$lib/api';
	import { formatClockTenths } from '$lib/format';

	let {
		clip,
		segments = [],
		duration = 0,
		busy = false,
		onpreview,
		onsave,
		oncancel
	}: {
		clip: Clip;
		/** Transcript segments: the sentences the start and end can snap to. */
		segments?: TranscriptSegment[];
		/** The video's length in seconds, or 0 when unknown. */
		duration?: number;
		/** Another job of the video is pending or running: saving must wait. */
		busy?: boolean;
		onpreview?: (start: number, end: number) => void;
		onsave: (changes: { start?: number; end?: number; title?: string }) => Promise<void>;
		oncancel: () => void;
	} = $props();

	// The draft starts from the clip as it is when the editor opens.
	// svelte-ignore state_referenced_locally
	let start = $state(clip.start);
	// svelte-ignore state_referenced_locally
	let end = $state(clip.end);
	// svelte-ignore state_referenced_locally
	let title = $state(clip.title);
	let saving = $state(false);
	let error = $state('');

	const round = (t: number) => Math.round(t * 1000) / 1000;
	const limit = $derived(duration > 0 ? duration : Infinity);

	// Sentence boundaries: a start snaps to where a sentence starts, an end to where one ends.
	const starts = $derived([...new Set(segments.map((s) => s.start))].sort((a, b) => a - b));
	const ends = $derived([...new Set(segments.map((s) => s.end))].sort((a, b) => a - b));
	const before = (list: number[], t: number) => list.findLast((x) => x < t - 0.05);
	const after = (list: number[], t: number) => list.find((x) => x > t + 0.05);

	const length = $derived(end - start);
	const problem = $derived(
		start < 0 || end <= start
			? 'The start must be before the end.'
			: length < clipLimits.minSeconds
				? `A clip lasts at least ${clipLimits.minSeconds} s.`
				: length > clipLimits.maxSeconds
					? `A clip lasts at most ${clipLimits.maxSeconds / 60} minutes.`
					: end > limit + 0.05
						? 'The end is after the end of the video.'
						: title.trim() === ''
							? 'The title cannot be empty.'
							: ''
	);
	const changed = $derived(start !== clip.start || end !== clip.end || title.trim() !== clip.title);

	function move(which: 'start' | 'end', to: number | undefined) {
		if (to === undefined) return;
		const t = round(Math.min(Math.max(to, 0), limit));
		if (which === 'start') start = t;
		else end = t;
	}

	async function save() {
		saving = true;
		error = '';
		try {
			const changes: { start?: number; end?: number; title?: string } = {};
			if (start !== clip.start) changes.start = start;
			if (end !== clip.end) changes.end = end;
			if (title.trim() !== clip.title) changes.title = title.trim();
			await onsave(changes);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<div class="editor" data-testid="clip-editor">
	<label class="title">
		Title
		<input bind:value={title} maxlength={clipLimits.maxTitle} data-testid="edit-title" />
	</label>

	{#each [{ name: 'Start', key: 'start', list: starts }, { name: 'End', key: 'end', list: ends }] as const as b (b.key)}
		{@const t = b.key === 'start' ? start : end}
		<div class="bound" data-testid="edit-{b.key}">
			<span class="name">{b.name}</span>
			<output data-testid="edit-{b.key}-time">{formatClockTenths(t)}</output>
			<div class="nudge">
				<button
					class="secondary"
					title={`${b.key === 'start' ? 'Start' : 'End'} of the sentence before`}
					disabled={before(b.list, t) === undefined}
					onclick={() => move(b.key, before(b.list, t))}
					data-testid="{b.key}-prev">⏮ Prev sentence</button
				>
				<button
					class="secondary"
					title={`${b.key === 'start' ? 'Start' : 'End'} of the sentence after`}
					disabled={after(b.list, t) === undefined}
					onclick={() => move(b.key, after(b.list, t))}
					data-testid="{b.key}-next">Next sentence ⏭</button
				>
				<button class="secondary" onclick={() => move(b.key, t - 1)} data-testid="{b.key}-minus">−1 s</button>
				<button class="secondary" onclick={() => move(b.key, t + 1)} data-testid="{b.key}-plus">+1 s</button>
			</div>
		</div>
	{/each}

	<p class="muted length">
		{length > 0 ? `${length.toFixed(1)} s` : ''}{problem ? '' : changed ? ' · not saved yet' : ''}
	</p>
	{#if problem}<p class="error" data-testid="edit-problem">{problem}</p>{/if}
	{#if error}<p class="error">{error}</p>{/if}

	<div class="buttons">
		<button class="secondary" onclick={() => onpreview?.(start, end)}>▶ Preview</button>
		<button
			class="primary"
			onclick={save}
			disabled={!!problem || !changed || saving || busy}
			title={busy ? 'Wait for the running job to finish' : ''}
			data-testid="edit-save">{saving ? 'Saving…' : 'Save'}</button
		>
		<button class="secondary" onclick={oncancel}>Cancel</button>
	</div>
	{#if changed && (start !== clip.start || end !== clip.end)}
		<p class="muted note">Saving new times finds the faces again for this clip; earlier exports stay until you export again.</p>
	{/if}
</div>

<style>
	.editor {
		display: grid;
		gap: 0.6rem;
		padding: 0.75rem;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--surface);
	}
	.title {
		display: grid;
		gap: 0.25rem;
		font-size: 0.9rem;
	}
	.title input {
		font: inherit;
		padding: 0.4rem 0.5rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--bg);
		color: inherit;
		min-width: 0;
	}
	.bound {
		display: grid;
		grid-template-columns: 3rem auto;
		align-items: center;
		gap: 0.25rem 0.5rem;
	}
	.name {
		font-size: 0.9rem;
	}
	output {
		font-variant-numeric: tabular-nums;
		font-weight: 600;
	}
	.nudge {
		grid-column: 1 / -1;
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.35rem;
	}
	@media (min-width: 640px) {
		.nudge {
			grid-template-columns: 1.6fr 1.6fr 1fr 1fr;
		}
	}
	.nudge button {
		padding-inline: 0.3rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.length,
	.note,
	.error {
		margin: 0;
		font-size: 0.85rem;
	}
	.error {
		color: var(--badge-err-fg);
	}
	.buttons {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}
</style>
