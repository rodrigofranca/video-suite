// Client for the UltraMedia API.
import { env } from '$env/dynamic/public';

export const API_URL = (env.PUBLIC_API_URL || 'http://localhost:8080').replace(/\/$/, '');

export type AssetStatus = 'uploading' | 'processing' | 'ready' | 'failed';
export type JobStatus = 'pending' | 'running' | 'done' | 'failed';

/** Technical metadata extracted by ffprobe. */
export interface MediaSummary {
	durationSeconds: number;
	width?: number;
	height?: number;
	videoCodec?: string;
	audioCodec?: string;
	bitRate?: number;
	frameRate?: number;
	container?: string;
	sizeBytes?: number;
}

export interface Asset {
	id: string;
	filename: string;
	status: AssetStatus;
	createdAt: string;
	updatedAt: string;
	thumbnailUrl: string | null;
	metadata: MediaSummary | null;
}

export interface Job {
	id: string;
	assetId: string;
	type: string;
	/** What people call the type, from the job table (docs/adr/0036). */
	label: string;
	/** Whether it can be run again by hand; exports and translations run with an export. */
	canRerun: boolean;
	status: JobStatus;
	progress: number;
	error: string | null;
	input: unknown;
	createdAt: string;
	startedAt: string | null;
	completedAt: string | null;
}

export interface Artifact {
	id: string;
	jobId: string;
	type: string;
	storageKey: string;
	metadata: Record<string, unknown>;
	createdAt: string;
}

export interface Playback {
	type: 'hls';
	manifestUrl: string;
	/** Qualities in the stream, best first; empty for older single-quality videos. */
	renditions: { name: string; width: number; height: number }[];
}

export interface Subtitles {
	language: string;
	url: string;
	/** The transcript's language, for subtitles a language model translated. */
	translatedFrom?: string;
}

export interface TranscriptSegment {
	start: number;
	end: number;
	text: string;
}

/** A word and when it was said. */
export interface TranscriptWord {
	start: number;
	end: number;
	word: string;
}

export interface Transcript {
	language: string;
	model: string;
	text: string;
	segments: TranscriptSegment[];
	/** Every word with its timing; empty for transcripts made before words were saved. */
	words: TranscriptWord[];
	createdAt: string;
}

export interface Chapter {
	/** Seconds from the start of the video. */
	start: number;
	title: string;
}

/** Summary, tags and chapters a language model wrote from the transcript. */
export interface Analysis {
	language: string;
	model: string;
	summary: string;
	tags: string[];
	chapters: Chapter[];
	createdAt: string;
}

export type ClipFormat = 'original' | 'vertical' | 'vertical-face' | 'vertical-blur';
/** Burned-in captions. All but 'lines' follow the transcript's word timings. */
export type CaptionStyle = 'highlight' | 'box' | 'karaoke' | 'one-word' | 'lines';
export const captionStyles: CaptionStyle[] = ['highlight', 'box', 'karaoke', 'one-word', 'lines'];
export type CaptionPosition = 'bottom' | 'middle' | 'top';
export const captionPositions: CaptionPosition[] = ['bottom', 'middle', 'top'];
/** Accent colors of the highlighted word, the box and the karaoke fill. */
export type CaptionColor = 'yellow' | 'green' | 'pink' | 'blue' | 'violet';
export const captionColors: { name: CaptionColor; hex: string }[] = [
	{ name: 'yellow', hex: '#FFE600' },
	{ name: 'green', hex: '#3CE85E' },
	{ name: 'pink', hex: '#FF4FB8' },
	{ name: 'blue', hex: '#33B5FF' },
	{ name: 'violet', hex: '#8B5CF6' }
];
/** Styles with an accent color, and their own. */
export const styleColor: Partial<Record<CaptionStyle, CaptionColor>> = {
	highlight: 'yellow',
	karaoke: 'yellow',
	box: 'violet'
};

/** Languages captions can be translated into (ISO 639-1). */
export const captionLanguages = ['en', 'es', 'pt', 'fr', 'de', 'it'];

/** How captions look besides their style, and their language. */
export interface CaptionOptions {
	position: CaptionPosition;
	color?: CaptionColor;
	caps: boolean;
	/** A language to translate the captions into; empty or missing for the transcript's own. */
	language?: string;
	/** A preset whose opening and closing the export joins (its other choices are sent explicitly). */
	presetId?: string;
}

/** An opening or closing in the library (docs/adr/0030). */
export interface Vignette {
	id: string;
	name: string;
	filename: string;
	sizeBytes: number;
	createdAt: string;
	/** Presets that use it; it cannot be deleted while they do. */
	usedBy: string[];
}

