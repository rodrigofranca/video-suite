<script lang="ts">
	import { metadataUrl, type AssetDetail } from '$lib/api';
	import { formatBitrate, formatBytes, formatClock, formatDate, formatResolution, formatTokens, formatUsd } from '$lib/format';

	// What is known about a video: its media, its file, the AI it used.
	let { asset }: { asset: AssetDetail } = $props();
	let meta = $derived(asset.metadata ?? null);
</script>

<dl class="card details" data-testid="details">
	{#if meta}
		<dt>Duration</dt>
		<dd>{formatClock(meta.durationSeconds)}</dd>
		{#if meta.width}
			<dt>Resolution</dt>
			<dd>{formatResolution(meta.width, meta.height)}</dd>
		{/if}
		{#if asset.playback && asset.playback.renditions.length > 0}
			<dt>Qualities</dt>
			<dd data-testid="qualities">{asset.playback.renditions.map((r) => r.name).join(' · ')}</dd>
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
	<dt>AI usage</dt>
	<dd data-testid="ai-usage">
		{#if asset.aiUsage.calls === 0}
			<span class="muted">None yet</span>
		{:else}
			{formatTokens(asset.aiUsage.promptTokens + asset.aiUsage.completionTokens)} tokens · {formatUsd(
				asset.aiUsage.costUsd
			)}{asset.aiUsage.unpricedCalls > 0 ? ' + unpriced' : ''}
			<span class="muted usage-detail">
				{asset.aiUsage.byAction
					.map((a) => `${a.label || a.key} ${formatTokens(a.promptTokens + a.completionTokens)}`)
					.join(' · ')} · {asset.aiUsage.calls}
				{asset.aiUsage.calls === 1 ? 'call' : 'calls'}
			</span>
		{/if}
	</dd>
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

<style>
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
	.usage-detail {
		display: block;
		font-size: 0.85rem;
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
