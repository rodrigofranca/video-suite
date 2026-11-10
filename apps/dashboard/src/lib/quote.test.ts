import { describe, expect, it } from 'vitest';
import { findParts, pickUnit, quoteProblem, quoteRange, quoteText, quoteTitle, units, unitsBySegment } from './quote';

const segments = [
	{ start: 0, end: 4, text: ' Bom dia a todos.' },
	{ start: 4, end: 9, text: ' A reforma foi aprovada.' }
];
const words = [
	{ start: 0.2, end: 0.5, word: 'Bom' },
	{ start: 0.5, end: 1, word: 'dia' },
	{ start: 1, end: 1.4, word: 'a' },
	{ start: 1.4, end: 2, word: 'todos.' },
	{ start: 4.3, end: 4.4, word: 'A' },
	{ start: 4.4, end: 5, word: 'reforma' },
	{ start: 5, end: 5.4, word: 'foi' },
	{ start: 5.4, end: 6.2, word: 'aprovada.' }
];

describe('units', () => {
	it('are the words, each in its segment', () => {
		const list = units(segments, words);
		expect(list).toHaveLength(8);
		expect(unitsBySegment(segments, list)).toEqual([
			[0, 1, 2, 3],
			[4, 5, 6, 7]
		]);
	});
	it('are the segments without word timings', () => {
		const list = units(segments, []);
		expect(list.map((u) => u.text)).toEqual(['Bom dia a todos.', 'A reforma foi aprovada.']);
		expect(unitsBySegment(segments, list)).toEqual([[0], [1]]);
	});
});

describe('pickUnit', () => {
	it('starts, ends either way round, and starts again', () => {
		let p = pickUnit(null, 6);
		expect(p).toEqual({ from: 6, to: 6, done: false });
		p = pickUnit(p, 4);
		expect(p).toEqual({ from: 4, to: 6, done: true });
		expect(pickUnit(p, 1)).toEqual({ from: 1, to: 1, done: false });
		expect(pickUnit({ from: 2, to: 2, done: false }, 2)).toEqual({ from: 2, to: 2, done: true });
	});
});

describe('the quote', () => {
	const list = units(segments, words);
	it('is snapped to its words, with room that does not take in the next word', () => {
		// "A reforma foi aprovada.": 0.15 s before, 0.25 s after, nothing said around it.
		expect(quoteRange(list, 4, 7)).toEqual({ start: 4.15, end: 6.45 });
		// "dia a": the words before and after are right there.
		expect(quoteRange(list, 1, 2)).toEqual({ start: 0.5, end: 1.4 });
		// The first word, at the start of the video; the last, at its end.
		expect(quoteRange(list, 0, 0).start).toBe(0.05);
		expect(quoteRange(list, 7, 7, 6.3).end).toBe(6.3);
	});
	it('says what was said, and makes a title of it', () => {
		expect(quoteText(list, 4, 7)).toBe('A reforma foi aprovada.');
		expect(quoteTitle('  A reforma   foi aprovada. ')).toBe('A reforma foi aprovada.');
		const long = quoteTitle('palavra '.repeat(30));
		expect([...long].length).toBeLessThanOrEqual(120);
		expect(long.endsWith('palavra…')).toBe(true);
	});
	it('must last from 3 s to 3 minutes', () => {
		expect(quoteProblem(4.15, 6.45)).toMatch(/at least 3 s/);
		expect(quoteProblem(0, 181)).toMatch(/at most 3 minutes/);
		expect(quoteProblem(0, 3)).toBe('');
	});
});

describe('findParts', () => {
	it('marks the text found, without minding accents or case', () => {
		expect(findParts('A Reforma Tributária foi aprovada.', 'tributaria')).toEqual([
			{ text: 'A Reforma ' },
			{ text: 'Tributária', match: true },
			{ text: ' foi aprovada.' }
		]);
		expect(findParts('ação e reação', 'acao')).toEqual([
			{ text: 'ação', match: true },
			{ text: ' e re' },
			{ text: 'ação', match: true }
		]);
	});
	it('finds nothing for an empty query or a missing one', () => {
		expect(findParts('texto', '  ')).toBeNull();
		expect(findParts('texto', 'outro')).toBeNull();
	});
});