/** A named set of export choices (docs/adr/0030). */
export interface Preset {
	id: string;
	name: string;
	captions: boolean;
	captionStyle: CaptionStyle;
	captionPosition: CaptionPosition;
	/** Empty for the style's own. */
	captionColor: CaptionColor | '';
	/** Null for the style's own. */
	captionCaps: boolean | null;
	/** Empty for the transcript's own. */
	captionLanguage: string;
	introHorizontalId: string | null;
	introVerticalId: string | null;
	outroHorizontalId: string | null;
	outroVerticalId: string | null;
	/** The watermark image, its corner, width (% of the clip's shorter side) and opacity (%). */
	watermarkId: string | null;
	watermarkPosition: WatermarkPosition;
	watermarkSize: number;
	watermarkOpacity: number;
	createdAt: string;
	updatedAt: string;
}

/** A watermark image in the library (docs/adr/0031). */
export interface Watermark {
	id: string;
	name: string;
	filename: string;
	sizeBytes: number;
	width: number;
	height: number;
	createdAt: string;
	usedBy: string[];
}
export type WatermarkPosition = 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left';
export const watermarkPositions: WatermarkPosition[] = ['top-right', 'top-left', 'bottom-right', 'bottom-left'];
export type PresetInput = Omit<Preset, 'id' | 'createdAt' | 'updatedAt'>;

/** The longest vignette accepted, in bytes and seconds (as the API and the worker check them). */
export const vignetteLimits = { maxBytes: 100 * 1024 ** 2, maxSeconds: 30 };
export const clipFormats: ClipFormat[] = ['original', 'vertical', 'vertical-face', 'vertical-blur'];

/** The newest export of a clip in one format. */
export interface ClipExport {
	format: ClipFormat;
	jobId: string;
	status: JobStatus;
	progress: number;
	error: string | null;
	captions: boolean;
	captionStyle?: CaptionStyle;
	captionPosition?: CaptionPosition;
	captionColor?: CaptionColor;
	captionCaps?: boolean;
	/** The captions' language (the file's once ready, else the one asked for). */
	captionLanguage?: string;
	/** The transcript's language, when the captions were translated. */
	translatedFrom?: string;
	/** The preset used, and whether the file has an opening and a closing. */
	preset?: string;
	intro?: boolean;
	outro?: boolean;
	watermark?: boolean;
	/** How a 'vertical-face' file was framed. */
	framing?: 'face' | 'center';
	width?: number;
	height?: number;
	sizeBytes?: number;
	/** A file can be downloaded (possibly from an earlier export). */
	ready: boolean;
	/** The file was cut before the clip's start or end was changed. */
	outdated?: boolean;
}

/** A moment that works as a short standalone video, suggested from the transcript or made from a quote in it. */
export interface Clip {
	id: string;
	start: number;
	end: number;
	duration: number;
	title: string;
	reason: string;
	/** "suggested" by the language model, or made from a "quote" (docs/adr/0039). */
	origin: 'suggested' | 'quote';
	/** 1 to 10, from the model; null for a quote. */
	score: number | null;
	rank: number;
	model: string;
	createdAt: string;
	/** Share of the clip with a face, or null before face tracking. */
	faceCoverage: number | null;
	/** What '9:16 face' follows: who speaks (several people) or the biggest face. */
	faceFollows: 'speaker' | 'face' | null;
	/** People on screen at once, or null before face tracking. */
	facePeople: number | null;
	/** When the clip was last changed by hand, or null. */
	editedAt: string | null;
	exports: ClipExport[];
}

export interface AssetDetail extends Asset {
	source: { storageKey: string; mimeType: string; sizeBytes: number; checksum: string } | null;
	jobs: Job[];
	artifacts: Artifact[];
	playback: Playback | null;
	subtitles: Subtitles[];
	/** What the language model used for this video, in total and per job type. */
	aiUsage: AIUsage & { byAction: AIUsageGroup[] };
}

/** Calls to the language model and what they used (docs/adr/0032). */
export interface AIUsage {
	calls: number;
	promptTokens: number;
	completionTokens: number;
	/** US dollars, summed over the calls the gateway priced. */
	costUsd: number;
	/** Calls without a price (the gateway did not know the model's). */
	unpricedCalls: number;
	/** Calls read back from the logs of jobs run before calls were recorded. */
	fromLogCalls: number;
}
export interface AIUsageGroup extends AIUsage {
	/** A day (YYYY-MM-DD), model, job type or video ID ("" for deleted videos). */
	key: string;
	/** For videos, the file name; for job types, what people call them. */
	label?: string;
}
export interface UsageSummary {
	days: number;
	since: string | null;
	total: AIUsage;
	byDay: AIUsageGroup[];
	byModel: AIUsageGroup[];
	byAction: AIUsageGroup[];
	byVideo: AIUsageGroup[];
}
/** Periods the usage page offers, in days; 0 is everything. */
/** A piece of what was said; match marks the words that matched a search. */
export interface TextPart {
	text: string;
	match?: boolean;
}

