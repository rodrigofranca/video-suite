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

export type ClipFormat = 'original' | 'vertical' | 'vertical-blur';
export const clipFormats: ClipFormat[] = ['original', 'vertical', 'vertical-blur'];

/** The newest export of a clip in one format. */
export interface ClipExport {
	format: ClipFormat;
	jobId: string;
	status: JobStatus;
	progress: number;
	error: string | null;
	captions: boolean;
	width?: number;
	height?: number;
	sizeBytes?: number;
	/** A file can be downloaded (possibly from an earlier export). */
	ready: boolean;
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
export const getClips = (id: string) =>
	getJSON<{ clips: Clip[] }>(`/assets/${encodeURIComponent(id)}/clips`).then((r) => r.clips);
/** Link to an exported clip file; inline plays it instead of downloading it. */
export const clipFileUrl = (clipId: string, format: ClipFormat, inline = false) =>
	`${API_URL}/clips/${encodeURIComponent(clipId)}/download?format=${format}${inline ? '&inline=1' : ''}`;
/** URL of the full metadata (summary and raw ffprobe output) as JSON. */
export const metadataUrl = (id: string) => `${API_URL}/assets/${encodeURIComponent(id)}/metadata`;
export const getJobLogs = (id: string) =>
	getJSON<{ logs: JobLog[] }>(`/jobs/${encodeURIComponent(id)}/logs`).then((r) => r.logs);

async function postJSON<T>(path: string, data: unknown): Promise<T> {
	let res: Response;
	try {
		res = await fetch(API_URL + path, {
			method: 'POST',
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

/** Runs one pipeline job of an asset again. */
export const rerunJob = (assetId: string, type: string) =>
	postJSON<Job>(`/assets/${encodeURIComponent(assetId)}/jobs`, { type });

/** Queues the export of a clip in one format. */
export const exportClip = (assetId: string, clipId: string, format: ClipFormat, captions: boolean) =>
	postJSON<Job>(`/assets/${encodeURIComponent(assetId)}/clips/${encodeURIComponent(clipId)}/exports`, {
		format,
		captions
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
