<script lang="ts">
	import {
		createPreset,
		deletePreset,
		listPresets,
		listVignettes,
		listWatermarks,
		updatePreset,
		vignetteLimits,
		type Preset,
		type PresetInput,
		type Vignette,
		type Watermark
	} from '$lib/api';
	import Library from '$lib/components/Library.svelte';
	import PresetForm from '$lib/components/PresetForm.svelte';
	import { describeCaptions } from '$lib/format';
	import { vignettes as vignetteKind, watermarks as watermarkKind } from '$lib/library';
	import { blankPreset, describeVignettes, describeWatermark, inputOf } from '$lib/presets';

	// Presets and the brand library (docs/adr/0030, 0031). The form and each
	// library are components; this page loads and lists.
	let presets = $state<Preset[] | null>(null);
	let vignettes = $state<Vignette[] | null>(null);
	let watermarks = $state<Watermark[] | null>(null);
	let error = $state('');

	async function load() {
		try {
			[presets, vignettes, watermarks] = await Promise.all([listPresets(), listVignettes(), listWatermarks()]);
			error = '';
		} catch (e) {
			error = (e as Error).message;
		}
	}
	load();

	let editing = $state<string | null>(null); // a preset's id, 'new', or null
	let initial = $state<PresetInput>(blankPreset());
	let confirming = $state(''); // id of the preset asking "delete?"

	function edit(p: Preset | null) {
		initial = p ? inputOf(p) : blankPreset();
		editing = p ? p.id : 'new';
	}

	async function save(input: PresetInput) {
		if (editing === 'new') await createPreset(input);
		else if (editing) await updatePreset(editing, input);
		editing = null;
		await load();
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
		{#key editing}
			<PresetForm {initial} {vignettes} watermarks={watermarks ?? []} onsave={save} oncancel={() => (editing = null)} />
		{/key}
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
						Opening: {describeVignettes(p.introHorizontalId, p.introVerticalId, vignettes ?? [])} · Closing: {describeVignettes(
							p.outroHorizontalId,
							p.outroVerticalId,
							vignettes ?? []
						)}
					</p>
					<p class="muted small">Watermark: {describeWatermark(p, watermarks ?? [])}</p>
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
	<Library kind={vignetteKind} items={vignettes} emptyText="No vignettes yet." onchange={load} />
</section>

<section>
	<h2>Watermarks</h2>
	<p class="muted small">
		A logo laid over the clip: a PNG (keeps its transparency) or JPEG, up to 5 MB and 4096 px a side.
	</p>
	<Library kind={watermarkKind} items={watermarks} emptyText="No watermark images yet." onchange={load} />
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
	.confirm {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.5rem;
		font-size: 0.9rem;
	}
</style>
