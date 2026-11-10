<script lang="ts">
	import { clipLimits } from '$lib/api';
	import { formatClockTenths } from '$lib/format';
	import { quoteProblem, quoteTitle } from '$lib/quote';

	// The quote picked in the transcript, about to become a clip
	// (docs/adr/0039): what was said, its range, a title, and the buttons.
	let {
		text,
		start,
		end,
		busy = false,
		onpreview,
		oncreate,
		oncancel
	}: {
		/** What was said; a new quote resets the title to it. */
		text: string;
		start: number;
		end: number;
		/** Another job of the video is pending or running: the clip must wait. */
		busy?: boolean;
		onpreview?: (start: number, end: number) => void;
		oncreate: (title: string) => Promise<void>;
		oncancel: () => void;
	} = $props();

	let title = $state('');
	let saving = $state(false);
	let error = $state('');

	$effect(() => {
		title = quoteTitle(text);
		error = '';
	});

	let problem = $derived(quoteProblem(start, end) || (title.trim() === '' ? 'The title cannot be empty.' : ''));

	async function create() {
		saving = true;
		error = '';
		try {
			await oncreate(title.trim());
		} catch (e) {
			error = (e as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<div class="bar" data-testid="quote-bar">
	<p class="quote">“{text}”</p>
	<p class="muted when" data-testid="quote-range">
		{formatClockTenths(start)}–{formatClockTenths(end)} · {(end - start).toFixed(1)} s
	</p>
	<label class="title">
		Title
		<input bind:value={title} maxlength={clipLimits.maxTitle} data-testid="quote-title" />
	</label>
	{#if problem}<p class="error" data-testid="quote-problem">{problem}</p>{/if}
	{#if busy && !problem}<p class="muted note">Wait until the video's jobs finish to make the clip.</p>{/if}
	{#if error}<p class="error">{error}</p>{/if}
	<div class="buttons">
		<button class="secondary" onclick={() => onpreview?.(start, end)} data-testid="quote-preview">▶ Preview</button>
		<button class="primary" disabled={!!problem || busy || saving} onclick={create} data-testid="quote-create">
			{saving ? 'Making the clip…' : '✂ Make the clip'}
		</button>
		<button class="secondary" onclick={oncancel}>Cancel</button>
	</div>
</div>

<style>
	.bar {
		position: sticky;
		bottom: 0;
		display: grid;
		gap: 0.4rem;
		margin: 0.5rem 0.5rem 0;
		padding: 0.75rem;
		border: 1px solid var(--accent);
		border-radius: 8px;
		background: var(--surface);
		box-shadow: 0 -4px 12px rgb(0 0 0 / 0.08);
	}
	p {
		margin: 0;
	}
	.quote {
		font-weight: 600;
		display: -webkit-box;
		-webkit-line-clamp: 3;
		line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.when,
	.note {
		font-size: 0.85rem;
	}
	.title {
		display: grid;
		gap: 0.2rem;
		font-size: 0.85rem;
	}
	.title input {
		font: inherit;
		font-size: 1rem;
		padding: 0.4rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--surface);
		color: var(--text);
	}
	.error {
		font-size: 0.9rem;
	}
	.buttons {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		align-items: center;
	}
</style>
