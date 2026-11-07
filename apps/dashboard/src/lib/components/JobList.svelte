<script lang="ts">
	import { getJobLogs, type Job, type JobLog } from '$lib/api';
	import { formatDuration, formatTime, jobLabel } from '$lib/format';
	import StatusBadge from './StatusBadge.svelte';

	let { jobs }: { jobs: Job[] } = $props();

	let openJob = $state<string | null>(null);
	let logs = $state<JobLog[]>([]);
	let logsError = $state('');

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
		<li class="job" data-testid="job-{job.type}">
			<button class="row" onclick={() => toggle(job)} aria-expanded={openJob === job.id}>
				<span class="name">{jobLabel(job.type)}</span>
				<StatusBadge status={job.status} />
				<span class="muted">{formatDuration(job.startedAt, job.completedAt)}</span>
				<span class="chevron" aria-hidden="true">{openJob === job.id ? '▾' : '▸'}</span>
			</button>
			{#if job.error}
				<p class="job-error">{job.error}</p>
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
