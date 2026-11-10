// Rules of the archive search (docs/adr/0038).

/** Longest query, as the API allows. */
export const maxSearchLength = 200;

/** The video page at a moment, keeping the search for the way back. */
export function momentLink(assetId: string, start: number, query: string): string {
	// Whisper gives times to the hundredth: keep them, so the moment's own segment is the one shown.
	const params = new URLSearchParams({ t: String(Math.max(0, Math.round(start * 100) / 100)) });
	if (query) params.set('q', query);
	return `/assets/${encodeURIComponent(assetId)}?${params}`;
}

/** The search results page for a query, or the empty search page. */
export function searchLink(query: string): string {
	const q = query.trim();
	return q ? `/search?${new URLSearchParams({ q })}` : '/search';
}

/** The time a link asks the video page to start at, if it is a usable one. */
export function startTime(params: URLSearchParams): number | null {
	const raw = params.get('t');
	if (raw === null || raw.trim() === '') return null;
	const t = Number(raw);
	return Number.isFinite(t) && t >= 0 ? t : null;
}

/** "3 moments", or how many more the video has than the ones shown. */
export function matchesText(matches: number, shown: number): string {
	if (matches > shown) return `${matches - shown} more in this video`;
	return matches === 1 ? '1 moment' : `${matches} moments`;
}

/** Index of the segment being said at a time, or -1. */
export function segmentAt(segments: { start: number; end: number }[], t: number): number {
	return segments.findIndex((s) => t >= s.start && t < s.end);
}
