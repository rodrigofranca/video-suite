<script lang="ts">
	import type { Analysis } from '$lib/api';
	import { formatClock, languageName } from '$lib/format';

	let {
		analysis,
		currentTime = 0,
		onseek
	}: {
		analysis: Analysis;
		currentTime?: number;
		onseek?: (seconds: number) => void;
	} = $props();

	// The chapter being played: the last one that started before currentTime.
	let active = $derived(analysis.chapters.findLastIndex((c) => currentTime >= c.start));
</script>

<div class="card analysis" data-testid="analysis">
	<p class="summary">{analysis.summary}</p>

	{#if analysis.tags.length > 0}
		<ul class="tags" aria-label="Tags">
			{#each analysis.tags as tag (tag)}
				<li>{tag}</li>
			{/each}
		</ul>
	{/if}

	{#if analysis.chapters.length > 0}
		<h3>Chapters</h3>
		<ol class="chapters">
			{#each analysis.chapters as chapter, i (i)}
				<li class:active={i === active}>
					<button onclick={() => onseek?.(chapter.start)} title="Play from {formatClock(chapter.start)}">
						<span class="time">{formatClock(chapter.start)}</span>
						<span>{chapter.title}</span>
					</button>
				</li>
			{/each}
		</ol>
	{/if}

	<p class="muted meta">Written by {analysis.model} from the {languageName(analysis.language)} transcript</p>
</div>

<style>
	.analysis {
		display: grid;
		gap: 0.75rem;
	}
	.summary {
		margin: 0;
		line-height: 1.5;
	}
	.tags {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
	}
	.tags li {
		background: var(--badge-neutral-bg);
		border-radius: 999px;
		padding: 0.15rem 0.6rem;
		font-size: 0.85rem;
	}
	h3 {
		margin: 0.25rem 0 0;
		font-size: 0.95rem;
	}
	.chapters {
		list-style: none;
		margin: 0 -1rem;
		padding: 0;
	}
	button {
		all: unset;
		box-sizing: border-box;
		width: 100%;
		display: grid;
		grid-template-columns: 3.2rem 1fr;
		gap: 0.5rem;
		padding: 0.35rem 1rem;
		cursor: pointer;
		line-height: 1.4;
	}
	button:hover,
	button:focus-visible {
		background: var(--badge-neutral-bg);
	}
	.active button {
		background: var(--badge-busy-bg);
	}
	.time {
		color: var(--accent);
		font-variant-numeric: tabular-nums;
		font-size: 0.85rem;
		padding-top: 0.05rem;
	}
	.meta {
		margin: 0;
		font-size: 0.8rem;
	}
</style>
