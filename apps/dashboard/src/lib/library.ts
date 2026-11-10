import {
	deleteVignette,
	deleteWatermark,
	uploadVignette,
	uploadWatermark,
	vignetteFileUrl,
	vignetteLimits,
	watermarkFileUrl
} from './api';
import { formatBytes } from './format';

// The brand library's kinds of file, as the API has them (docs/adr/0033,
// 0037): what each accepts, how it is named and shown, and its calls. One
// upload form and one list serve every kind.

/** A file in the library. */
export interface LibraryItem {
	id: string;
	name: string;
	sizeBytes: number;
	usedBy: string[];
}

export interface LibraryKind<T extends LibraryItem = LibraryItem> {
	/** For test ids and messages: "vignette", "watermark". */
	id: string;
	accept: string;
	namePlaceholder: string;
	/** How the list shows a file: a video to play, or an image. */
	preview: 'video' | 'image';
	/** Why a picked file cannot be uploaded, or ''. */
	check(file: File): string;
	/** The name a file gets when none is typed. */
	nameOf(file: File): string;
	/** What the browser can tell about a file before upload, and why it cannot be used (or ''). */
	inspect?(file: File): Promise<{ note: string; problem: string }>;
	upload(name: string, file: File, onProgress: (fraction: number) => void): Promise<T>;
	remove(id: string): Promise<void>;
	fileUrl(id: string): string;
	/** Size and such, for the list. */
	details(item: T): string;
}

export const vignettes: LibraryKind = {
	id: 'vignette',
	accept: 'video/mp4,.mp4',
	namePlaceholder: 'e.g. Logo opening',
	preview: 'video',
	check: (file) =>
		!file.name.toLowerCase().endsWith('.mp4')
			? 'Only .mp4 files are accepted.'
			: file.size > vignetteLimits.maxBytes
				? 'A vignette can be up to 100 MB.'
				: '',
	nameOf: (file) => file.name.replace(/\.mp4$/i, ''),
	inspect: (file) =>
		new Promise((resolve) => {
			// Its size and length, read in the browser, to warn early.
			const probe = document.createElement('video');
			const url = URL.createObjectURL(file);
			probe.preload = 'metadata';
			probe.onloadedmetadata = () => {
				URL.revokeObjectURL(url);
				const seconds = probe.duration;
				resolve({
					note: `${probe.videoWidth}x${probe.videoHeight} · ${seconds.toFixed(1)} s`,
					problem:
						seconds > vignetteLimits.maxSeconds
							? `It lasts ${seconds.toFixed(1)} s; vignettes can last up to ${vignetteLimits.maxSeconds} s.`
							: ''
				});
			};
			probe.onerror = () => {
				URL.revokeObjectURL(url);
				resolve({ note: '', problem: '' });
			};
			probe.src = url;
		}),
	upload: uploadVignette,
	remove: deleteVignette,
	fileUrl: vignetteFileUrl,
	details: (v) => formatBytes(v.sizeBytes)
};

export const watermarks: LibraryKind<LibraryItem & { width: number; height: number }> = {
	id: 'watermark',
	accept: 'image/png,image/jpeg',
	namePlaceholder: 'e.g. Channel logo',
	preview: 'image',
	check: (file) =>
		!/\.(png|jpe?g)$/i.test(file.name)
			? 'Only PNG and JPEG images are accepted.'
			: file.size > 5 * 1024 ** 2
				? 'An image can be up to 5 MB.'
				: '',
	nameOf: (file) => file.name.replace(/\.(png|jpe?g)$/i, ''),
	upload: uploadWatermark,
	remove: deleteWatermark,
	fileUrl: watermarkFileUrl,
	details: (w) => `${w.width}x${w.height} · ${formatBytes(w.sizeBytes)}`
};
