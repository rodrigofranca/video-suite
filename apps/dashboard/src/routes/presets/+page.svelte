<script lang="ts">
	import {
		captionColors,
		captionLanguages,
		captionPositions,
		captionStyles,
		createPreset,
		deletePreset,
		deleteVignette,
		listPresets,
		listVignettes,
		updatePreset,
		uploadVignette,
		vignetteFileUrl,
		vignetteLimits,
		type Preset,
		type PresetInput,
		type Vignette
	} from '$lib/api';
	import { captionStyleLabel, describeCaptions, formatBytes, languageName } from '$lib/format';

	let presets = $state<Preset[] | null>(null);
	let vignettes = $state<Vignette[] | null>(null);
	let error = $state('');

	async function load() {
		try {
			[presets, vignettes] = await Promise.all([listPresets(), listVignettes()]);
			error = '';
		} catch (e) {
			error = (e as Error).message;
		}
	}
	load();

	const vignetteName = (id: string | null) => vignettes?.find((v) => v.id === id)?.name ?? '';
	const capitalize = (s: string) => s[0].toUpperCase() + s.slice(1);

	/** "Logo (horizontal) · Logo vertical (vertical)", or "none". */
	function describeVignette(horizontal: string | null, vertical: string | null): string {
		const parts = [];
		if (horizontal) parts.push(`${vignetteName(horizontal)} (16:9)`);
		if (vertical) parts.push(`${vignetteName(vertical)} (9:16)`);
		return parts.join(' · ') || 'none';
	}

	// ---- Preset form ----

	const blank = (): PresetInput => ({
		name: '',
		captions: true,
		captionStyle: 'highlight',
		captionPosition: 'bottom',
		captionColor: '',
		captionCaps: null,
		captionLanguage: '',
		introHorizontalId: null,
		introVerticalId: null,
		outroHorizontalId: null,
		outroVerticalId: null
	});
	let editing = $state<string | null>(null); // a preset's id, 'new', or null
	let form = $state<PresetInput>(blank());
	let saving = $state(false);
	let formError = $state('');
	let confirming = $state(''); // id of the preset asking "delete?"

	function edit(p: Preset | null) {
		formError = '';
		if (p) {
			const { id, createdAt, updatedAt, ...input } = p;
			void id, createdAt, updatedAt;
			form = { ...input };
			editing = p.id;
		} else {
			form = blank();
			editing = 'new';
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		formError = '';
		try {
			if (editing === 'new') await createPreset(form);
			else if (editing) await updatePreset(editing, form);
			editing = null;
			await load();
		} catch (err) {
			formError = (err as Error).message;
		} finally {
			saving = false;
		}
	}

	async function remove(id: string) {
		try {
			await deletePreset(id);
			confirming = '';
			await load();
		} catch (e) {
			error = (e as Error).message;
		}
	}

	// ---- Vignette upload ----

	let file = $state<File | null>(null);
	let vignetteNameInput = $state('');
	let fileNote = $state('');
	let progress = $state<number | null>(null);
	let uploadError = $state('');

	function pick(e: Event) {
		file = (e.currentTarget as HTMLInputElement).files?.[0] ?? null;
		uploadError = fileNote = '';
		if (!file) return;
		if (!file.name.toLowerCase().endsWith('.mp4')) uploadError = 'Only .mp4 files are accepted.';
		else if (file.size > vignetteLimits.maxBytes) uploadError = 'A vignette can be up to 100 MB.';
		if (!vignetteNameInput) vignetteNameInput = file.name.replace(/\.mp4$/i, '');
		// Read its size and length in the browser, to warn early.
		const probe = document.createElement('video');
		const url = URL.createObjectURL(file);
		probe.preload = 'metadata';
		probe.onloadedmetadata = () => {
			fileNote = `${probe.videoWidth}x${probe.videoHeight} · ${probe.duration.toFixed(1)} s`;
			if (probe.duration > vignetteLimits.maxSeconds)
				uploadError = `It lasts ${probe.duration.toFixed(1)} s; vignettes can last up to ${vignetteLimits.maxSeconds} s.`;
			URL.revokeObjectURL(url);
		};
		probe.onerror = () => URL.revokeObjectURL(url);
		probe.src = url;
	}

	async function upload(e: SubmitEvent) {
		e.preventDefault();
		const form = e.currentTarget as HTMLFormElement; // gone from the event after the first await
		if (!file || uploadError) return;
		progress = 0;
		try {
			await uploadVignette(vignetteNameInput, file, (p) => (progress = p));
			file = null;
			vignetteNameInput = fileNote = '';
			form.reset();
			await load();
		} catch (err) {
			uploadError = (err as Error).message;
		} finally {
			progress = null;
		}
	}

	let previewing = $state(''); // id of the vignette playing
	let vignetteError = $state('');
	async function removeVignette(id: string) {
		vignetteError = '';
		try {
			await deleteVignette(id);
			await load();
		} catch (e) {
			vignetteError = (e as Error).message;
		}
	}
</script>

<svelte:head><title>Presets · UltraMedia</title></svelte:head>

<h1>Presets</h1>
<p class="muted intro">
	A preset keeps how captions look, their language, and an opening and closing joined to the clip. Pick one on a clip
	before exporting.
</p>

{#if error}<p class="card error">{error}</p>{/if}

<section>
	<div class="section-head">
		<h2>Presets</h2>
		{#if editing === null}
			<button class="secondary" onclick={() => edit(null)} data-testid="new-preset">+ New preset</button>
		{/if}
	</div>

	{#if editing !== null && vignettes}
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
			{#if formError}<p class="error">{formError}</p>{/if}
			<div class="buttons">
				<button class="primary" disabled={saving} data-testid="preset-save">{saving ? 'Saving…' : 'Save preset'}</button>
				<button type="button" class="secondary" onclick={() => (editing = null)}>Cancel</button>
			</div>
		</form>
	{/if}

	{#if presets && presets.length > 0}
		<ul class="list" data-testid="presets">
			{#each presets as p (p.id)}
				<li class="card">
					<div class="item-head">
						<strong>{p.name}</strong>
						<span class="spacer"></span>
						<button class="secondary" onclick={() => edit(p)} disabled={editing !== null}>✎ Edit</button>
						<button class="secondary" onclick={() => (confirming = p.id)}>🗑 Delete</button>
					</div>
					<p class="muted small">{describeCaptions(p)}</p>
					<p class="muted small">
						Opening: {describeVignette(p.introHorizontalId, p.introVerticalId)} · Closing: {describeVignette(
							p.outroHorizontalId,
							p.outroVerticalId
						)}
					</p>
					{#if confirming === p.id}
						<div class="confirm">
							<span>Delete {p.name}? Its vignettes stay in the library, and exported files are kept.</span>
							<button class="danger" onclick={() => remove(p.id)}>Yes, delete</button>
							<button class="secondary" onclick={() => (confirming = '')}>Cancel</button>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{:else if presets && editing === null}
		<p class="card muted">No presets yet.</p>
	{/if}
</section>

<section>
	<h2>Vignettes</h2>
	<p class="muted small">
		Openings and closings: short MP4 files, up to 100 MB and {vignetteLimits.maxSeconds} s, with or without sound.
	</p>
	<form class="card upload" onsubmit={upload}>
		<label>
			Name
			<input bind:value={vignetteNameInput} maxlength="60" placeholder="e.g. Logo opening" data-testid="vignette-name" />
		</label>
		<label class="file">
			File
			<input type="file" accept="video/mp4,.mp4" onchange={pick} disabled={progress !== null} data-testid="vignette-file" />
		</label>
		{#if fileNote}<span class="muted small">{fileNote}</span>{/if}
		<button class="primary" disabled={!file || !!uploadError || progress !== null} data-testid="vignette-upload">
			{progress === null ? 'Upload' : `Uploading… ${Math.round(progress * 100)}%`}
		</button>
		{#if uploadError}<p class="error wide">{uploadError}</p>{/if}
	</form>

	{#if vignetteError}<p class="error">{vignetteError}</p>{/if}
	{#if vignettes && vignettes.length > 0}
		<ul class="list" data-testid="vignettes">
			{#each vignettes as v (v.id)}
				<li class="card">
					<div class="item-head">
						<strong>{v.name}</strong>
						<span class="muted small">{formatBytes(v.sizeBytes)}</span>
						<span class="spacer"></span>
						<button class="secondary" onclick={() => (previewing = previewing === v.id ? '' : v.id)}>
							{previewing === v.id ? 'Hide' : '▶ Play'}
						</button>
						<button
							class="secondary"
							disabled={v.usedBy.length > 0}
							title={v.usedBy.length > 0 ? 'Used by a preset: take it out first' : 'Delete this vignette'}
							onclick={() => removeVignette(v.id)}>🗑 Delete</button
						>
					</div>
					{#if v.usedBy.length > 0}<p class="muted small">Used by {v.usedBy.join(', ')}</p>{/if}
					{#if previewing === v.id}
						<!-- svelte-ignore a11y_media_has_caption -->
						<video src={vignetteFileUrl(v.id)} controls playsinline autoplay></video>
					{/if}
				</li>
			{/each}
		</ul>
	{:else if vignettes}
		<p class="card muted">No vignettes yet.</p>
	{/if}
</section>

<style>
	.intro {
		margin: 0.25rem 0 1.25rem;
	}
	section {
		margin-bottom: 2rem;
	}
	.section-head {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 1rem;
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
	input[type='file'] {
		max-width: 100%;
		font-size: 0.85rem;
	}
	.buttons {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}
	.upload {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.75rem;
		margin: 0.5rem 0 0.75rem;
	}
	.wide {
		flex-basis: 100%;
		margin: 0;
	}
	.confirm {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.5rem;
		font-size: 0.9rem;
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