/** A moment of a video whose speech matches a search (docs/adr/0038). */
export interface SearchMoment {
	start: number;
	end: number;
	parts: TextPart[];
}

export interface SearchVideo {
	asset: Asset;
	language: string;
	/** How many moments match; moments are the best of them, in time order. */
	matches: number;
	moments: SearchMoment[];
}

export interface SearchResult {
	query: string;
	/** How many videos have a transcript: those a search looks through. */
	searched: number;
	videos: SearchVideo[];
	/** More videos after these: ask again with offset. */
	more: boolean;
}

export const searchSpeech = (q: string, offset = 0) =>
	getJSON<SearchResult>(`/search?q=${encodeURIComponent(q)}${offset ? `&offset=${offset}` : ''}`);

export const usagePeriods = [1, 7, 30, 90, 0] as const;
export const getUsage = (days: number) => getJSON<UsageSummary>(`/usage?days=${days}`);

export interface JobLog {
	id: string;
	level: 'info' | 'warning' | 'error';
	message: string;
	timestamp: string;
}

export class ApiError extends Error {
	constructor(
		message: string,
		public status: number
	) {
		super(message);
	}
}

async function getJSON<T>(path: string): Promise<T> {
	let res: Response;
	try {
		res = await fetch(API_URL + path);
	} catch {
		throw new ApiError(`Could not reach the API at ${API_URL}. Is it running (make dev-api)?`, 0);
	}
	const body = await res.json().catch(() => ({}));
	if (!res.ok) throw new ApiError(body.error ?? `Request failed (${res.status})`, res.status);
	return body as T;
}

export const listAssets = () => getJSON<{ assets: Asset[] }>('/assets').then((r) => r.assets);
export const getAsset = (id: string) => getJSON<AssetDetail>(`/assets/${encodeURIComponent(id)}`);
export const getTranscript = (id: string) => getJSON<Transcript>(`/assets/${encodeURIComponent(id)}/transcript`);
export const getAnalysis = (id: string) => getJSON<Analysis>(`/assets/${encodeURIComponent(id)}/analysis`);
/** The clips; whether the transcript has word timings (for word-by-word
 * captions); its language; and the languages it has been translated into. */
export interface ClipList {
	clips: Clip[];
	wordTimings: boolean;
	transcriptLanguage: string;
	translations: string[];
}
export const getClips = (id: string) => getJSON<ClipList>(`/assets/${encodeURIComponent(id)}/clips`);
/** Link to an exported clip file; inline plays it instead of downloading it. */
export const clipFileUrl = (clipId: string, format: ClipFormat, inline = false) =>
	`${API_URL}/clips/${encodeURIComponent(clipId)}/download?format=${format}${inline ? '&inline=1' : ''}`;
/** URL of the full metadata (summary and raw ffprobe output) as JSON. */
export const metadataUrl = (id: string) => `${API_URL}/assets/${encodeURIComponent(id)}/metadata`;
export const getJobLogs = (id: string) =>
	getJSON<{ logs: JobLog[] }>(`/jobs/${encodeURIComponent(id)}/logs`).then((r) => r.logs);

const postJSON = <T>(path: string, data: unknown) => sendJSON<T>('POST', path, data);

async function sendJSON<T>(method: 'POST' | 'PUT' | 'PATCH' | 'DELETE', path: string, data?: unknown): Promise<T> {
	let res: Response;
	try {
		res = await fetch(API_URL + path, {
			method,
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data)
		});
	} catch {
		throw new ApiError(`Could not reach the API at ${API_URL}. Is it running (make dev-api)?`, 0);
	}
	const body = await res.json().catch(() => ({}));
	if (!res.ok) throw new ApiError(body.error ?? `Request failed (${res.status})`, res.status);
	return body as T;
}

/** Deletes a video, its clips and every stored file. */
export const deleteAsset = (assetId: string) => sendJSON<void>('DELETE', `/assets/${encodeURIComponent(assetId)}`);

/** Deletes a clip and its exported files. */
export const deleteClip = (assetId: string, clipId: string) =>
	sendJSON<void>('DELETE', `/assets/${encodeURIComponent(assetId)}/clips/${encodeURIComponent(clipId)}`);

/** Limits of a clip edited by hand (as the API checks them). */
export const clipLimits = { minSeconds: 3, maxSeconds: 180, maxTitle: 120 };

