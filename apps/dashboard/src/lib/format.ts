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
	'transcode-hls': 'HLS encoding'
};

export const jobLabel = (type: string) => jobLabels[type] ?? type;
