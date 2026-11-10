<script lang="ts">
	// Deleting the video, asked first, at the bottom of its page (docs/adr/0027).
	let { filename, busy, ondelete }: { filename: string; busy: boolean; ondelete: () => Promise<void> } = $props();

	let confirming = $state(false);
	let deleting = $state(false);
	let error = $state('');

	async function remove() {
		deleting = true;
		error = '';
		try {
			await ondelete();
		} catch (e) {
			error = (e as Error).message;
			deleting = false;
		}
	}
</script>

<section class="danger-zone" data-testid="danger-zone">
	<h2>Delete video</h2>
	{#if confirming}
		<p>
			Delete <strong>{filename}</strong>, its clips and every stored file (original, HLS, thumbnail, subtitles,
			exports)? This cannot be undone.
		</p>
		<div class="buttons">
			<button class="danger" disabled={deleting || busy} onclick={remove} data-testid="delete-asset-yes"
				>{deleting ? 'Deleting…' : 'Yes, delete the video'}</button
			>
			<button class="secondary" onclick={() => (confirming = false)}>Cancel</button>
		</div>
	{:else}
		<button
			class="secondary"
			disabled={busy}
			title={busy ? 'Wait for the running jobs to finish' : ''}
			onclick={() => (confirming = true)}
			data-testid="delete-asset">🗑 Delete video…</button
		>
		{#if busy}<span class="muted">Available when no job is running.</span>{/if}
	{/if}
	{#if error}<p class="error">{error}</p>{/if}
</section>

<style>
	.danger-zone {
		margin-top: 2rem;
		padding-top: 1rem;
		border-top: 1px solid var(--border);
	}
	.danger-zone p {
		margin: 0 0 0.75rem;
	}
	.buttons {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}
</style>
