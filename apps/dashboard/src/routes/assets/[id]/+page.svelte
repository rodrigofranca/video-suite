<script lang="ts">
	import { page } from '$app/state';
	import { getAsset, type AssetDetail } from '$lib/api';
	import { formatBytes, formatDate } from '$lib/format';
	import { poll } from '$lib/poll.svelte';
	import HlsPlayer from '$lib/components/HlsPlayer.svelte';
	import JobList from '$lib/components/JobList.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let asset = $state<AssetDetail | null>(null);
	let error = $state('');

	const finished = () => asset?.status === 'ready' || asset?.status === 'failed';

	poll(
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

	let hlsError = $derived(asset?.jobs.find((j) => j.type === 'transcode-hls')?.error ?? null);
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
			<HlsPlayer src={asset.playback.manifestUrl} />
		{:else if asset.status === 'failed'}
			<div class="placeholder card">
				<p class="error">This video could not be processed.</p>
				{#if hlsError}<p class="muted small">{hlsError}</p>{/if}
			</div>
		{:else}
			<div class="placeholder card" data-testid="processing">
				<span class="spinner" aria-hidden="true"></span>
				<p>Processing… the video appears here when HLS encoding finishes.</p>
			</div>
		{/if}
	</section>

	<div class="columns">
		<section>
			<h2>Processing jobs</h2>
			<JobList jobs={asset.jobs} />
		</section>

		<section>
			<h2>Details</h2>
			<dl class="card details">
				<dt>Uploaded</dt>
				<dd>{formatDate(asset.createdAt)}</dd>
				{#if asset.source}
					<dt>Size</dt>
					<dd>{formatBytes(asset.source.sizeBytes)}</dd>
					<dt>SHA-256</dt>
					<dd class="mono">{asset.source.checksum.slice(0, 16)}…</dd>
				{/if}
				{#if asset.playback}
					<dt>HLS playlist</dt>
					<dd class="mono"><a href={asset.playback.manifestUrl} target="_blank" rel="noreferrer">index.m3u8</a></dd>
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
	.placeholder {
		aspect-ratio: 16 / 9;
		display: grid;
		place-content: center;
		justify-items: center;
		gap: 0.75rem;
		text-align: center;
	}
	.placeholder p {
		margin: 0;
		max-width: 28rem;
	}
	.small {
		font-size: 0.85rem;
		overflow-wrap: anywhere;
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
	.mono {
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		font-size: 0.8rem;
	}
</style>
