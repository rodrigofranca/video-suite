<script lang="ts">
	import { getJobLogs, type Job, type JobLog } from '$lib/api';
	import { formatDuration, formatTime, jobLabel } from '$lib/format';
	import StatusBadge from './StatusBadge.svelte';

	let { jobs, onRerun }: { jobs: Job[]; onRerun?: (type: string) => Promise<void> } = $props();

	let openJob = $state<string | null>(null);
	let logs = $state<JobLog[]>([]);
	let logsError = $state('');
	let rerunning = $state<string | null>(null);
	let rerunError = $state('');

	// Jobs come in creation order; a re-run adds a new attempt of a type.
	let busy = $derived(jobs.some((j) => j.status === 'pending' || j.status === 'running'));
	let latest = $derived(new Map(jobs.map((j) => [j.type, j.id])));
	let attempts = $derived.by(() => {
		const count = new Map<string, number>();
		const byJob = new Map<string, { n: number; total: number }>();
		for (const j of jobs) count.set(j.type, (count.get(j.type) ?? 0) + 1);
		const seen = new Map<string, number>();
		for (const j of jobs) {
			const n = (seen.get(j.type) ?? 0) + 1;
			seen.set(j.type, n);
			byJob.set(j.id, { n, total: count.get(j.type)! });
		}
		return byJob;
	});

	const canRerun = (job: Job) =>
		!!onRerun && !busy && latest.get(job.type) === job.id && (job.status === 'done' || job.status === 'failed');

	async function rerun(job: Job) {
		if (!onRerun) return;
		rerunning = job.type;
		rerunError = '';
		try {
			await onRerun(job.type);
			openJob = null;
		} catch (e) {
			rerunError = (e as Error).message;
		} finally {
			rerunning = null;
		}
	}

	async function toggle(job: Job) {
		if (openJob === job.id) {
			openJob = null;
			return;
		}
		openJob = job.id;
		logs = [];
		logsError = '';
		try {
			logs = await getJobLogs(job.id);
		} catch (e) {
			logsError = (e as Error).message;
		}
	}
</script>

<ol class="jobs">
	{#each jobs as job (job.id)}
		{@const attempt = attempts.get(job.id)}
		<li class="job" class:old={latest.get(job.type) !== job.id} data-testid="job-{job.type}">
			<button class="row" onclick={() => toggle(job)} aria-expanded={openJob === job.id}>
				<span class="name">
					{jobLabel(job.type)}
					{#if attempt && attempt.total > 1}<span class="attempt">attempt {attempt.n}</span>{/if}
				</span>
				<StatusBadge status={job.status} />
				<span class="muted">
					{job.status === 'running' && job.progress > 0
						? `${job.progress}%`
						: formatDuration(job.startedAt, job.completedAt)}
				</span>
				<span class="chevron" aria-hidden="true">{openJob === job.id ? '▾' : '▸'}</span>
			</button>
			{#if job.status === 'running' && job.progress > 0}
				<div class="progress" data-testid="job-progress">
					<div class="bar" style="width: {job.progress}%"></div>
				</div>
			{/if}
			{#if job.error}
				<p class="job-error">{job.error}</p>
			{/if}
			{#if canRerun(job) && (job.status === 'failed' || openJob === job.id)}
				<div class="actions">
					<button class="secondary" onclick={() => rerun(job)} disabled={rerunning !== null} data-testid="rerun">
						{rerunning === job.type ? 'Starting…' : 'Run again'}
					</button>
					{#if rerunError && rerunning === null}<span class="job-error inline">{rerunError}</span>{/if}
				</div>
			{/if}
			{#if openJob === job.id}
				<div class="logs" data-testid="logs">
					{#if logsError}
						<p class="job-error">{logsError}</p>
					{:else if logs.length === 0}
						<p class="muted">No logs yet.</p>
					{:else}
						{#each logs as log (log.id)}
							<div class="log {log.level}">
								<span class="time">{formatTime(log.timestamp)}</span>
								<span class="msg">{log.message}</span>
							</div>
						{/each}
					{/if}
				</div>
			{/if}
		</li>
	{/each}
</ol>

<style>
	.jobs {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: 0.5rem;
	}
	.job {
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--surface);
	}
	.row {
		all: unset;
		box-sizing: border-box;
		width: 100%;
		display: grid;
		grid-template-columns: 1fr auto auto 1rem;
		gap: 0.75rem;
		align-items: center;
		padding: 0.65rem 0.85rem;
		cursor: pointer;
	}
	.row:focus-visible {
		outline: 2px solid var(--accent);
		border-radius: 8px;
	}
	.name {
		font-weight: 600;
	}
	.attempt {
		font-weight: 400;
		font-size: 0.8rem;
		color: var(--muted);
		margin-left: 0.35rem;
	}
	.old {
		opacity: 0.6;
	}
	.actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
		margin: 0 0.85rem 0.65rem;
	}
	.inline {
		margin: 0;
	}
	.progress {
		height: 4px;
		margin: 0 0.85rem 0.65rem;
		background: var(--badge-neutral-bg);
		border-radius: 999px;
		overflow: hidden;
	}
	.bar {
		height: 100%;
		background: var(--accent);
		transition: width 0.5s;
	}
	.job-error {
		margin: 0 0.85rem 0.65rem;
		color: var(--badge-err-fg);
		font-size: 0.85rem;
		overflow-wrap: anywhere;
	}
	.logs {
		border-top: 1px solid var(--border);
		padding: 0.5rem 0.85rem 0.75rem;
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		font-size: 0.78rem;
		max-height: 320px;
		overflow: auto;
	}
	.log {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.75rem;
		padding: 0.1rem 0;
	}
	.log .msg {
		overflow-wrap: anywhere;
	}
	.log.warning .msg {
		color: var(--badge-busy-fg);
	}
	.log.error .msg {
		color: var(--badge-err-fg);
	}
	.time,
	.muted,
	.chevron {
		color: var(--muted);
	}
</style>
