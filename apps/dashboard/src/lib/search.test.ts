import { describe, expect, it } from 'vitest';
import { matchesText, momentLink, searchLink, segmentAt, startTime } from './search';

describe('search links', () => {
	it('opens the video at the moment, keeping the query', () => {
		expect(momentLink('a1', 120.85, 'reforma tributária')).toBe('/assets/a1?t=120.85&q=reforma+tribut%C3%A1ria');
		expect(momentLink('a1', 1 / 3, '')).toBe('/assets/a1?t=0.33');
		expect(momentLink('a1', 0, '')).toBe('/assets/a1?t=0');
	});
	it('goes back to the results', () => {
		expect(searchLink('  "a phrase" ')).toBe('/search?q=%22a+phrase%22');
		expect(searchLink('  ')).toBe('/search');
	});
});

describe('startTime', () => {
	const at = (q: string) => startTime(new URLSearchParams(q));
	it('reads a time in seconds', () => {
		expect(at('t=12.5')).toBe(12.5);
		expect(at('t=0')).toBe(0);
	});
	it('ignores a missing or unusable time', () => {
		expect(at('')).toBeNull();
		expect(at('t=')).toBeNull();
		expect(at('t=-3')).toBeNull();
		expect(at('t=abc')).toBeNull();
		expect(at('t=Infinity')).toBeNull();
	});
});

describe('matchesText', () => {
	it('counts the moments, or the ones not shown', () => {
		expect(matchesText(1, 1)).toBe('1 moment');
		expect(matchesText(2, 2)).toBe('2 moments');
		expect(matchesText(7, 3)).toBe('4 more in this video');
	});
});

describe('segmentAt', () => {
	const segments = [
		{ start: 0, end: 2 },
		{ start: 2, end: 5 }
	];
	it('finds the segment being said', () => {
		expect(segmentAt(segments, 0)).toBe(0);
		expect(segmentAt(segments, 2)).toBe(1);
		expect(segmentAt(segments, 5)).toBe(-1);
	});
});
