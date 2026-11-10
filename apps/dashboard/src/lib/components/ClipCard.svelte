<script lang="ts">
	import { captionStyles, clipFormats, type Clip, type ClipFormat } from '$lib/api';
	import { presetExtras } from '$lib/clips';
	import { ExportChoices, type CaptionChoice } from '$lib/export-choices.svelte';
	import { captionStyleLabel, formatClock } from '$lib/format';
	import CaptionLook from './CaptionLook.svelte';
	import ClipEditor from './ClipEditor.svelte';
	import ExportFormat from './ExportFormat.svelte';
	import type { ClipActions, ClipContext } from './clip-types';

	// One clip, suggested or made from a quote: its title and range, editing
	// and deleting, how it will be exported, and its exports. Its choices
	// live here, with the clip.
	let { clip, context, actions }: { clip: Clip; context: ClipContext; actions: ClipActions } = $props();

	const choices = new ExportChoices(() => context);
	let editing = $state(false);
	let confirmingDelete = $state(false);
	let deleting = $state(false);
	let starting = $state<ClipFormat | ''>('');
	let error = $state('');
	let captionsOn = $derived(context.hasSubtitles || context.wordTimings);
	let extras = $derived(choices.preset ? presetExtras(choices.preset, context.vignettes, context.watermarks) : '');

	async function attempt(work: () => Promise<void>) {
		error = '';
		try {
			await work();
		} catch (e) {
			error = (e as Error).message;
		}
	}

	async function remove() {
		deleting = true;
		await attempt(async () => {
			await actions.ondelete?.(clip.id);
			confirmingDelete = false;
		});
		deleting = false;
	}

	async function exportAs(format: ClipFormat) {
		starting = format;
		const r = choices.request();
		await attempt(() => actions.onexport?.(clip.id, format, r.captions, r.style, r.options) ?? Promise.resolve());
		starting = '';
	}
</script>

<li class="card clip" id="clip-{clip.id}" data-origin={clip.origin}>
	<div class="head">
		{#if clip.origin === 'quote'}
			<span class="quote" title="Made from a quote in the transcript">Quote</span>
		{:else}
			<span class="rank">#{clip.rank}</span>
		{/if}
		<h3>{clip.title}</h3>
		{#if clip.editedAt}<span class="edited" title="Changed by hand">Edited</span>{/if}
		{#if clip.score !== null}
			<span class="score" title="How likely it is to hold attention, from the model">{clip.score}/10</span>
		{/if}
	</div>
	<p class="muted meta">{formatClock(clip.start)}–{formatClock(clip.end)} · {Math.round(clip.duration)} s</p>
	{#if clip.reason}<p class="reason">{clip.reason}</p>{/if}

	{#if confirmingDelete}
		<div class="confirm-delete" data-testid="delete-clip-confirm">
			<p>Delete this clip{clip.exports.some((e) => e.ready) ? ' and its exported files' : ''}? This cannot be undone.</p>
			<button class="danger" disabled={deleting || context.busy} onclick={remove} data-testid="delete-clip-yes"
				>{deleting ? 'Deleting…' : 'Yes, delete'}</button
			>
			<button class="secondary" onclick={() => (confirmingDelete = false)}>Cancel</button>
		</div>
	{/if}
	{#if editing && actions.onedit}
		{@const onedit = actions.onedit}
		<ClipEditor
			{clip}
			segments={context.segments}
			duration={context.duration}
			busy={context.busy}
			onpreview={actions.onpreview}
			onsave={async (changes) => {
				await onedit(clip.id, changes);
				editing = false;
			}}
			oncancel={() => (editing = false)}
		/>
	{/if}

	<div class="actions">
		<button class="secondary" onclick={() => actions.onpreview?.(clip.start, clip.end)}>▶ Preview</button>
		{#if actions.onedit && !editing}
			<button class="secondary" onclick={() => (editing = true)} data-testid="edit-clip">✎ Edit</button>
		{/if}
		{#if actions.ondelete && !confirmingDelete}
			<button
				class="secondary"
				disabled={context.busy}
				title={context.busy ? 'Wait for the running job to finish' : 'Delete this clip'}
				onclick={() => (confirmingDelete = true)}
				data-testid="delete-clip">🗑 Delete</button
			>
		{/if}
		{#if context.presets.length > 0}
			<label class="choice">
				Preset
				<select value={choices.presetId} onchange={(e) => choices.applyPreset(e.currentTarget.value)} data-testid="preset">
					<option value="">None</option>
					{#each context.presets as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
				</select>
			</label>
		{/if}
		{#if captionsOn}
			<label class="choice">
				Captions
				<select
					value={choices.captions}
					onchange={(e) => (choices.captions = e.currentTarget.value as CaptionChoice)}
					data-testid="caption-style"
				>
					{#each captionStyles as style (style)}
						<option value={style} disabled={!choices.available(style)}>{captionStyleLabel(style)}</option>
					{/each}
					<option value="off">Off</option>
				</select>
			</label>
		{/if}
	</div>
	{#if extras}
		<p class="muted hint" data-testid="preset-vignettes">Exports {extras}.</p>
	{/if}
	{#if captionsOn && choices.captions !== 'off'}
		<CaptionLook {choices} transcriptLanguage={context.transcriptLanguage} translations={context.translations} />
	{/if}

	<div class="formats">
		{#each clipFormats as format (format)}
			<ExportFormat
				{clip}
				{format}
				busy={context.busy}
				starting={starting === format}
				disabled={starting !== ''}
				tracking={context.tracking}
				translating={context.translating}
				onexport={() => exportAs(format)}
			/>
		{/each}
	</div>
	{#if clip.exports.some((e) => e.status === 'pending' || e.status === 'running')}<span class="sr-only"
			>Export running</span
		>{/if}
	{#if error}<p class="error">{error}</p>{/if}
</li>

<style>
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
	.quote {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--accent-fg);
		background: var(--accent);
		border-radius: 999px;
		padding: 0.1rem 0.5rem;
	}
	.clip {
		scroll-margin-top: 1rem;
	}
	.edited {
		font-size: 0.75rem;
		color: var(--badge-busy-fg);
		background: var(--badge-busy-bg);
		border-radius: 999px;
		padding: 0.1rem 0.45rem;
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
	.actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}
	.choice {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.9rem;
	}
	.choice select {
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
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.5rem;
		align-items: start;
	}
	@media (min-width: 560px) {
		.formats {
			grid-template-columns: repeat(4, minmax(0, 1fr));
		}
	}
	.error {
		margin: 0;
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
	}
</style>
