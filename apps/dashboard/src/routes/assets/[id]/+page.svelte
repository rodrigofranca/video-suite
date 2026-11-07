<script lang="ts">
	import { page } from '$app/state';
	import {
		getAnalysis,
		getAsset,
		getTranscript,
		metadataUrl,
		rerunJob,
		type Analysis,
		type AssetDetail,
		type Transcript
	} from '$lib/api';
	import { formatBitrate, formatBytes, formatClock, formatDate, formatResolution } from '$lib/format';
	import { poll } from '$lib/poll.svelte';
	import AnalysisPanel from '$lib/components/Analysis.svelte';
	import HlsPlayer from '$lib/components/HlsPlayer.svelte';
	import JobList from '$lib/components/JobList.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import TranscriptPanel from '$lib/components/Transcript.svelte';

	let asset = $state<AssetDetail | null>(null);
	let error = $state('');

	const finished = () =>
		(asset?.status === 'ready' || asset?.status === 'failed') &&
		!asset.jobs.some((j) => j.status === 'pending' || j.status === 'running');

	const updates = poll(
		async () => {
			const id = page.params.id!;
			try {
				asset = await getAsset(id);
				error = '';
			} catch (e) {
				error = (e as Error).message;
			}
		},
		() => !finished() && !error
	);

	let hlsJob = $derived(asset?.jobs.findLast((j) => j.type === 'transcode-hls'));

	// Transcript: fetched when the newest transcription job finishes.
	let player = $state<HlsPlayer>();
	let currentTime = $state(0);
	let transcript = $state<Transcript | null>(null);
	let transcriptMissing = $state(false);
	let transcriptFor = ''; // completedAt of the job the transcript was loaded for
	let transcribeJob = $derived(asset?.jobs.findLast((j) => j.type === 'transcribe-audio'));

	$effect(() => {
		const job = transcribeJob;
		if (!job || job.status !== 'done' || !job.completedAt || job.completedAt === transcriptFor) return;
		transcriptFor = job.completedAt;
		getTranscript(page.params.id!)
			.then((t) => {
				transcript = t;
				transcriptMissing = false;
			})
			.catch(() => {
				transcript = null;
				transcriptMissing = true; // no audio track, so nothing was transcribed
			});
	});

	// Summary, tags and chapters: fetched when the newest analysis job finishes.
	let analysis = $state<Analysis | null>(null);
	let analysisMissing = $state(false);
	let analysisFor = '';
	let analyzeJob = $derived(asset?.jobs.findLast((j) => j.type === 'analyze-transcript'));

	$effect(() => {
		const job = analyzeJob;
		if (!job || job.status !== 'done' || !job.completedAt || job.completedAt === analysisFor) return;
		analysisFor = job.completedAt;
		getAnalysis(page.params.id!)
			.then((a) => {
				analysis = a;
				analysisMissing = false;
			})
			.catch(() => {
				analysis = null;
				analysisMissing = true; // no speech, so nothing was analyzed
			});
	});

	async function rerun(type: string) {
		await rerunJob(page.params.id!, type);
		await updates.refresh();
	}
	let meta = $derived(asset?.metadata ?? null);
	let encodingText = $derived(
		hlsJob?.status === 'running'
			? hlsJob.progress > 0
				? `Encoding HLS… ${hlsJob.progress}%`
				: 'Encoding HLS…'
			: 'Processing… the video appears here when HLS encoding finishes.'
	);
</script>

<svelte:head><title>{asset?.filename ?? 'Video'} · UltraMedia</title></svelte:head>

<p class="back"><a href="/assets">← Videos</a></p>

