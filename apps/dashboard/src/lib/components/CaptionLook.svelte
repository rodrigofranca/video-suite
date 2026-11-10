<script lang="ts">
	import { captionColors, captionLanguages, captionPositions, type CaptionPosition } from '$lib/api';
	import type { ExportChoices } from '$lib/export-choices.svelte';
	import { languageName } from '$lib/format';

	// How a clip's captions look and their language (docs/adr/0025, 0029).
	let {
		choices,
		transcriptLanguage = '',
		translations = []
	}: {
		choices: ExportChoices;
		transcriptLanguage?: string;
		/** Languages the transcript is already translated into. */
		translations?: string[];
	} = $props();

	let otherLanguages = $derived(captionLanguages.filter((l) => l !== transcriptLanguage));
	const capitalize = (s: string) => s[0].toUpperCase() + s.slice(1);
</script>

<div class="look" data-testid="caption-look">
	{#if transcriptLanguage}
		<label>
			Language
			<select bind:value={choices.language} data-testid="caption-language">
				<option value="">{languageName(transcriptLanguage)} (original)</option>
				{#each otherLanguages as l (l)}<option value={l}>{languageName(l)}</option>{/each}
			</select>
		</label>
	{/if}
	<label>
		Position
		<select
			value={choices.position}
			onchange={(e) => (choices.position = e.currentTarget.value as CaptionPosition)}
			data-testid="caption-position"
		>
			{#each captionPositions as p (p)}<option value={p}>{capitalize(p)}</option>{/each}
		</select>
	</label>
	{#if choices.color}
		<div class="swatches" role="radiogroup" aria-label="Color" data-testid="caption-color">
			{#each captionColors as c (c.name)}
				<button
					class="swatch"
					class:on={choices.color === c.name}
					style="--swatch: {c.hex}"
					role="radio"
					aria-checked={choices.color === c.name}
					aria-label={capitalize(c.name)}
					title={capitalize(c.name)}
					onclick={() => (choices.color = c.name)}
					data-testid="color-{c.name}"
				></button>
			{/each}
		</div>
	{/if}
	<label class="caps">
		<input
			type="checkbox"
			checked={choices.caps}
			onchange={(e) => (choices.caps = e.currentTarget.checked)}
			data-testid="caption-caps"
		/>
		CAPS
	</label>
</div>
{#if choices.language && !translations.includes(choices.language)}
	<p class="muted hint" data-testid="translate-note">
		The first export in {languageName(choices.language)} translates the transcript with the language model; the next
		ones reuse it.
	</p>
{/if}

<style>
	.look {
		display: flex;
		align-items: center;
		gap: 0.6rem 1rem;
		flex-wrap: wrap;
		font-size: 0.9rem;
	}
	.look label {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.look select {
		font: inherit;
		padding: 0.2rem 0.3rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: inherit;
	}
	.swatches {
		display: flex;
		gap: 0.35rem;
	}
	.swatch {
		width: 1.6rem;
		height: 1.6rem;
		border-radius: 50%;
		border: 2px solid var(--border);
		background: var(--swatch);
		padding: 0;
		cursor: pointer;
	}
	.swatch.on {
		border-color: var(--text);
		box-shadow: 0 0 0 2px var(--surface) inset;
	}
	.caps input {
		width: 1.1rem;
		height: 1.1rem;
	}
	.hint {
		margin: 0 0 0.5rem;
		font-size: 0.85rem;
	}
</style>
