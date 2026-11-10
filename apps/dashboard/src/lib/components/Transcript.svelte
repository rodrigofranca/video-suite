<script lang="ts">
	import type { Clip, Transcript } from '$lib/api';
	import { formatClock, languageName } from '$lib/format';
	import { findParts, pickUnit, quoteRange, quoteText, units, unitsBySegment, type Pick } from '$lib/quote';
	import { segmentAt } from '$lib/search';
	import QuoteBar from './QuoteBar.svelte';
	import type { QuoteActions } from './clip-types';

	// The transcript: each line plays from there; text can be found in it;
	// and a quote can be picked, word by word, to make a clip (docs/adr/0039).

	let {
		transcript,
		currentTime = 0,
		focus = null,
		onseek,
		quote
	}: {
		transcript: Transcript;
		currentTime?: number;
		/** A time to scroll the transcript to, such as a moment found by a search. */
		focus?: number | null;
		onseek?: (seconds: number) => void;
		/** Offers picking a quote to make a clip; left out where it is not offered. */
		quote?: QuoteActions;
	} = $props();

	let active = $derived(segmentAt(transcript.segments, currentTime));

	// Finding text in the transcript, without minding accents or case.
	let findText = $state('');
	let parts = $derived(transcript.segments.map((s) => findParts(s.text, findText)));
	let found = $derived(parts.flatMap((p, i) => (p ? [i] : [])));
	let at = $state(0);

	// Picking a quote: the words (or the segments, without word timings).
	let picking = $state(false);
	let pick = $state<Pick | null>(null);
	let made = $state<Clip | null>(null);
	let list = $derived(units(transcript.segments, transcript.words));
	let bySegment = $derived(unitsBySegment(transcript.segments, list));
	let range = $derived(pick?.done ? quoteRange(list, pick.from, pick.to, quote?.duration ?? 0) : null);
	const inPick = (u: number) => pick !== null && u >= pick.from && u <= pick.to;

	// Scroll the list, not the page, so a segment shows a third of the way down.
	let ol = $state<HTMLOListElement>();
	function scrollTo(segment: number) {
		const item = ol?.children[segment] as HTMLElement | undefined;
		if (ol && item) ol.scrollTop = item.offsetTop - ol.clientHeight / 3;
	}
	$effect(() => {
		if (!ol || focus === null) return;
		const t = focus;
		scrollTo(transcript.segments.findIndex((s) => s.end > t));
	});
	$effect(() => {
		// A new search goes to its first match.
		findText;
		at = 0;
	});
	$effect(() => {
		if (found.length > 0) scrollTo(found[Math.min(at, found.length - 1)]);
	});

	function step(by: number) {
		if (found.length > 0) at = (at + by + found.length) % found.length;
	}

	function togglePick() {
		picking = !picking;
		pick = null;
		made = null;
	}

	async function create(title: string) {
		if (!quote || !range) return;
		made = await quote.oncreate(range.start, range.end, title);
		picking = false;
		pick = null;
	}
</script>

