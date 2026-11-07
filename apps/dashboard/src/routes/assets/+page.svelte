<script lang="ts">
	import { listAssets, type Asset } from '$lib/api';
	import { formatClock, formatDate, formatResolution } from '$lib/format';
	import { poll } from '$lib/poll.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let assets = $state<Asset[] | null>(null);
	let error = $state('');

	poll(async () => {
		try {
			assets = await listAssets();
			error = '';
		} catch (e) {
			error = (e as Error).message;
		}
	});
</script>

<svelte:head><title>Videos · UltraMedia</title></svelte:head>

<div class="title">
	<h1>Videos</h1>
</div>

{#if error}
	<p class="card error">{error}</p>
{:else if assets === null}
	<p class="muted">Loading…</p>
{:else if assets.length === 0}
	<div class="card empty">
		<p>No videos yet.</p>
		<a class="button" href="/upload">Upload an MP4</a>
	</div>
{:else}
	<ul class="list">
		{#each assets as asset (asset.id)}
			<li>
				<a class="card item" href="/assets/{asset.id}" data-testid="asset-row">
					<div class="thumb">
						{#if asset.thumbnailUrl}
							<img src={asset.thumbnailUrl} alt="" loading="lazy" data-testid="thumbnail" />
						{/if}
						{#if asset.metadata?.durationSeconds}
							<span class="duration">{formatClock(asset.metadata.durationSeconds)}</span>
						{/if}
					</div>
					<div class="info">
						<span class="name">{asset.filename}</span>
						<span class="muted meta">
							{[formatResolution(asset.metadata?.width, asset.metadata?.height), formatDate(asset.createdAt)]
								.filter(Boolean)
								.join(' · ')}
						</span>
						<StatusBadge status={asset.status} />
					</div>
				</a>
			</li>
		{/each}
	</ul>
{/if}

<style>
	.title {
		margin-bottom: 1rem;
	}
	.empty {
		text-align: center;
		padding: 2rem 1rem;
	}
	.list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.5rem;
	}
	.item {
		display: grid;
		grid-template-columns: 8rem 1fr;
		gap: 0.85rem;
		align-items: center;
		padding: 0.6rem;
		color: inherit;
		text-decoration: none;
	}
	.item:hover {
		border-color: var(--accent);
	}
	.thumb {
		position: relative;
		aspect-ratio: 16 / 9;
		border-radius: 6px;
		overflow: hidden;
		background: var(--badge-neutral-bg);
	}
	.thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
	.duration {
		position: absolute;
		right: 0.3rem;
		bottom: 0.3rem;
		padding: 0 0.3rem;
		border-radius: 4px;
		background: rgb(0 0 0 / 0.75);
		color: #fff;
		font-size: 0.75rem;
		font-variant-numeric: tabular-nums;
	}
	.info {
		display: grid;
		gap: 0.2rem;
		justify-items: start;
		min-width: 0;
	}
	.name {
		font-weight: 600;
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.meta {
		font-size: 0.85rem;
	}
	@media (min-width: 640px) {
		.item {
			grid-template-columns: 10rem 1fr;
		}
	}
</style>
