<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Job } from '$lib/api';

	// A part of the video page that shows what a job made: the content once
	// there is some, else what the job is doing.
	let {
		title,
		job,
		ready,
		empty = false,
		failed,
		emptyText,
		runningText,
		runningId,
		waiting,
		class: className = '',
		children
	}: {
		title: string;
		job: Job;
		/** There is something to show. */
		ready: boolean;
		/** The job finished with nothing to show. */
		empty?: boolean;
		failed: string;
		emptyText: string;
		runningText: string;
		/** data-testid of the running message. */
		runningId: string;
		/** Before the job runs. */
		waiting: string;
		class?: string;
		children: Snippet;
	} = $props();
</script>

<section class="job-section {className}">
	<h2>{title}</h2>
	{#if ready}
		{@render children()}
	{:else if job.status === 'failed'}
		<p class="muted">{failed}</p>
	{:else if job.status === 'done' && empty}
		<p class="muted">{emptyText}</p>
	{:else if job.status === 'running'}
		<p class="muted" data-testid={runningId}>{runningText}</p>
	{:else}
		<p class="muted">{waiting}</p>
	{/if}
</section>

<style>
	.job-section {
		margin-bottom: 1.5rem;
	}
</style>