<div class="card transcript" data-testid="transcript">
	<div class="top">
		<p class="muted meta">
			{languageName(transcript.language)} · Whisper {transcript.model} · {transcript.segments.length} segments
		</p>
		{#if transcript.segments.length > 0}
			<div class="tools">
				<input
					type="search"
					bind:value={findText}
					placeholder="Find in the transcript"
					aria-label="Find in the transcript"
					data-testid="transcript-find"
				/>
				{#if findText.trim()}
					<span class="muted count" data-testid="find-count"
						>{found.length ? `${Math.min(at, found.length - 1) + 1} of ${found.length}` : 'Not found'}</span
					>
					<button class="secondary" disabled={found.length < 2} onclick={() => step(-1)} aria-label="Previous">▲</button>
					<button class="secondary" disabled={found.length < 2} onclick={() => step(1)} aria-label="Next">▼</button>
				{/if}
				{#if quote}
					<button class="secondary" aria-pressed={picking} onclick={togglePick} data-testid="clip-quote">
						{picking ? 'Stop picking' : '✂ Clip a quote'}
					</button>
				{/if}
			</div>
			{#if picking}
				<p class="hint" data-testid="pick-hint">
					{pick?.done
						? 'Tap a word to start again.'
						: pick
							? 'Now tap the last word of the quote.'
							: transcript.words.length > 0
								? 'Tap the first word of the quote.'
								: 'This transcript has no word timings: tap the first sentence of the quote.'}
				</p>
			{/if}
		{/if}
	</div>
	{#if made}
		<p class="made" data-testid="quote-made">
			Clip made: <a href="#clip-{made.id}">“{made.title}”</a>, first in Clips.
		</p>
	{/if}
	{#if transcript.segments.length === 0}
		<p class="muted none">No speech found.</p>
	{:else}
		<ol bind:this={ol} class:picking>
			{#each transcript.segments as seg, i (i)}
				<li class:active={i === active && !picking} class:found={parts[i] !== null}>
					{#if picking}
						<div class="row">
							<span class="time">{formatClock(seg.start)}</span>
							<span class="text"
								>{#each bySegment[i] as u (u)}<button
										class="word"
										class:picked={inPick(u)}
										onclick={() => (pick = pickUnit(pick, u))}
										data-testid="word">{list[u].text}</button
									>{' '}{/each}</span
							>
						</div>
					{:else}
						<button class="line" onclick={() => onseek?.(seg.start)} title="Play from {formatClock(seg.start)}">
							<span class="time">{formatClock(seg.start)}</span>
							<span class="text"
								>{#if parts[i]}{#each parts[i]! as p, k (k)}{#if p.match}<mark>{p.text}</mark>{:else}{p.text}{/if}{/each}{:else}{seg.text}{/if}</span
							>
						</button>
					{/if}
				</li>
			{/each}
		</ol>
		{#if picking && pick && quote}
			{@const r = range ?? quoteRange(list, pick.from, pick.from, quote.duration)}
			<QuoteBar
				text={quoteText(list, pick.from, pick.to)}
				start={r.start}
				end={r.end}
				busy={quote.busy}
				onpreview={quote.onpreview}
				oncreate={create}
				oncancel={togglePick}
			/>
		{/if}
	{/if}
</div>

<style>
	.transcript {
		padding: 0.5rem 0;
	}
	.top {
		display: grid;
		gap: 0.4rem;
		margin: 0.25rem 1rem 0.5rem;
	}
	.meta {
		margin: 0;
		font-size: 0.85rem;
	}
	.tools {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.4rem;
	}
	.tools input {
		flex: 1 1 10rem;
		min-width: 0;
		font: inherit;
		padding: 0.35rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--surface);
		color: var(--text);
	}
	.count {
		font-size: 0.85rem;
		white-space: nowrap;
	}
	.hint,
	.made,
	.none {
		margin: 0;
		font-size: 0.9rem;
	}
	.made {
		margin: 0 1rem 0.5rem;
	}
	.none {
		margin: 0 1rem;
	}
	ol {
		list-style: none;
		margin: 0;
		padding: 0;
		max-height: 360px;
		overflow: auto;
		position: relative; /* items' offsetTop is from the list */
	}
	ol.picking {
		max-height: 420px;
	}
	.line {
		all: unset;
		box-sizing: border-box;
		width: 100%;
		cursor: pointer;
	}
	.line,
	.row {
		display: grid;
		grid-template-columns: 3.2rem 1fr;
		gap: 0.5rem;
		padding: 0.4rem 1rem;
		line-height: 1.4;
	}
	.line:hover,
	.line:focus-visible {
		background: var(--badge-neutral-bg);
	}
	.active .line {
		background: var(--badge-busy-bg);
	}
	.found .row {
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
	}
	.active mark {
		background: var(--badge-ok-bg);
	}
	.word {
		all: unset;
		cursor: pointer;
		border-radius: 4px;
		padding: 0.1rem 0;
		line-height: 1.7;
	}
	.word:hover,
	.word:focus-visible {
		background: var(--badge-neutral-bg);
		outline: 1px solid var(--border);
	}
	.word.picked {
		background: var(--accent);
		color: var(--accent-fg);
		box-shadow:
			0.15em 0 0 var(--accent),
			-0.15em 0 0 var(--accent);
	}
</style>
