import type { Preset, PresetInput, Vignette, Watermark } from './api';

// Presets in words, and the form's starting point (docs/adr/0030, 0031).
// Plain functions, so they are tested without a browser.

export const watermarkSizes = [10, 15, 20, 25, 30];
export const watermarkOpacities = [100, 80, 60, 40];

/** A new preset: highlighted-word captions at the bottom, nothing joined, no watermark. */
export const blankPreset = (): PresetInput => ({
	name: '',
	captions: true,
	captionStyle: 'highlight',
	captionPosition: 'bottom',
	captionColor: '',
	captionCaps: null,
	captionLanguage: '',
	introHorizontalId: null,
	introVerticalId: null,
	outroHorizontalId: null,
	outroVerticalId: null,
	watermarkId: null,
	watermarkPosition: 'top-right',
	watermarkSize: 15,
	watermarkOpacity: 80,
	jumpCuts: false
});

/** A preset's choices, to edit them. */
export function inputOf(p: Preset): PresetInput {
	const { id, createdAt, updatedAt, ...input } = p;
	void id, createdAt, updatedAt;
	return { ...input };
}

/** "Top right" for "top-right". */
export const cornerLabel = (position: string): string => {
	const s = position.replace('-', ' ');
	return s[0].toUpperCase() + s.slice(1);
};

/** An opening or closing: "Logo (16:9) · Logo tall (9:16)", or "none". */
export function describeVignettes(horizontal: string | null, vertical: string | null, vignettes: Vignette[]): string {
	const name = (id: string) => vignettes.find((v) => v.id === id)?.name ?? '';
	const parts = [];
	if (horizontal) parts.push(`${name(horizontal)} (16:9)`);
	if (vertical) parts.push(`${name(vertical)} (9:16)`);
	return parts.join(' · ') || 'none';
}

/** "Logo, top right, 15%, 80% opacity", or "none". */
export function describeWatermark(p: PresetInput, watermarks: Watermark[]): string {
	const name = watermarks.find((w) => w.id === p.watermarkId)?.name;
	if (!p.watermarkId || !name) return 'none';
	return `${name}, ${cornerLabel(p.watermarkPosition).toLowerCase()}, ${p.watermarkSize}%, ${p.watermarkOpacity}% opacity`;
}

/** Where a preview of a 16:9 or 9:16 frame puts the watermark, as CSS: the
 * margin is 4% of the shorter side and the width a share of it, as in the export. */
export function watermarkPreviewStyle(p: PresetInput, vertical: boolean): string {
	const ratio = 9 / 16; // the shorter side over the longer one
	const across = vertical ? 1 : ratio; // the shorter side, as a share of the frame's width
	const down = vertical ? ratio : 1; // ...and of its height
	const [v, h] = p.watermarkPosition.split('-');
	return (
		`width: ${p.watermarkSize * across}%; opacity: ${p.watermarkOpacity / 100};` +
		`${v}: ${4 * down}%; ${h}: ${4 * across}%;`
	);
}
