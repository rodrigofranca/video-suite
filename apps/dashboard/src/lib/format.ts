export function formatBytes(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	const units = ['KB', 'MB', 'GB'];
	let value = bytes / 1024;
	let unit = 0;
	while (value >= 1024 && unit < units.length - 1) {
		value /= 1024;
		unit++;
	}
	return `${value.toFixed(value < 10 ? 1 : 0)} ${units[unit]}`;
}

export function formatDate(iso: string): string {
	return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
}

export function formatTime(iso: string): string {
	return new Date(iso).toLocaleTimeString(undefined, { hour12: false });
}

/** Seconds between two timestamps, e.g. "3.4 s". */
export function formatDuration(start: string | null, end: string | null): string {
	if (!start || !end) return '';
	const s = (new Date(end).getTime() - new Date(start).getTime()) / 1000;
	return s < 60 ? `${s.toFixed(1)} s` : `${Math.floor(s / 60)} min ${Math.round(s % 60)} s`;
}

const jobLabels: Record<string, string> = {
	'extract-metadata': 'Metadata',
	'generate-thumbnail': 'Thumbnail',
	'transcode-hls': 'HLS encoding',
	'extract-audio': 'Audio',
	'transcribe-audio': 'Transcription',
	'analyze-transcript': 'Summary',
	'suggest-clips': 'Clips',
	'export-clip': 'Clip export'
};

const clipFormatLabels: Record<string, string> = {
	original: 'Original',
	vertical: '9:16 crop',
	'vertical-blur': '9:16 blur'
};

export const clipFormatLabel = (format: string) => clipFormatLabels[format] ?? format;

const captionStyleLabels: Record<string, string> = {
	highlight: 'Highlighted word',
	box: 'Word box',
	karaoke: 'Karaoke',
	'one-word': 'One word at a time',
	lines: 'Lines'
};

export const captionStyleLabel = (style: string) => captionStyleLabels[style] ?? style;

const languageNames = new Intl.DisplayNames(['en'], { type: 'language' });

/** "en" → "English"; falls back to the code. */
export function languageName(code: string): string {
	try {
		return languageNames.of(code) ?? code;
	} catch {
		return code;
	}
}

export const jobLabel = (type: string) => jobLabels[type] ?? type;

/** Media length as m:ss or h:mm:ss. */
export function formatClock(seconds: number): string {
	const total = Math.round(seconds);
	const h = Math.floor(total / 3600);
	const m = Math.floor((total % 3600) / 60);
	const s = String(total % 60).padStart(2, '0');
	return h > 0 ? `${h}:${String(m).padStart(2, '0')}:${s}` : `${m}:${s}`;
}

export function formatResolution(width?: number, height?: number): string {
	return width && height ? `${width}×${height}` : '';
}

export function formatBitrate(bitsPerSecond?: number): string {
	if (!bitsPerSecond) return '';
	return bitsPerSecond >= 1e6 ? `${(bitsPerSecond / 1e6).toFixed(1)} Mb/s` : `${Math.round(bitsPerSecond / 1e3)} kb/s`;
}
