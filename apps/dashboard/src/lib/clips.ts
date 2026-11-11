import { styleColor, type Clip, type ClipExport, type Preset, type Vignette, type Watermark } from './api';
import { captionStyleLabel, languageName } from './format';

// What the clips show in words (docs/adr/0037). Plain functions, so they are
// tested without a browser.

/** How a '9:16 face' export will be framed, before there is one. */
export function faceStatus(clip: Pick<Clip, 'faceCoverage' | 'faceFollows' | 'facePeople'>, tracking: boolean): string {
	if (clip.faceCoverage === null) return tracking ? 'Finding faces…' : 'No face tracking yet: centred';
	if (clip.faceCoverage < 0.3) return 'Few faces found: centred';
	if (clip.faceFollows === 'speaker') return `Follows who speaks · ${clip.facePeople} people`;
	return `Face in ${Math.round(clip.faceCoverage * 100)}% of the clip`;
}

/** What an exported file has, after its size: " · 31 s, jump cuts · centred · Karaoke, top, caps · Channel (opening + watermark)". */
export function describeExport(ex: ClipExport): string {
	let text = ex.jumpCuts ? ` · ${ex.durationSeconds ? `${Math.round(ex.durationSeconds)} s, ` : ''}jump cuts` : '';
	if (ex.framing === 'center') text += ' · centred';
	if (ex.captions) {
		const style = ex.captionStyle ?? 'lines';
		text += ` · ${captionStyleLabel(style)}`;
		if (ex.captionPosition && ex.captionPosition !== 'bottom') text += `, ${ex.captionPosition}`;
		if (ex.captionColor && ex.captionColor !== styleColor[style]) text += `, ${ex.captionColor}`;
		if (ex.captionCaps && ex.captionStyle !== 'one-word') text += ', caps';
		if (ex.translatedFrom && ex.captionLanguage) text += `, ${languageName(ex.captionLanguage)}`;
	}
	if (ex.preset) text += ` · ${ex.preset}`;
	const extras = [ex.intro && 'opening', ex.outro && 'closing', ex.watermark && 'watermark'].filter(Boolean);
	if (extras.length > 0) text += ` (${extras.join(' + ')})`;
	return text;
}

/** What a preset adds to an export besides captions, or '': "join the opening “Logo”, and lay the watermark “Badge”". */
export function presetExtras(p: Preset, vignettes: Vignette[], watermarks: Watermark[]): string {
	const name = (id: string | null) => vignettes.find((v) => v.id === id)?.name;
	const opening = name(p.introVerticalId) ?? name(p.introHorizontalId);
	const closing = name(p.outroVerticalId) ?? name(p.outroHorizontalId);
	const joined = [opening && `the opening “${opening}”`, closing && `the closing “${closing}”`].filter(Boolean).join(' and ');
	const mark = watermarks.find((w) => w.id === p.watermarkId)?.name;
	return [joined && `join ${joined}`, mark && `lay the watermark “${mark}”`].filter(Boolean).join(', and ');
}
