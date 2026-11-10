<script lang="ts">
	import {
		captionColors,
		captionLanguages,
		captionPositions,
		captionStyles,
		watermarkFileUrl,
		watermarkPositions,
		type PresetInput,
		type Vignette,
		type Watermark
	} from '$lib/api';
	import { captionStyleLabel, languageName } from '$lib/format';
	import { cornerLabel, watermarkOpacities, watermarkPreviewStyle, watermarkSizes } from '$lib/presets';

	// A preset being made or changed (docs/adr/0030, 0031): its captions,
	// openings and closings, and watermark, with a preview of the watermark
	// in both shapes.
	let {
		initial,
		vignettes,
		watermarks,
		onsave,
		oncancel
	}: {
		initial: PresetInput;
		vignettes: Vignette[];
		watermarks: Watermark[];
		onsave: (input: PresetInput) => Promise<void>;
		oncancel: () => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	let form = $state<PresetInput>({ ...initial });
	let saving = $state(false);
	let formError = $state('');
	const capitalize = (s: string) => s[0].toUpperCase() + s.slice(1);

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		formError = '';
		try {
			await onsave(form);
		} catch (err) {
			formError = (err as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<form class="card preset-form" onsubmit={save} data-testid="preset-form">
	<label class="wide">
		Name
		<input bind:value={form.name} maxlength="60" required placeholder="e.g. My channel" data-testid="preset-name" />
	</label>
	<fieldset>
		<legend>Captions</legend>
		<label class="check">
			<input type="checkbox" bind:checked={form.captions} data-testid="preset-captions" /> Burn captions in
		</label>
		{#if form.captions}
			<div class="row">
				<label>
					Style
					<select bind:value={form.captionStyle} data-testid="preset-style">
						{#each captionStyles as s (s)}<option value={s}>{captionStyleLabel(s)}</option>{/each}
					</select>
				</label>
				<label>
					Position
					<select bind:value={form.captionPosition}>
						{#each captionPositions as p (p)}<option value={p}>{capitalize(p)}</option>{/each}
					</select>
				</label>
				<label>
					Color
					<select bind:value={form.captionColor} data-testid="preset-color">
						<option value="">The style's own</option>
						{#each captionColors as c (c.name)}<option value={c.name}>{capitalize(c.name)}</option>{/each}
					</select>
				</label>
				<label>
					Capitals
					<select
						value={form.captionCaps === null ? '' : String(form.captionCaps)}
						onchange={(e) => {
							const v = e.currentTarget.value;
							form.captionCaps = v === '' ? null : v === 'true';
						}}
					>
						<option value="">The style's own</option>
						<option value="true">CAPS</option>
						<option value="false">Normal</option>
					</select>
				</label>
				<label>
					Language
					<select bind:value={form.captionLanguage} data-testid="preset-language">
						<option value="">Spoken language</option>
						{#each captionLanguages as l (l)}<option value={l}>{languageName(l)}</option>{/each}
					</select>
				</label>
			</div>
		{/if}
	</fieldset>
	{#each [{ legend: 'Opening', h: 'introHorizontalId', v: 'introVerticalId' }, { legend: 'Closing', h: 'outroHorizontalId', v: 'outroVerticalId' }] as const as slot (slot.legend)}
		<fieldset>
			<legend>{slot.legend}</legend>
			{#if vignettes.length === 0}
				<p class="muted small">Upload a vignette below to use it here.</p>
			{:else}
				<div class="row">
					<label>
						For 16:9 clips
						<select bind:value={form[slot.h]} data-testid="preset-{slot.legend.toLowerCase()}-horizontal">
							<option value={null}>None</option>
							{#each vignettes as v (v.id)}<option value={v.id}>{v.name}</option>{/each}
						</select>
					</label>
					<label>
						For 9:16 clips
						<select bind:value={form[slot.v]} data-testid="preset-{slot.legend.toLowerCase()}-vertical">
							<option value={null}>None</option>
							{#each vignettes as v (v.id)}<option value={v.id}>{v.name}</option>{/each}
						</select>
					</label>
				</div>
			{/if}
		</fieldset>
	{/each}
	<p class="muted small">
		With only one version, clips in the other shape show it whole over a blurred copy of itself.
	</p>
	<fieldset>
		<legend>Watermark</legend>
		{#if watermarks.length === 0}
			<p class="muted small">Upload an image below to use it here.</p>
		{:else}
			<div class="row">
				<label>
					Image
					<select bind:value={form.watermarkId} data-testid="preset-watermark">
						<option value={null}>None</option>
						{#each watermarks as w (w.id)}<option value={w.id}>{w.name}</option>{/each}
					</select>
				</label>
				{#if form.watermarkId}
					<label>
						Corner
						<select bind:value={form.watermarkPosition} data-testid="preset-watermark-position">
							{#each watermarkPositions as p (p)}<option value={p}>{cornerLabel(p)}</option>{/each}
						</select>
					</label>
					<label>
						Size
						<select bind:value={form.watermarkSize} data-testid="preset-watermark-size">
							{#each watermarkSizes as n (n)}<option value={n}>{n}%</option>{/each}
						</select>
					</label>
					<label>
						Opacity
						<select bind:value={form.watermarkOpacity} data-testid="preset-watermark-opacity">
							{#each watermarkOpacities as n (n)}<option value={n}>{n}%</option>{/each}
						</select>
					</label>
				{/if}
			</div>
			{#if form.watermarkId}
				<div class="previews" data-testid="watermark-preview">
					{#each [true, false] as vertical (vertical)}
						<div class="frame" class:vertical>
							<img src={watermarkFileUrl(form.watermarkId)} alt="" style={watermarkPreviewStyle(form, vertical)} />
							<span>{vertical ? '9:16' : '16:9'}</span>
						</div>
					{/each}
				</div>
				<p class="muted small">
					The size is a share of the clip's shorter side, so it looks the same in both shapes. It goes under the
					captions and not on the opening or closing.
				</p>
			{/if}
		{/if}
	</fieldset>
	{#if formError}<p class="error">{formError}</p>{/if}
	<div class="buttons">
		<button class="primary" disabled={saving} data-testid="preset-save">{saving ? 'Saving…' : 'Save preset'}</button>
		<button type="button" class="secondary" onclick={oncancel}>Cancel</button>
	</div>
</form>

<style>
	.previews {
		display: flex;
		align-items: end;
		gap: 1rem;
		margin-top: 0.6rem;
	}
	.frame {
		position: relative;
		width: 160px;
		aspect-ratio: 16 / 9;
		background: linear-gradient(135deg, #3b4252, #1f2430);
		border-radius: 6px;
		overflow: hidden;
	}
	.frame.vertical {
		width: 90px;
		aspect-ratio: 9 / 16;
	}
	.frame img {
		position: absolute;
		height: auto;
	}
	.frame span {
		position: absolute;
		left: 50%;
		bottom: 4px;
		transform: translateX(-50%);
		font-size: 0.7rem;
		color: #c8ccd4;
	}
	.small {
		font-size: 0.85rem;
		margin: 0.25rem 0 0;
	}
	.preset-form {
		display: grid;
		gap: 0.75rem;
		margin-bottom: 0.75rem;
	}
	fieldset {
		border: 1px solid var(--border);
		border-radius: 8px;
		padding: 0.5rem 0.75rem 0.75rem;
		margin: 0;
		min-width: 0;
	}
	legend {
		font-weight: 600;
		padding: 0 0.25rem;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.6rem 1rem;
		margin-top: 0.4rem;
	}
	label {
		display: grid;
		gap: 0.2rem;
		font-size: 0.9rem;
		min-width: 0;
	}
	label.check {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	input:not([type='checkbox']):not([type='file']),
	select {
		font: inherit;
		padding: 0.35rem 0.5rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: inherit;
		max-width: 100%;
	}
	.buttons {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}
</style>
