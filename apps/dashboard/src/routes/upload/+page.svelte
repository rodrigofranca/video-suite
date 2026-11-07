<script lang="ts">
	import { goto } from '$app/navigation';
	import { uploadAsset } from '$lib/api';
	import { formatBytes } from '$lib/format';

	const MAX_BYTES = 2 * 1024 ** 3; // must match the API limit

	let file = $state<File | null>(null);
	let progress = $state<number | null>(null);
	let error = $state('');

	function pick(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		file = input.files?.[0] ?? null;
		error = '';
		if (file && !file.name.toLowerCase().endsWith('.mp4')) error = 'Only .mp4 files are accepted.';
		else if (file && file.size > MAX_BYTES) error = 'The file is larger than 2 GB.';
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (!file || error) return;
		progress = 0;
		try {
			const { assetId } = await uploadAsset(file, (p) => (progress = p));
			await goto(`/assets/${assetId}`);
		} catch (err) {
			error = (err as Error).message;
			progress = null;
		}
	}
</script>

<svelte:head><title>Upload · UltraMedia</title></svelte:head>

<h1>Upload a video</h1>
<p class="muted">MP4 up to 2 GB. Processing starts as soon as the upload finishes.</p>

<form class="card" onsubmit={submit}>
	<label class="drop">
		<input type="file" accept="video/mp4,.mp4" onchange={pick} disabled={progress !== null} data-testid="file" />
		{#if file}
			<span class="name">{file.name}</span>
			<span class="muted">{formatBytes(file.size)}</span>
		{:else}
			<span>Choose an MP4 file</span>
		{/if}
	</label>

	{#if progress !== null}
		<div class="progress" role="progressbar" aria-valuenow={Math.round(progress * 100)} aria-valuemin={0} aria-valuemax={100}>
			<div class="bar" style="width: {progress * 100}%"></div>
		</div>
		<p class="muted">{progress < 1 ? `Uploading… ${Math.round(progress * 100)}%` : 'Saving…'}</p>
	{/if}

	{#if error}
		<p class="error" data-testid="upload-error">{error}</p>
	{/if}

	<button class="primary" type="submit" disabled={!file || !!error || progress !== null}>Upload</button>
</form>

<style>
	h1 {
		margin-bottom: 0.25rem;
	}
	form {
		margin-top: 1rem;
		display: grid;
		gap: 1rem;
	}
	.drop {
		display: grid;
		gap: 0.25rem;
		justify-items: center;
		padding: 2rem 1rem;
		border: 2px dashed var(--border);
		border-radius: 10px;
		cursor: pointer;
		text-align: center;
	}
	.drop:hover {
		border-color: var(--accent);
	}
	.drop input {
		max-width: 100%;
	}
	.name {
		font-weight: 600;
		overflow-wrap: anywhere;
	}
	.progress {
		height: 8px;
		background: var(--badge-neutral-bg);
		border-radius: 999px;
		overflow: hidden;
	}
	.bar {
		height: 100%;
		background: var(--accent);
		transition: width 0.2s;
	}
	button {
		justify-self: start;
	}
	p {
		margin: 0;
	}
</style>
