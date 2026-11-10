<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { AssetPage } from '$lib/asset-page.svelte';
	import AnalysisPanel from '$lib/components/Analysis.svelte';
	import AssetDetails from '$lib/components/AssetDetails.svelte';
	import ClipList from '$lib/components/Clips.svelte';
	import DeleteVideo from '$lib/components/DeleteVideo.svelte';
	import HlsPlayer from '$lib/components/HlsPlayer.svelte';
	import JobList from '$lib/components/JobList.svelte';
	import JobSection from '$lib/components/JobSection.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import TranscriptPanel from '$lib/components/Transcript.svelte';
	import VideoPlaceholder from '$lib/components/VideoPlaceholder.svelte';

	// The video page shows the state in AssetPage (docs/adr/0037); the parts
	// are components.
	const v = new AssetPage(() => page.params.id!);
	let player = $state<HlsPlayer>();
	let currentTime = $state(0);
	const seek = (t: number) => player?.seek(t);

	let clipContext = $derived({
		busy: v.busy,
		hasSubtitles: (v.asset?.subtitles.length ?? 0) > 0,
		wordTimings: v.clips.value?.wordTimings ?? false,
		transcriptLanguage: v.clips.value?.transcriptLanguage ?? '',
		translations: v.clips.value?.translations ?? [],
		tracking: v.trackingFaces,
		translating: v.translating,
		presets: v.presets,
		vignettes: v.vignettes,
		watermarks: v.watermarks,
		segments: v.transcript.value?.segments ?? [],
		duration: v.asset?.metadata?.durationSeconds ?? 0
	});
	const clipActions = {
		onpreview: (start: number, end: number) => player?.playRange(start, end),
		onexport: v.export.bind(v),
		onedit: v.editClip.bind(v),
		ondelete: v.deleteClip.bind(v)
	};

	async function deleteVideo() {
		await v.deleteVideo();
		await goto('/assets');
	}
</script>

<svelte:head><title>{v.asset?.filename ?? 'Video'} · UltraMedia</title></svelte:head>

<p class="back"><a href="/assets">← Videos</a></p>

{#if v.error}
	<p class="card error">{v.error}</p>
{:else if !v.asset}
	<p class="muted">Loading…</p>
{:else}
	{@const asset = v.asset}
	<div class="title">
		<h1>{asset.filename}</h1>
		<StatusBadge status={asset.status} />
	</div>

	<section class="player">
		{#if asset.playback}
			<HlsPlayer
				bind:this={player}
				src={asset.playback.manifestUrl}
				poster={asset.thumbnailUrl}
				subtitles={asset.subtitles}
				ontime={(t) => (currentTime = t)}
			/>
		{:else}
			<VideoPlaceholder {asset} hlsJob={v.hlsJob} text={v.encodingText} />
		{/if}
	</section>

	{#if v.analyzeJob}
		<JobSection
			class="analysis-section"
			title="Summary"
			job={v.analyzeJob}
			ready={!!v.analysis.value}
			empty={v.analysis.missing}
			failed="The summary could not be written; open the Summary job below to see why."
			emptyText="No summary: nothing was said in the video."
			runningText="Writing the summary…"
			runningId="summarizing"
			waiting="The summary, tags and chapters appear here after the transcript."
		>
			<AnalysisPanel analysis={v.analysis.value!} {currentTime} onseek={seek} />
		</JobSection>
	{/if}

	{#if v.suggestJob}
		<JobSection
			class="clips-section"
			title="Clips"
			job={v.suggestJob}
			ready={(v.clips.value?.clips.length ?? 0) > 0}
			empty={!!v.clips.value}
			failed="Clips could not be suggested; open the Clips job below to see why."
			emptyText="No clips: the video is too short, has no speech, or no moment works on its own."
			runningText="Finding the best moments…"
			runningId="suggesting"
			waiting="Suggested clips appear here after the summary."
		>
			<ClipList clips={v.clips.value!.clips} context={clipContext} actions={clipActions} />
		</JobSection>
	{/if}

	{#if v.transcribeJob}
		<JobSection
			class="transcript-section"
			title="Transcript"
			job={v.transcribeJob}
			ready={!!v.transcript.value}
			empty={v.transcript.missing}
			failed="Transcription failed; open the Transcription job below to see why."
			emptyText="No transcript: the video has no audio track."
			runningText={`Transcribing…${v.transcribeJob.progress > 0 ? ` ${v.transcribeJob.progress}%` : ''}`}
			runningId="transcribing"
			waiting="The transcript appears here after HLS encoding and audio extraction."
		>
			<TranscriptPanel transcript={v.transcript.value!} {currentTime} onseek={seek} />
		</JobSection>
	{/if}

	<div class="columns">
		<section>
			<h2>Processing jobs</h2>
			<JobList
				jobs={v.pipelineJobs}
				onRerun={(type) => v.rerun(type)}
				confirmRerun={v.resuggestNote ? { 'suggest-clips': v.resuggestNote } : {}}
			/>
		</section>
		<section>
			<h2>Details</h2>
			<AssetDetails {asset} />
		</section>
	</div>

	<DeleteVideo filename={asset.filename} busy={v.busy} ondelete={deleteVideo} />
{/if}

<style>
	.back {
		margin: 0 0 0.75rem;
	}
	.back a {
		text-decoration: none;
	}
	.title {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
		margin-bottom: 1rem;
	}
	h1 {
		overflow-wrap: anywhere;
	}
	.player {
		margin-bottom: 1.5rem;
	}
	.columns {
		display: grid;
		gap: 1.5rem;
	}
	@media (min-width: 760px) {
		.columns {
			grid-template-columns: 3fr 2fr;
		}
	}
</style>
