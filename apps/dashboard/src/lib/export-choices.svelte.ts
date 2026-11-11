import { styleColor, type CaptionColor, type CaptionOptions, type CaptionPosition, type CaptionStyle, type Preset } from './api';

/** A clip's captions: a style, or none. */
export type CaptionChoice = CaptionStyle | 'off';

/** What the choices depend on, from the video. */
export interface ExportContext {
	/** Word-by-word styles need the transcript's word timings. */
	wordTimings: boolean;
	/** The lines style needs the subtitles. */
	hasSubtitles: boolean;
	/** The transcript's language. */
	transcriptLanguage: string;
	presets: Preset[];
}

/**
 * How one clip will be exported (docs/adr/0021, 0025, 0029, 0030, 0040):
 * its captions, their look and language, jump cuts, and a preset. Until something is
 * picked, captions follow the word when the transcript allows, else lines,
 * else none, and each style brings its own color and case. A preset fills
 * in every choice, which can still be changed.
 */
export class ExportChoices {
	#context: () => ExportContext;
	#captions = $state<CaptionChoice | null>(null);
	#color = $state<CaptionColor | null>(null);
	#caps = $state<boolean | null>(null);
	position = $state<CaptionPosition>('bottom');
	/** The captions' language: the transcript's own ('') or a translation. */
	language = $state('');
	presetId = $state('');
	#jumpCuts = $state(false);

	constructor(context: () => ExportContext) {
		this.#context = context;
	}

	/** Whether a style can be burned in this video's clips. */
	available(style: CaptionStyle): boolean {
		const c = this.#context();
		return style === 'lines' ? c.hasSubtitles : c.wordTimings;
	}

	get fallback(): CaptionChoice {
		const c = this.#context();
		return c.wordTimings ? 'highlight' : c.hasSubtitles ? 'lines' : 'off';
	}

	get captions(): CaptionChoice {
		return this.#captions ?? this.fallback;
	}
	set captions(choice: CaptionChoice) {
		this.#captions = choice;
	}

	/** The accent color, for styles that have one. */
	get color(): CaptionColor | undefined {
		const c = this.captions;
		return c === 'off' || !styleColor[c] ? undefined : (this.#color ?? styleColor[c]);
	}
	set color(color: CaptionColor) {
		this.#color = color;
	}

	/** Capitals: the one-word style's look, unless changed. */
	get caps(): boolean {
		return this.#caps ?? this.captions === 'one-word';
	}
	set caps(on: boolean) {
		this.#caps = on;
	}

	/** Jump cuts need the transcript's word timings. */
	get canJumpCut(): boolean {
		return this.#context().wordTimings;
	}

	/** Cut the pauses and filler sounds out, where the transcript allows. */
	get jumpCuts(): boolean {
		return this.#jumpCuts && this.canJumpCut;
	}
	set jumpCuts(on: boolean) {
		this.#jumpCuts = on;
	}

	get preset(): Preset | undefined {
		return this.#context().presets.find((p) => p.id === this.presetId);
	}

	/** Picks a preset and takes its choices; '' picks none and keeps them. */
	applyPreset(id: string) {
		this.presetId = id;
		const p = this.preset;
		if (!p) return;
		this.#captions = !p.captions ? 'off' : this.available(p.captionStyle) ? p.captionStyle : this.fallback;
		this.position = p.captionPosition;
		this.#color = p.captionColor || null;
		this.#caps = p.captionCaps;
		this.language = p.captionLanguage === this.#context().transcriptLanguage ? '' : p.captionLanguage;
		this.#jumpCuts = p.jumpCuts;
	}

	/** The export request for these choices. */
	request(): { captions: boolean; style: CaptionStyle; options: CaptionOptions } {
		const c = this.captions;
		return {
			captions: c !== 'off',
			style: c === 'off' ? 'highlight' : c,
			options: {
				position: this.position,
				color: this.color,
				caps: this.caps,
				language: this.language,
				presetId: this.presetId || undefined,
				jumpCuts: this.jumpCuts
			}
		};
	}
}