{#if error}
	<p class="card error">{error}</p>
{:else if !asset}
	<p class="muted">Loading…</p>
{:else}
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
			<div
				class="placeholder card"
				class:with-image={!!asset.thumbnailUrl}
				style={asset.thumbnailUrl ? `background-image: url(${asset.thumbnailUrl})` : ''}
				data-testid={asset.status === 'failed' ? 'failed' : 'processing'}
			>
				<div class="overlay">
					{#if asset.status === 'failed'}
						<p class="error">This video could not be processed.</p>
						{#if hlsJob?.error}<p class="small">{hlsJob.error}</p>{/if}
					{:else}
						<span class="spinner" aria-hidden="true"></span>
						<p data-testid="encoding-status">{encodingText}</p>
					{/if}
				</div>
			</div>
		{/if}
	</section>

	{#if analyzeJob}
		<section class="analysis-section">
			<h2>Summary</h2>
			{#if analysis}
				<AnalysisPanel {analysis} {currentTime} onseek={(t) => player?.seek(t)} />
			{:else if analyzeJob.status === 'failed'}
				<p class="muted">The summary could not be written; open the Summary job below to see why.</p>
			{:else if analyzeJob.status === 'done' && analysisMissing}
				<p class="muted">No summary: nothing was said in the video.</p>
			{:else if analyzeJob.status === 'running'}
				<p class="muted" data-testid="summarizing">Writing the summary…</p>
			{:else}
				<p class="muted">The summary, tags and chapters appear here after the transcript.</p>
			{/if}
		</section>
	{/if}

	{#if transcribeJob}
		<section class="transcript-section">
			<h2>Transcript</h2>
			{#if transcript}
				<TranscriptPanel {transcript} {currentTime} onseek={(t) => player?.seek(t)} />
			{:else if transcribeJob.status === 'failed'}
				<p class="muted">Transcription failed; open the Transcription job below to see why.</p>
			{:else if transcribeJob.status === 'done' && transcriptMissing}
				<p class="muted">No transcript: the video has no audio track.</p>
			{:else if transcribeJob.status === 'running'}
				<p class="muted" data-testid="transcribing">
					Transcribing…{transcribeJob.progress > 0 ? ` ${transcribeJob.progress}%` : ''}
				</p>
			{:else}
				<p class="muted">The transcript appears here after HLS encoding and audio extraction.</p>
			{/if}
		</section>
	{/if}

	<div class="columns">
		<section>
			<h2>Processing jobs</h2>
			<JobList jobs={asset.jobs} onRerun={rerun} />
		</section>

		<section>
			<h2>Details</h2>
			<dl class="card details" data-testid="details">
				{#if meta}
					<dt>Duration</dt>
					<dd>{formatClock(meta.durationSeconds)}</dd>
					{#if meta.width}
						<dt>Resolution</dt>
						<dd>{formatResolution(meta.width, meta.height)}</dd>
					{/if}
					<dt>Video</dt>
					<dd>{meta.videoCodec}{meta.frameRate ? ` · ${meta.frameRate} fps` : ''}</dd>
					{#if meta.audioCodec}
						<dt>Audio</dt>
						<dd>{meta.audioCodec}</dd>
					{/if}
					{#if meta.bitRate}
						<dt>Bitrate</dt>
						<dd>{formatBitrate(meta.bitRate)}</dd>
					{/if}
				{/if}
				{#if asset.source}
					<dt>Size</dt>
					<dd>{formatBytes(asset.source.sizeBytes)}</dd>
				{/if}
				<dt>Uploaded</dt>
				<dd>{formatDate(asset.createdAt)}</dd>
				{#if asset.source}
					<dt>SHA-256</dt>
					<dd class="mono">{asset.source.checksum.slice(0, 16)}…</dd>
				{/if}
				{#if asset.playback || meta}
					<dt>Files</dt>
					<dd class="links">
						{#if asset.playback}
							<a href={asset.playback.manifestUrl} target="_blank" rel="noreferrer">HLS playlist</a>
						{/if}
						{#if meta}
							<a href={metadataUrl(asset.id)} target="_blank" rel="noreferrer">ffprobe JSON</a>
						{/if}
					</dd>
				{/if}
				<dt>Asset ID</dt>
				<dd class="mono">{asset.id}</dd>
			</dl>
		</section>
	</div>
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
	.analysis-section,
	.transcript-section {
		margin-bottom: 1.5rem;
	}
	.placeholder {
		aspect-ratio: 16 / 9;
		padding: 0;
		overflow: hidden;
		background-size: cover;
		background-position: center;
	}
	.overlay {
		height: 100%;
		display: grid;
		place-content: center;
		justify-items: center;
		gap: 0.75rem;
		padding: 1rem;
		text-align: center;
	}
	.with-image .overlay {
		background: rgb(0 0 0 / 0.55);
		color: #fff;
	}
	.overlay p {
		margin: 0;
		max-width: 28rem;
	}
	.small {
		font-size: 0.85rem;
		overflow-wrap: anywhere;
		opacity: 0.85;
	}
	.spinner {
		width: 28px;
		height: 28px;
		border: 3px solid var(--border);
		border-top-color: var(--accent);
		border-radius: 50%;
		animation: spin 0.9s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
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
	.details {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.4rem 1rem;
		margin: 0;
		font-size: 0.9rem;
	}
	dt {
		color: var(--muted);
	}
	dd {
		margin: 0;
		overflow-wrap: anywhere;
	}
	.links {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
	}
	.mono {
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		font-size: 0.8rem;
	}
</style>
