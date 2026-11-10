<script lang="ts">
	import type { Transcript } from '$lib/api';
	import { formatClock, languageName } from '$lib/format';
	import { segmentAt } from '$lib/search';

	let {
		transcript,
		currentTime = 0,
		focus = null,
		onseek
	}: {
		transcript: Transcript;
		currentTime?: number;
		/** A time to scroll the transcript to, such as a moment found by a search. */
		focus?: number | null;
		onseek?: (seconds: number) => void;
	} = $props();

	let active = $derived(segmentAt(transcript.segments, currentTime));

	// Scroll the list, not the page, so the moment shows a third of the way down.
	let list = $state<HTMLOListElement>();
	$effect(() => {
		if (!list || focus === null) return;
		const t = focus;
		const item = list.children[transcript.segments.findIndex((s) => s.end > t)] as HTMLElement | undefined;
		if (item) list.scrollTop = item.offsetTop - list.clientHeight / 3;
	});
</script>

<div class="card transcript" data-testid="transcript">
	<p class="muted meta">
		{languageName(transcript.language)} · Whisper {transcript.model} · {transcript.segments.length} segments
	</p>
	{#if transcript.segments.length === 0}
		<p class="muted">No speech found.</p>
	{:else}
		<ol bind:this={list}>
			{#each transcript.segments as seg, i (i)}
				<li class:active={i === active}>
					<button onclick={() => onseek?.(seg.start)} title="Play from {formatClock(seg.start)}">
						<span class="time">{formatClock(seg.start)}</span>
						<span class="text">{seg.text}</span>
					</button>
				</li>
			{/each}
		</ol>
	{/if}
</div>

<style>
	.transcript {
		padding: 0.5rem 0;
	}
	.meta {
		margin: 0.25rem 1rem 0.5rem;
		font-size: 0.85rem;
	}
	ol {
		list-style: none;
		margin: 0;
		padding: 0;
		max-height: 360px;
		overflow: auto;
		position: relative; /* items' offsetTop is from the list */
	}
	button {
		all: unset;
		box-sizing: border-box;
		width: 100%;
		display: grid;
		grid-template-columns: 3.2rem 1fr;
		gap: 0.5rem;
		padding: 0.4rem 1rem;
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
</style>
