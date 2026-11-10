import { describe, expect, it } from 'vitest';
import type { Preset } from './api';
import { ExportChoices, type ExportContext } from './export-choices.svelte';
import { blankPreset } from './presets';

const preset = (extra: Partial<Preset>): Preset => ({ ...blankPreset(), id: 'p', name: 'P', createdAt: '', updatedAt: '', ...extra });
const context = (extra: Partial<ExportContext> = {}): ExportContext => ({
	wordTimings: true,
	hasSubtitles: true,
	transcriptLanguage: 'pt',
	presets: [],
	...extra
});

describe('export choices', () => {
	it('starts from what the transcript allows', () => {
		expect(new ExportChoices(() => context()).captions).toBe('highlight');
		expect(new ExportChoices(() => context({ wordTimings: false })).captions).toBe('lines');
		expect(new ExportChoices(() => context({ wordTimings: false, hasSubtitles: false })).captions).toBe('off');
		const c = new ExportChoices(() => context({ wordTimings: false }));
		expect(c.available('karaoke')).toBe(false);
		expect(c.available('lines')).toBe(true);
	});

	it('gives each style its own color and case until one is picked', () => {
		const c = new ExportChoices(() => context());
		expect(c.color).toBe('yellow');
		c.captions = 'box';
		expect(c.color).toBe('violet');
		c.captions = 'lines';
		expect(c.color).toBeUndefined(); // lines have no accent
		c.captions = 'one-word';
		expect(c.caps).toBe(true);
		c.caps = false;
		expect(c.caps).toBe(false);
		c.captions = 'box';
		c.color = 'green';
		expect(c.request()).toEqual({
			captions: true,
			style: 'box',
			options: { position: 'bottom', color: 'green', caps: false, language: '', presetId: undefined }
		});
	});

	it("takes a preset's choices, which can still be changed", () => {
		const presets = [
			preset({ id: 'karaoke', captionStyle: 'karaoke', captionPosition: 'top', captionColor: 'pink', captionCaps: true, captionLanguage: 'en' }),
			preset({ id: 'plain', captionStyle: 'karaoke', captionLanguage: 'pt' }),
			preset({ id: 'none', captions: false })
		];
		const c = new ExportChoices(() => context({ presets, wordTimings: false }));
		c.applyPreset('karaoke');
		// Karaoke needs word timings: the captions fall back to lines.
		expect([c.captions, c.position, c.caps, c.language]).toEqual(['lines', 'top', true, 'en']);
		c.applyPreset('plain');
		expect([c.caps, c.language]).toEqual([false, '']); // the transcript's own language is no translation
		c.applyPreset('none');
		expect(c.request().captions).toBe(false);
		expect(c.request().options.presetId).toBe('none');
		c.applyPreset('');
		expect(c.preset).toBeUndefined();
		expect(c.request().options.presetId).toBeUndefined();
	});
});
