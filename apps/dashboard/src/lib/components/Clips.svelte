<script lang="ts">
	import type { Clip } from '$lib/api';
	import ClipCard from './ClipCard.svelte';
	import type { ClipActions, ClipContext } from './clip-types';

	// The suggested clips, best first (docs/adr/0020). Each card keeps its own
	// export choices.
	let { clips, context, actions }: { clips: Clip[]; context: ClipContext; actions: ClipActions } = $props();
</script>

{#if context.hasSubtitles && !context.wordTimings}
	<p class="muted hint">
		Captions that follow each word need a new transcription: run <strong>Transcription</strong> again below.
	</p>
{/if}
<ol class="clips" data-testid="clips">
	{#each clips as clip (clip.id)}
		<ClipCard {clip} {context} {actions} />
	{/each}
</ol>

<style>
	.clips {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.75rem;
	}
	.hint {
		margin: 0 0 0.5rem;
		font-size: 0.85rem;
	}
</style>
