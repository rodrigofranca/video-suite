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
	'track-faces': 'Face tracking',
	'export-clip': 'Clip export',
	'translate-transcript': 'Translation'
};

const clipFormatLabels: Record<string, string> = {
	original: 'Original',
	vertical: '9:16 crop',
	'vertical-face': '9:16 face',
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

/** Like formatClock, with tenths of a second: "1:08.4". */
export function formatClockTenths(seconds: number): string {
	const tenths = Math.round(seconds * 10);
	const whole = Math.floor(tenths / 10);
	return `${formatClock(whole)}.${tenths % 10}`;
}

export function formatResolution(width?: number, height?: number): string {
	return width && height ? `${width}×${height}` : '';
}

export function formatBitrate(bitsPerSecond?: number): string {
	if (!bitsPerSecond) return '';
	return bitsPerSecond >= 1e6 ? `${(bitsPerSecond / 1e6).toFixed(1)} Mb/s` : `${Math.round(bitsPerSecond / 1e3)} kb/s`;
}

/** Token counts, compact: 950, 12.3K, 4.1M. */
export function formatTokens(n: number): string {
	if (n < 1000) return String(n);
	if (n < 1_000_000) return `${(n / 1000).toFixed(n < 10_000 ? 1 : 0)}K`;
	return `${(n / 1_000_000).toFixed(1)}M`;
}

/** US dollars, with the decimals small amounts need: US$ 0.00035, US$ 0.12, US$ 3.40. */
export function formatUsd(v: number): string {
	if (v === 0) return 'US$ 0';
	const digits = v >= 1 ? 2 : Math.min(6, Math.max(2, 1 - Math.floor(Math.log10(v)))); // two significant digits
	return `US$ ${v.toFixed(digits)}`;
}

/** What a preset does, on one line: "Word box, top, pink, caps · English". */
export function describeCaptions(p: {
	captions: boolean;
	captionStyle: string;
	captionPosition: string;
	captionColor: string;
	captionCaps: boolean | null;
	captionLanguage: string;
}): string {
	if (!p.captions) return 'No captions';
	const parts = [captionStyleLabel(p.captionStyle)];
	if (p.captionPosition && p.captionPosition !== 'bottom') parts.push(p.captionPosition);
	if (p.captionColor) parts.push(p.captionColor);
	if (p.captionCaps !== null) parts.push(p.captionCaps ? 'caps' : 'no caps');
	return parts.join(', ') + (p.captionLanguage ? ` · ${languageName(p.captionLanguage)}` : ' · spoken language');
}
