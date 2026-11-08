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
}

export interface Subtitles {
	language: string;
	url: string;
}

export interface TranscriptSegment {
	start: number;
	end: number;
	text: string;
}

export interface Transcript {
	language: string;
	model: string;
	text: string;
	segments: TranscriptSegment[];
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

/** How captions look besides their style. */
export interface CaptionOptions {
	position: CaptionPosition;
	color?: CaptionColor;
	caps: boolean;
}
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

/** A moment that works as a short standalone video, suggested from the transcript. */
export interface Clip {
	id: string;
	start: number;
	end: number;
	duration: number;
	title: string;
	reason: string;
	score: number;
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
}

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
/** The clips, and whether the transcript has word timings (for word-by-word captions). */
export const getClips = (id: string) =>
	getJSON<{ clips: Clip[]; wordTimings: boolean }>(`/assets/${encodeURIComponent(id)}/clips`);
/** Link to an exported clip file; inline plays it instead of downloading it. */
export const clipFileUrl = (clipId: string, format: ClipFormat, inline = false) =>
	`${API_URL}/clips/${encodeURIComponent(clipId)}/download?format=${format}${inline ? '&inline=1' : ''}`;
/** URL of the full metadata (summary and raw ffprobe output) as JSON. */
export const metadataUrl = (id: string) => `${API_URL}/assets/${encodeURIComponent(id)}/metadata`;
export const getJobLogs = (id: string) =>
	getJSON<{ logs: JobLog[] }>(`/jobs/${encodeURIComponent(id)}/logs`).then((r) => r.logs);

const postJSON = <T>(path: string, data: unknown) => sendJSON<T>('POST', path, data);

async function sendJSON<T>(method: 'POST' | 'PATCH' | 'DELETE', path: string, data?: unknown): Promise<T> {
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
export const editClip = (assetId: string, clipId: string, changes: { start?: number; end?: number; title?: string }) =>
	sendJSON<Clip>('PATCH', `/assets/${encodeURIComponent(assetId)}/clips/${encodeURIComponent(clipId)}`, changes);

/** Runs one pipeline job of an asset again. */
export const rerunJob = (assetId: string, type: string) =>
	postJSON<Job>(`/assets/${encodeURIComponent(assetId)}/jobs`, { type });

/** Queues the export of a clip in one format. */
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
		captionCaps: options?.caps
	});

/** Uploads a file, reporting progress from 0 to 1. Uses XHR because fetch has no upload progress. */
export function uploadAsset(
	file: File,
	onProgress: (fraction: number) => void
): Promise<{ assetId: string; status: AssetStatus }> {
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('POST', `${API_URL}/assets/upload`);
		xhr.upload.onprogress = (e) => {
			if (e.lengthComputable) onProgress(e.loaded / e.total);
		};
		xhr.onload = () => {
			let body: { assetId?: string; status?: AssetStatus; error?: string } = {};
			try {
				body = JSON.parse(xhr.responseText);
			} catch {
				/* not JSON */
			}
			if (xhr.status === 201 && body.assetId) {
				resolve({ assetId: body.assetId, status: body.status ?? 'processing' });
			} else {
				reject(new ApiError(body.error ?? `Upload failed (${xhr.status})`, xhr.status));
			}
		};
		xhr.onerror = () =>
			reject(new ApiError(`Could not reach the API at ${API_URL}. Is it running (make dev-api)?`, 0));
		const form = new FormData();
		form.append('file', file);
		xhr.send(form);
	});
}
