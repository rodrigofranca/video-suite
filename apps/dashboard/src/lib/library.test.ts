import { describe, expect, it } from 'vitest';
import { vignettes, watermarks } from './library';

const file = (name: string, size = 1) => ({ name, size }) as File;

describe('library kinds', () => {
	it('accepts what the API accepts', () => {
		expect(vignettes.check(file('intro.MP4'))).toBe('');
		expect(vignettes.check(file('intro.mov'))).toBe('Only .mp4 files are accepted.');
		expect(vignettes.check(file('intro.mp4', 101 * 1024 ** 2))).toBe('A vignette can be up to 100 MB.');
		expect(watermarks.check(file('logo.jpeg'))).toBe('');
		expect(watermarks.check(file('logo.gif'))).toBe('Only PNG and JPEG images are accepted.');
		expect(watermarks.check(file('logo.png', 6 * 1024 ** 2))).toBe('An image can be up to 5 MB.');
	});

	it('names a file after itself', () => {
		expect(vignettes.nameOf(file('Logo opening.mp4'))).toBe('Logo opening');
		expect(watermarks.nameOf(file('badge.PNG'))).toBe('badge');
		expect(watermarks.details({ id: 'w', name: 'b', sizeBytes: 2048, usedBy: [], width: 260, height: 80 })).toBe('260x80 · 2.0 KB');
	});
});
