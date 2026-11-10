import type { Clip, Job } from './api';

// Rules about a video's jobs that the pages show (docs/adr/0037). Plain
// functions, so they are tested without a browser.

/** A job (or an export) that has not finished: pending or running. */
export const isActive = (job: { status: string } | undefined): boolean =>
	job?.status === 'pending' || job?.status === 'running';

/** The newest job of a type (jobs come oldest first). */
export const newest = (jobs: Job[], type: string): Job | undefined => jobs.findLast((j) => j.type === type);

/** Whether a job of a type is pending or running. */
export const running = (jobs: Job[], type: string): boolean => jobs.some((j) => j.type === type && isActive(j));

/** The key that loads a job's result: its completion time, once it is done. */
export const doneKey = (job: Job | undefined): string | null => (job?.status === 'done' && job.completedAt) || null;

/** What the player area says while the video is being processed. */
export function encodingText(hls: Job | undefined): string {
	if (hls?.status !== 'running') return 'Processing… the video appears here when HLS encoding finishes.';
	return hls.progress > 0 ? `Encoding HLS… ${hls.progress}%` : 'Encoding HLS…';
}

/** What suggesting clips again does, asked before it runs (docs/adr/0026); empty without clips. */
export function resuggestNote(clips: Pick<Clip, 'editedAt'>[] | null | undefined): string {
	if (!clips || clips.length === 0) return '';
	const edited = clips.filter((c) => c.editedAt).length;
	const others = clips.length - edited;
	const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`;
	if (edited === 0) return `This replaces the ${plural(others, 'clip')} below with new suggestions.`;
	const keep = `The ${plural(edited, 'clip')} you edited stay${edited === 1 ? 's' : ''}`;
	return others === 0
		? `${keep}; new suggestions are added around ${edited === 1 ? 'it' : 'them'}.`
		: `This replaces the other ${plural(others, 'clip')} with new suggestions. ${keep}.`;
}
