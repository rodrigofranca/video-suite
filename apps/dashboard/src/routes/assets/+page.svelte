<script lang="ts">
	import { listAssets, type Asset } from '$lib/api';
	import { formatDate } from '$lib/format';
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
					<span class="name">{asset.filename}</span>
					<StatusBadge status={asset.status} />
					<span class="muted date">{formatDate(asset.createdAt)}</span>
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
		grid-template-columns: 1fr auto;
		grid-template-areas: 'name status' 'date date';
		gap: 0.25rem 0.75rem;
		align-items: center;
		color: inherit;
		text-decoration: none;
	}
	.item:hover {
		border-color: var(--accent);
	}
	.name {
		grid-area: name;
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.date {
		grid-area: date;
		font-size: 0.85rem;
	}
	@media (min-width: 640px) {
		.item {
			grid-template-columns: 1fr auto 12rem;
			grid-template-areas: 'name status date';
		}
		.date {
			text-align: right;
		}
	}
</style>
