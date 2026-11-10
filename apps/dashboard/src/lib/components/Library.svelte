<script lang="ts" generics="T extends LibraryItem">
	import type { LibraryItem, LibraryKind } from '$lib/library';

	// One kind of file in the brand library: the upload form and the list
	// (docs/adr/0037). The kind says what it accepts and how it shows a file.
	let {
		kind,
		items,
		emptyText,
		onchange
	}: {
		kind: LibraryKind<T>;
		items: T[] | null;
		emptyText: string;
		/** After an upload or a delete: load the library again. */
		onchange: () => Promise<void>;
	} = $props();

	let file = $state<File | null>(null);
	let name = $state('');
	let note = $state('');
	let problem = $state('');
	let progress = $state<number | null>(null);
	let listError = $state('');
	let playing = $state(''); // id of the file playing

	async function pick(e: Event) {
		file = (e.currentTarget as HTMLInputElement).files?.[0] ?? null;
		problem = note = '';
		if (!file) return;
		problem = kind.check(file);
		if (!name) name = kind.nameOf(file);
		const picked = file;
		const found = await kind.inspect?.(picked);
		if (found && file === picked) {
			note = found.note;
			problem ||= found.problem;
		}
	}

	async function upload(e: SubmitEvent) {
		e.preventDefault();
		const form = e.currentTarget as HTMLFormElement; // gone from the event after the first await
		if (!file || problem) return;
		progress = 0;
		try {
			await kind.upload(name, file, (p) => (progress = p));
			file = null;
			name = note = '';
			form.reset();
			await onchange();
		} catch (err) {
			problem = (err as Error).message;
		} finally {
			progress = null;
		}
	}

	async function remove(id: string) {
		listError = '';
		try {
			await kind.remove(id);
			await onchange();
		} catch (e) {
			listError = (e as Error).message;
		}
	}
</script>

<form class="card upload" onsubmit={upload}>
	<label>
		Name
		<input bind:value={name} maxlength="60" placeholder={kind.namePlaceholder} data-testid="{kind.id}-name" />
	</label>
	<label class="file">
		File
		<input type="file" accept={kind.accept} onchange={pick} disabled={progress !== null} data-testid="{kind.id}-file" />
	</label>
	{#if note}<span class="muted small">{note}</span>{/if}
	<button class="primary" disabled={!file || !!problem || progress !== null} data-testid="{kind.id}-upload">
		{progress === null ? 'Upload' : `Uploading… ${Math.round(progress * 100)}%`}
	</button>
	{#if problem}<p class="error wide">{problem}</p>{/if}
</form>

{#if listError}<p class="error">{listError}</p>{/if}
{#if items && items.length > 0}
	<ul class="list" data-testid="{kind.id}s">
		{#each items as item (item.id)}
			<li class="card">
				<div class="item-head">
					{#if kind.preview === 'image'}<img class="thumb" src={kind.fileUrl(item.id)} alt="" />{/if}
					<strong>{item.name}</strong>
					<span class="muted small">{kind.details(item)}</span>
					<span class="spacer"></span>
					{#if kind.preview === 'video'}
						<button class="secondary" onclick={() => (playing = playing === item.id ? '' : item.id)}>
							{playing === item.id ? 'Hide' : '▶ Play'}
						</button>
					{/if}
					<button
						class="secondary"
						disabled={item.usedBy.length > 0}
						title={item.usedBy.length > 0 ? 'Used by a preset: take it out first' : `Delete this ${kind.id}`}
						onclick={() => remove(item.id)}>🗑 Delete</button
					>
				</div>
				{#if item.usedBy.length > 0}<p class="muted small">Used by {item.usedBy.join(', ')}</p>{/if}
				{#if playing === item.id}
					<!-- svelte-ignore a11y_media_has_caption -->
					<video src={kind.fileUrl(item.id)} controls playsinline autoplay></video>
				{/if}
			</li>
		{/each}
	</ul>
{:else if items}
	<p class="card muted">{emptyText}</p>
{/if}

<style>
	.upload {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.75rem;
		margin: 0.5rem 0 0.75rem;
	}
	label {
		display: grid;
		gap: 0.2rem;
		font-size: 0.9rem;
		min-width: 0;
	}
	input:not([type='file']) {
		font: inherit;
		padding: 0.35rem 0.5rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: inherit;
		max-width: 100%;
	}
	input[type='file'] {
		max-width: 100%;
		font-size: 0.85rem;
	}
	.wide {
		flex-basis: 100%;
		margin: 0;
	}
	.list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.6rem;
	}
	.item-head {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	.spacer {
		flex: 1;
	}
	.small {
		font-size: 0.85rem;
		margin: 0.25rem 0 0;
	}
	/* A checkerboard shows the image's transparency. */
	.thumb {
		height: 32px;
		max-width: 96px;
		object-fit: contain;
		border-radius: 4px;
		background: repeating-conic-gradient(#c8ccd4 0% 25%, #f1f2f5 0% 50%) 50% / 12px 12px;
	}
	video {
		display: block;
		width: 100%;
		max-height: 360px;
		margin-top: 0.5rem;
		border-radius: 8px;
		background: #000;
	}
</style>