/** Changes a clip's start, end (seconds from the start of the video) or title. */
/** Makes a clip from a quote (docs/adr/0039); without a title, the quote is the title. */
export const createClip = (assetId: string, range: { start: number; end: number; title?: string }) =>
	sendJSON<Clip>('POST', `/assets/${encodeURIComponent(assetId)}/clips`, range);

export const editClip = (assetId: string, clipId: string, changes: { start?: number; end?: number; title?: string }) =>
	sendJSON<Clip>('PATCH', `/assets/${encodeURIComponent(assetId)}/clips/${encodeURIComponent(clipId)}`, changes);

/** Runs one pipeline job of an asset again. */
export const rerunJob = (assetId: string, type: string) =>
	postJSON<Job>(`/assets/${encodeURIComponent(assetId)}/jobs`, { type });

/** Queues the export of a clip in one format. Captions in a language the
 * transcript has no translation into yet are translated first. */
export const exportClip = (
	assetId: string,
	clipId: string,
	format: ClipFormat,
	captions: boolean,
	captionStyle: CaptionStyle,
	options?: CaptionOptions
) =>
	postJSON<Job>(`/assets/${encodeURIComponent(assetId)}/clips/${encodeURIComponent(clipId)}/exports`, {
		format,
		captions,
		captionStyle,
		captionPosition: options?.position,
		captionColor: options?.color,
		captionCaps: options?.caps,
		captionLanguage: options?.language || undefined,
		presetId: options?.presetId || undefined
	});

export const listVignettes = () => getJSON<{ vignettes: Vignette[] }>('/vignettes').then((r) => r.vignettes);
/** Link that plays a vignette (a short-lived signed link behind a redirect). */
export const vignetteFileUrl = (id: string) => `${API_URL}/vignettes/${encodeURIComponent(id)}/file`;
export const deleteVignette = (id: string) => sendJSON<void>('DELETE', `/vignettes/${encodeURIComponent(id)}`);
export const listPresets = () => getJSON<{ presets: Preset[] }>('/presets').then((r) => r.presets);
export const createPreset = (p: PresetInput) => postJSON<Preset>('/presets', p);
export const updatePreset = (id: string, p: PresetInput) => sendJSON<Preset>('PUT', `/presets/${encodeURIComponent(id)}`, p);
export const deletePreset = (id: string) => sendJSON<void>('DELETE', `/presets/${encodeURIComponent(id)}`);

export const listWatermarks = () => getJSON<{ watermarks: Watermark[] }>('/watermarks').then((r) => r.watermarks);
/** Link that shows a watermark image (a short-lived signed link behind a redirect). */
export const watermarkFileUrl = (id: string) => `${API_URL}/watermarks/${encodeURIComponent(id)}/file`;
export const deleteWatermark = (id: string) => sendJSON<void>('DELETE', `/watermarks/${encodeURIComponent(id)}`);
/** Uploads a PNG or JPEG watermark image, reporting progress from 0 to 1. */
export function uploadWatermark(name: string, file: File, onProgress: (fraction: number) => void): Promise<Watermark> {
	return sendForm<Watermark>('/watermarks/upload', [['name', name], ['file', file]], 201, onProgress);
}

/** Uploads a vignette to the library, reporting progress from 0 to 1. */
export function uploadVignette(name: string, file: File, onProgress: (fraction: number) => void): Promise<Vignette> {
	return sendForm<Vignette>('/vignettes/upload', [['name', name], ['file', file]], 201, onProgress);
}

/** Uploads a file, reporting progress from 0 to 1. */
export function uploadAsset(
	file: File,
	onProgress: (fraction: number) => void
): Promise<{ assetId: string; status: AssetStatus }> {
	return sendForm<{ assetId: string; status: AssetStatus }>('/assets/upload', [['file', file]], 201, onProgress);
}

/** Posts a multipart form, in order, reporting upload progress from 0 to 1.
 * Uses XHR because fetch has no upload progress. */
function sendForm<T>(
	path: string,
	fields: [string, string | File][],
	okStatus: number,
	onProgress: (fraction: number) => void
): Promise<T> {
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('POST', API_URL + path);
		xhr.upload.onprogress = (e) => {
			if (e.lengthComputable) onProgress(e.loaded / e.total);
		};
		xhr.onload = () => {
			let body: T & { error?: string } = {} as T & { error?: string };
			try {
				body = JSON.parse(xhr.responseText);
			} catch {
				/* not JSON */
			}
			if (xhr.status === okStatus) resolve(body);
			else reject(new ApiError(body.error ?? `Upload failed (${xhr.status})`, xhr.status));
		};
		xhr.onerror = () =>
			reject(new ApiError(`Could not reach the API at ${API_URL}. Is it running (make dev-api)?`, 0));
		const form = new FormData();
		for (const [name, value] of fields) form.append(name, value);
		xhr.send(form);
	});
}
