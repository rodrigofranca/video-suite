<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { searchSpeech, type SearchResult } from '$lib/api';
	import { formatClock, formatDate, languageName } from '$lib/format';
	import { matchesText, maxSearchLength, momentLink, searchLink } from '$lib/search';

	// Where something was said, in every video with a transcript
	// (docs/adr/0038). The query lives in the URL, so a search can be shared
	// and the back button returns to it.
	let query = $derived(page.url.searchParams.get('q')?.trim() ?? '');
	let typed = $state('');
	let result = $state<SearchResult | null>(null);
	let loading = $state(false);
	let error = $state('');
	let asked = 0; // the latest request: an earlier answer that arrives late is dropped

	$effect(() => {
		typed = query;
	});

	$effect(() => {
		const q = query;
		result = null;
		error = '';
		if (q) run(q, 0);
	});

	async function run(q: string, offset: number) {
		const n = ++asked;
		loading = true;
		try {
			const r = await searchSpeech(q, offset);
			if (n !== asked) return;
			result = offset && result ? { ...r, videos: [...result.videos, ...r.videos] } : r;
			error = '';
		} catch (e) {
			if (n === asked) error = (e as Error).message;
		} finally {
			if (n === asked) loading = false;
		}
	}

	function submit(e: SubmitEvent) {
		e.preventDefault();
		goto(searchLink(typed), { keepFocus: true });
	}
</script>

<svelte:head><title>{query ? `${query} · ` : ''}Search · UltraMedia</title></svelte:head>

<h1>Search</h1>

<form class="search" role="search" onsubmit={submit}>
	<input
		type="search"
		bind:value={typed}
		maxlength={maxSearchLength}
		placeholder="What was said…"
		aria-label="Search what was said in the videos"
		enterkeyhint="search"
		data-testid="search-input"
	/>
	<button class="primary" type="submit" disabled={!typed.trim()}>Search</button>
</form>

{#if !query}
	<p class="muted tips">
		Finds where something was said, in every video with a transcript. Accents and word endings don't matter. Put
		<strong>"a phrase"</strong> in quotes, <strong>-word</strong> to leave a word out, and <strong>or</strong> between
		alternatives.
	</p>
{:else if error}
	<p class="card error">{error}</p>
{:else if !result}
	<p class="muted">Searching…</p>
{:else if result.videos.length === 0}
	<p class="card muted" data-testid="no-results">
		{#if result.searched === 0}
			No video has a transcript yet.
		{:else}
			Nothing found for “{result.query}” in {result.searched}
			{result.searched === 1 ? 'video' : 'videos'} with a transcript.
		{/if}
	</p>
{:else}
	{@const q = result.query}
	<ul class="results" data-testid="search-results">
		{#each result.videos as v (v.asset.id)}
			<li class="card video">
				<a class="head" href="/assets/{v.asset.id}">
					<div class="thumb">
						{#if v.asset.thumbnailUrl}<img src={v.asset.thumbnailUrl} alt="" loading="lazy" />{/if}
					</div>
					<div class="info">
						<span class="name">{v.asset.filename}</span>
						<span class="muted meta">{formatDate(v.asset.createdAt)} · {languageName(v.language)}</span>
					</div>
				</a>
				<ol class="moments">
					{#each v.moments as m (m.start)}
						<li>
							<a href={momentLink(v.asset.id, m.start, q)} data-testid="moment">
								<span class="time">{formatClock(m.start)}</span>
								<span class="text"
									>{#each m.parts as p, i (i)}{#if p.match}<mark>{p.text}</mark>{:else}{p.text}{/if}{/each}</span
								>
							</a>
						</li>
					{/each}
				</ol>
				{#if v.matches > v.moments.length}
					<p class="muted more-in">{matchesText(v.matches, v.moments.length)}</p>
				{/if}
			</li>
		{/each}
	</ul>
	{#if result.more}
		<button class="secondary more" onclick={() => run(q, result!.videos.length)} disabled={loading}>
			{loading ? 'Loading…' : 'More videos'}
		</button>
	{/if}
{/if}

<style>
	h1 {
		margin-bottom: 0.75rem;
	}
	.search {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1rem;
	}
	.search input {
		flex: 1;
		min-width: 0;
		font: inherit;
		padding: 0.5rem 0.75rem;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--surface);
		color: var(--text);
	}
	.tips {
		margin: 0;
	}
	.results {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.75rem;
	}
	.video {
		padding: 0.6rem 0 0.4rem;
	}
	.head {
		display: grid;
		grid-template-columns: 6rem 1fr;
		gap: 0.75rem;
		align-items: center;
		padding: 0 0.6rem 0.5rem;
		color: inherit;
		text-decoration: none;
	}
	.thumb {
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
	.info {
		display: grid;
		gap: 0.1rem;
		min-width: 0;
	}
	.name {
		font-weight: 600;
		overflow-wrap: anywhere;
	}
	.meta {
		font-size: 0.85rem;
	}
	.moments {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.moments a {
		display: grid;
		grid-template-columns: 3.2rem 1fr;
		gap: 0.5rem;
		padding: 0.45rem 0.6rem;
		color: inherit;
		text-decoration: none;
		line-height: 1.4;
	}
	.moments a:hover,
	.moments a:focus-visible {
		background: var(--badge-neutral-bg);
	}
	.time {
		color: var(--accent);
		font-variant-numeric: tabular-nums;
		font-size: 0.85rem;
		padding-top: 0.05rem;
	}
	mark {
		background: var(--badge-busy-bg);
		color: inherit;
		font-weight: 600;
		border-radius: 3px;
		padding: 0 0.1em;
	}
	.more-in {
		margin: 0.2rem 0.6rem 0.2rem 4.3rem;
		font-size: 0.85rem;
	}
	.more {
		margin-top: 1rem;
	}
</style>
