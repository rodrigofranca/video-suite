import type { CaptionOptions, CaptionStyle, Clip, ClipFormat, Preset, TranscriptSegment, Vignette, Watermark } from '$lib/api';
import type { ExportContext } from '$lib/export-choices.svelte';

/** What the clips need to know about their video. */
export interface ClipContext extends ExportContext {
	/** Another job of the video is pending or running: exports and edits wait. */
	busy: boolean;
	/** A face tracking job, or a translation, is pending or running. */
	tracking: boolean;
	translating: boolean;
	/** Languages the transcript is already translated into. */
	translations: string[];
	presets: Preset[];
	/** The library, to name what a preset adds. */
	vignettes: Vignette[];
	watermarks: Watermark[];
	/** Transcript segments, to snap an edited clip to sentences, and the video's length (0 when unknown). */
	segments: TranscriptSegment[];
	duration: number;
}

/** What can be done to a clip; each is left out where it is not offered. */
export interface ClipActions {
	onpreview?: (start: number, end: number) => void;
	onexport?: (clipId: string, format: ClipFormat, captions: boolean, style: CaptionStyle, options: CaptionOptions) => Promise<void>;
	onedit?: (clipId: string, changes: { start?: number; end?: number; title?: string }) => Promise<void>;
	ondelete?: (clipId: string) => Promise<void>;
}

/** What making a clip from a quote in the transcript needs (docs/adr/0039). */
export interface QuoteActions {
	/** Another job of the video is pending or running: the clip must wait. */
	busy: boolean;
	/** The video's length, or 0 when unknown. */
	duration: number;
	onpreview?: (start: number, end: number) => void;
	oncreate: (start: number, end: number, title: string) => Promise<Clip>;
}
