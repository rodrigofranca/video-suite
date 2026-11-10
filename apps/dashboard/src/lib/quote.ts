import { clipLimits, type TextPart, type TranscriptSegment, type TranscriptWord } from './api';

// Rules of a clip made from a quote in the transcript (docs/adr/0039):
// what can be picked, the range it becomes, its title, and finding text in
// the transcript. Plain functions, tested without a browser.

/** Something that can be picked: a word, or a whole segment when the transcript has no word timings. */
export interface Unit {
	start: number;
	end: number;
	text: string;
}

/** The words to pick from, or the segments for a transcript without word timings. */
export function units(segments: TranscriptSegment[], words: TranscriptWord[] | undefined): Unit[] {
	if (words && words.length > 0) return words.map((w) => ({ start: w.start, end: w.end, text: w.word }));
	return segments.map((s) => ({ start: s.start, end: s.end, text: s.text.trim() }));
}

/** The units said in each segment, by index, so a segment can show its words. */
export function unitsBySegment(segments: TranscriptSegment[], list: Unit[]): number[][] {
	const out: number[][] = segments.map(() => []);
	if (segments.length === 0) return out;
	let s = 0;
	list.forEach((u, i) => {
		while (s < segments.length - 1 && u.start >= segments[s].end) s++;
		out[s].push(i);
	});
	return out;
}

/** A pick in progress: the first unit tapped, and the last once tapped. */
export interface Pick {
	from: number;
	to: number;
	done: boolean;
}

/** Tapping a unit: the first tap starts a quote, the second ends it (either way round), a third starts again. */
export function pickUnit(pick: Pick | null, i: number): Pick {
	if (!pick || pick.done) return { from: i, to: i, done: false };
	return { from: Math.min(pick.from, i), to: Math.max(pick.from, i), done: true };
}

// A little room around the words, as long as it does not take in the next
// or the previous word: speech starts a little before a word's timing.
const lead = 0.15;
const tail = 0.25;
const round = (t: number) => Math.round(t * 1000) / 1000;

/** The clip's range for the units from..to: snapped to them, with a little room. */
export function quoteRange(list: Unit[], from: number, to: number, duration = 0): { start: number; end: number } {
	const before = from > 0 ? list[from - 1].end : 0;
	const after = to < list.length - 1 ? list[to + 1].start : duration > 0 ? duration : Infinity;
	const start = Math.max(before, list[from].start - lead, 0);
	const end = Math.min(after, list[to].end + tail);
	return { start: round(Math.min(start, list[from].start)), end: round(Math.max(end, list[to].end)) };
}

/** What was said from unit `from` to unit `to`. */
export function quoteText(list: Unit[], from: number, to: number): string {
	return list
		.slice(from, to + 1)
		.map((u) => u.text)
		.join(' ')
		.replace(/\s+/g, ' ')
		.trim();
}

/** A clip title from a quote: the quote, cut at a word with "…" past the longest title. */
export function quoteTitle(quote: string): string {
	const q = quote.replace(/\s+/g, ' ').trim();
	const chars = [...q];
	if (chars.length <= clipLimits.maxTitle) return q;
	let cut = chars.slice(0, clipLimits.maxTitle - 1).join('');
	const space = cut.lastIndexOf(' ');
	if (space > clipLimits.maxTitle / 2) cut = cut.slice(0, space);
	return cut.replace(/[\s,;:.-]+$/, '') + '…';
}

/** Why the range cannot be a clip yet, or ''. */
export function quoteProblem(start: number, end: number): string {
	const length = end - start;
	if (length < clipLimits.minSeconds) return `A clip lasts at least ${clipLimits.minSeconds} s: pick more words.`;
	if (length > clipLimits.maxSeconds) return `A clip lasts at most ${clipLimits.maxSeconds / 60} minutes: pick fewer words.`;
	return '';
}

/** Text for finding, without accents or case, with where each of its characters came from. */
function fold(text: string): { folded: string; from: number[] } {
	let folded = '';
	const from: number[] = [];
	let i = 0;
	for (const ch of text) {
		const f = ch.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase();
		for (let k = 0; k < f.length; k++) from.push(i);
		folded += f;
		i += ch.length;
	}
	from.push(i);
	return { folded, from };
}

/** The text in parts, marking where the query is, without minding accents or case; null when it is not there. */
export function findParts(text: string, query: string): TextPart[] | null {
	const q = fold(query.trim()).folded;
	if (!q) return null;
	const { folded, from } = fold(text);
	const parts: TextPart[] = [];
	let last = 0;
	for (let at = folded.indexOf(q); at >= 0; at = folded.indexOf(q, at + q.length)) {
		const a = from[at];
		const b = from[at + q.length];
		if (a > last) parts.push({ text: text.slice(last, a) });
		parts.push({ text: text.slice(a, b), match: true });
		last = b;
	}
	if (parts.length === 0) return null;
	if (last < text.length) parts.push({ text: text.slice(last) });
	return parts;
}
