import { describe, expect, it } from 'vitest';
import type { ClipExport, Preset } from './api';
import { describeExport, faceStatus, presetExtras } from './clips';
import { blankPreset } from './presets';

const ex = (extra: Partial<ClipExport>): ClipExport => ({
	format: 'vertical',
	jobId: 'j',
	status: 'done',
	progress: 100,
	error: null,
	captions: false,
	ready: true,
	...extra
});

describe('clips in words', () => {
	it('says how a 9:16 face export will be framed', () => {
		const clip = (faceCoverage: number | null, faceFollows: 'speaker' | 'face' | null = null, facePeople: number | null = null) => ({
			faceCoverage,
			faceFollows,
			facePeople
		});
		expect(faceStatus(clip(null), true)).toBe('Finding faces…');
		expect(faceStatus(clip(null), false)).toBe('No face tracking yet: centred');
		expect(faceStatus(clip(0.2), false)).toBe('Few faces found: centred');
		expect(faceStatus(clip(0.9, 'speaker', 2), false)).toBe('Follows who speaks · 2 people');
		expect(faceStatus(clip(0.756, 'face', 1), false)).toBe('Face in 76% of the clip');
	});

	it('describes an exported file', () => {
		expect(describeExport(ex({}))).toBe('');
		expect(describeExport(ex({ framing: 'center' }))).toBe(' · centred');
		// The style's own color and the one-word style's capitals go unsaid.
		expect(describeExport(ex({ captions: true, captionStyle: 'highlight', captionColor: 'yellow' }))).toBe(' · Highlighted word');
		expect(describeExport(ex({ captions: true, captionStyle: 'one-word', captionCaps: true }))).toBe(' · One word at a time');
		expect(
			describeExport(
				ex({
					captions: true,
					captionStyle: 'box',
					captionPosition: 'top',
					captionColor: 'pink',
					captionCaps: true,
					captionLanguage: 'es',
					translatedFrom: 'en',
					preset: 'Channel',
					intro: true,
					watermark: true
				})
			)
		).toBe(' · Word box, top, pink, caps, Spanish · Channel (opening + watermark)');
	});

	it('says what a preset adds', () => {
		const p = { ...blankPreset(), id: 'p', createdAt: '', updatedAt: '' } as Preset;
		const vignettes = [
			{ id: 'v1', name: 'Wide logo', filename: '', sizeBytes: 1, createdAt: '', usedBy: [] },
			{ id: 'v2', name: 'Tall logo', filename: '', sizeBytes: 1, createdAt: '', usedBy: [] }
		];
		const marks = [{ id: 'w', name: 'Badge', filename: '', sizeBytes: 1, width: 8, height: 8, createdAt: '', usedBy: [] }];
		expect(presetExtras(p, vignettes, marks)).toBe('');
		// The vertical version is named first; either one is enough.
		expect(presetExtras({ ...p, introHorizontalId: 'v1', introVerticalId: 'v2' }, vignettes, marks)).toBe('join the opening “Tall logo”');
		expect(presetExtras({ ...p, outroHorizontalId: 'v1', watermarkId: 'w' }, vignettes, marks)).toBe(
			'join the closing “Wide logo”, and lay the watermark “Badge”'
		);
	});
});
