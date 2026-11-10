<script lang="ts">
	import type { AssetDetail, Job } from '$lib/api';

	// Where the player goes until the video is playable: the thumbnail, with
	// how processing goes, or why it failed.
	let { asset, hlsJob, text }: { asset: AssetDetail; hlsJob?: Job; text: string } = $props();
</script>

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
			<p data-testid="encoding-status">{text}</p>
		{/if}
	</div>
</div>

<style>
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
</style>
