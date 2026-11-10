import { describe, expect, it } from 'vitest';
import type { Job } from './api';
import { doneKey, encodingText, isActive, newest, resuggestNote, running } from './jobs';

const job = (type: string, status: Job['status'], extra: Partial<Job> = {}): Job => ({
	id: `${type}-${status}`,
	assetId: 'a',
	type,
	label: type,
	canRerun: true,
	status,
	progress: 0,
	error: null,
	input: null,
	createdAt: '',
	startedAt: null,
	completedAt: null,
	...extra
});

describe('jobs', () => {
	it('knows which jobs are still going', () => {
		expect(isActive(job('x', 'pending'))).toBe(true);
		expect(isActive(job('x', 'running'))).toBe(true);
		expect(isActive(job('x', 'done'))).toBe(false);
		expect(isActive(undefined)).toBe(false);
		const jobs = [job('hls', 'failed', { id: 'old' }), job('hls', 'running', { id: 'new' }), job('audio', 'pending')];
		expect(newest(jobs, 'hls')?.id).toBe('new');
		expect(running(jobs, 'audio')).toBe(true);
		expect(running(jobs, 'thumb')).toBe(false);
	});

	it('loads a result once per finished run', () => {
		expect(doneKey(job('x', 'done', { completedAt: 't1' }))).toBe('t1');
		expect(doneKey(job('x', 'failed', { completedAt: 't1' }))).toBeNull();
		expect(doneKey(job('x', 'running'))).toBeNull();
		expect(doneKey(undefined)).toBeNull();
	});

	it('says how encoding goes', () => {
		expect(encodingText(undefined)).toMatch(/appears here when HLS encoding finishes/);
		expect(encodingText(job('transcode-hls', 'running'))).toBe('Encoding HLS…');
		expect(encodingText(job('transcode-hls', 'running', { progress: 42 }))).toBe('Encoding HLS… 42%');
	});

	it('says what suggesting clips again does', () => {
		const clip = (edited: boolean) => ({ editedAt: edited ? '2026-11-10' : null });
		expect(resuggestNote(null)).toBe('');
		expect(resuggestNote([])).toBe('');
		expect(resuggestNote([clip(false), clip(false)])).toBe('This replaces the 2 clips below with new suggestions.');
		expect(resuggestNote([clip(true)])).toBe('The 1 clip you edited stays; new suggestions are added around it.');
		expect(resuggestNote([clip(true), clip(true), clip(false)])).toBe(
			'This replaces the other 1 clip with new suggestions. The 2 clips you edited stay.'
		);
	});
});
