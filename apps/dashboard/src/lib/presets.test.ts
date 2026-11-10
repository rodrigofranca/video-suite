import { describe, expect, it } from 'vitest';
import type { Preset } from './api';
import { blankPreset, cornerLabel, describeVignettes, describeWatermark, inputOf, watermarkPreviewStyle } from './presets';

const vignettes = [{ id: 'v', name: 'Logo', filename: '', sizeBytes: 1, createdAt: '', usedBy: [] }];
const marks = [{ id: 'w', name: 'Badge', filename: '', sizeBytes: 1, width: 8, height: 8, createdAt: '', usedBy: [] }];

describe('presets in words', () => {
	it('names openings, closings and the watermark', () => {
		expect(describeVignettes(null, null, vignettes)).toBe('none');
		expect(describeVignettes('v', 'v', vignettes)).toBe('Logo (16:9) · Logo (9:16)');
		expect(describeWatermark(blankPreset(), marks)).toBe('none');
		expect(describeWatermark({ ...blankPreset(), watermarkId: 'w', watermarkPosition: 'bottom-left' }, marks)).toBe(
			'Badge, bottom left, 15%, 80% opacity'
		);
		expect(cornerLabel('top-right')).toBe('Top right');
	});

	it('edits a preset without its id and dates', () => {
		const p: Preset = { ...blankPreset(), name: 'Channel', id: 'p', createdAt: 'c', updatedAt: 'u' };
		const input = inputOf(p);
		expect(input).toEqual({ ...blankPreset(), name: 'Channel' });
		expect('id' in input).toBe(false);
	});

	it('places the preview as the export places the watermark', () => {
		const p = { ...blankPreset(), watermarkSize: 20, watermarkOpacity: 60, watermarkPosition: 'bottom-left' as const };
		// 9:16: the shorter side is the width.
		expect(watermarkPreviewStyle(p, true)).toBe('width: 20%; opacity: 0.6;bottom: 2.25%; left: 4%;');
		// 16:9: the shorter side is the height, 9/16 of the width.
		expect(watermarkPreviewStyle(p, false)).toBe('width: 11.25%; opacity: 0.6;bottom: 4%; left: 2.25%;');
	});
});
