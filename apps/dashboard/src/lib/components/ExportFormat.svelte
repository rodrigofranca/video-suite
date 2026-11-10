<script lang="ts">
	import { clipFileUrl, type Clip, type ClipFormat } from '$lib/api';
	import { describeExport, faceStatus } from '$lib/clips';
	import { clipFormatLabel, formatBytes, languageName } from '$lib/format';
	import { isActive } from '$lib/jobs';

	// One format of a clip: its export button, how the newest export went,
	// and the file once it is ready (docs/adr/0020).
	let {
		clip,
		format,
		busy = false,
		starting = false,
		disabled = false,
		tracking = false,
		translating = false,
		onexport
	}: {
		clip: Clip;
		format: ClipFormat;
		/** Another job of the video runs: exports wait. */
		busy?: boolean;
		/** This export is being asked for. */
		starting?: boolean;
		/** Another export of the clip is being asked for. */
		disabled?: boolean;
		tracking?: boolean;
		translating?: boolean;
		onexport: () => void;
	} = $props();

	let ex = $derived(clip.exports.find((e) => e.format === format));
	let active = $derived(isActive(ex));
</script>

<div class="format" data-testid="export-{format}">
	<button
		class="secondary"
		disabled={busy || active || disabled}
		title={busy ? 'Wait for the running job to finish' : `${ex?.ready ? 'Export again' : 'Export'} as ${clipFormatLabel(format)}`}
		onclick={onexport}
	>
		{starting ? 'Starting…' : clipFormatLabel(format)}
	</button>
	{#if format === 'vertical-face' && !ex}
		<span class="status">{faceStatus(clip, tracking)}</span>
	{/if}
	{#if ex && active && ex.status === 'pending' && ex.captionLanguage && translating}
		<span class="status" data-testid="translating">Translating into {languageName(ex.captionLanguage)}…</span>
	{:else if ex && active}
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
		<span class="status">{ex.sizeBytes ? formatBytes(ex.sizeBytes) : ''}{describeExport(ex)}</span>
	{/if}
</div>

<style>
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
	.status.warn {
		color: var(--badge-busy-fg);
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
	a {
		font-size: 0.8rem;
		overflow-wrap: anywhere;
	}
</style>
