// Client for the UltraMedia API.
import { env } from '$env/dynamic/public';

export const API_URL = (env.PUBLIC_API_URL || 'http://localhost:8080').replace(/\/$/, '');

export type AssetStatus = 'uploading' | 'processing' | 'ready' | 'failed';
export type JobStatus = 'pending' | 'running' | 'done' | 'failed';

export interface Asset {
	id: string;
	filename: string;
	status: AssetStatus;
	createdAt: string;
	updatedAt: string;
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

export interface AssetDetail extends Asset {
	source: { storageKey: string; mimeType: string; sizeBytes: number; checksum: string } | null;
	jobs: Job[];
	artifacts: Artifact[];
	playback: Playback | null;
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
export const getJobLogs = (id: string) =>
	getJSON<{ logs: JobLog[] }>(`/jobs/${encodeURIComponent(id)}/logs`).then((r) => r.logs);

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
