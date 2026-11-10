import {
	deleteAsset,
	deleteClip,
	editClip,
	exportClip,
	getAnalysis,
	getAsset,
	getClips,
	getTranscript,
	listPresets,
	listVignettes,
	listWatermarks,
	rerunJob,
	type Analysis,
	type AssetDetail,
	type CaptionOptions,
	type CaptionStyle,
	type ClipFormat,
	type ClipList,
	type Preset,
	type Transcript,
	type Vignette,
	type Watermark
} from './api';
import { JobResult } from './job-result.svelte';
import { doneKey, encodingText, isActive, newest, resuggestNote, running } from './jobs';
import { poll } from './poll.svelte';

/**
 * The video page's state and what it can do (docs/adr/0037). It polls the
 * video until its jobs end, loads the transcript, the summary and the clips
 * as their jobs finish, and runs the page's actions. The page and its
 * components only show it. Create it while the page starts.
 */
export class AssetPage {
	asset = $state<AssetDetail | null>(null);
	error = $state('');

	jobs = $derived(this.asset?.jobs ?? []);
	busy = $derived(this.jobs.some(isActive));
	hlsJob = $derived(newest(this.jobs, 'transcode-hls'));
	transcribeJob = $derived(newest(this.jobs, 'transcribe-audio'));
	analyzeJob = $derived(newest(this.jobs, 'analyze-transcript'));
	suggestJob = $derived(newest(this.jobs, 'suggest-clips'));
	pipelineJobs = $derived(this.jobs.filter((j) => j.type !== 'export-clip'));
	trackingFaces = $derived(running(this.jobs, 'track-faces'));
	translating = $derived(running(this.jobs, 'translate-transcript'));
	encodingText = $derived(encodingText(this.hlsJob));

	transcript: JobResult<Transcript>;
	analysis: JobResult<Analysis>;
	clips: JobResult<ClipList>;

	/** The presets and the files they name, for the clips' preset choice. */
	presets = $state<Preset[]>([]);
	vignettes = $state<Vignette[]>([]);
	watermarks = $state<Watermark[]>([]);

	#id: () => string;
	#updates: { refresh: () => Promise<void> };

	constructor(id: () => string) {
		this.#id = id;
		const finished = () =>
			(this.asset?.status === 'ready' || this.asset?.status === 'failed') && !this.busy;
		this.#updates = poll(
			async () => {
				try {
					this.asset = await getAsset(this.#id());
					this.error = '';
				} catch (e) {
					this.error = (e as Error).message;
				}
			},
			() => !finished() && !this.error
		);
		this.transcript = new JobResult(
			() => doneKey(this.transcribeJob),
			() => getTranscript(this.#id())
		);
		this.analysis = new JobResult(
			() => doneKey(this.analyzeJob),
			() => getAnalysis(this.#id())
		);
		// Clips change with new suggestions, a new transcription (word
		// timings), face tracking (face tracks) and every export that moves.
		this.clips = new JobResult(
			() => {
				const suggested = doneKey(this.suggestJob);
				if (!suggested) return null;
				const exports = this.jobs.filter((j) => j.type === 'export-clip');
				return [
					suggested,
					this.transcribeJob?.completedAt ?? '',
					newest(this.jobs, 'track-faces')?.completedAt ?? '',
					...exports.map((j) => `${j.id}:${j.status}:${j.progress}`)
				].join(',');
			},
			() => getClips(this.#id())
		);
		Promise.all([listPresets(), listVignettes(), listWatermarks()])
			.then(([p, v, w]) => {
				this.presets = p;
				this.vignettes = v;
				this.watermarks = w;
			})
			.catch(() => {});
	}

	/** What suggesting clips again does, asked before it runs. */
	get resuggestNote(): string {
		return resuggestNote(this.clips.value?.clips);
	}

	async rerun(type: string) {
		await rerunJob(this.#id(), type);
		await this.#updates.refresh();
	}

	async export(clipId: string, format: ClipFormat, captions: boolean, style: CaptionStyle, options: CaptionOptions) {
		await exportClip(this.#id(), clipId, format, captions, style, options);
		await this.#updates.refresh();
	}

	async editClip(clipId: string, changes: { start?: number; end?: number; title?: string }) {
		await editClip(this.#id(), clipId, changes);
		await this.clips.reload();
		await this.#updates.refresh();
	}

	async deleteClip(clipId: string) {
		await deleteClip(this.#id(), clipId);
		await this.clips.reload();
	}

	/** Deletes the video and everything about it; the caller leaves the page. */
	async deleteVideo() {
		await deleteAsset(this.#id());
	}
}
