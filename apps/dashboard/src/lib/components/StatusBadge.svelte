<script lang="ts">
	let { status }: { status: string } = $props();
</script>

<span class="badge {status}" data-testid="status">{status}</span>

<style>
	.badge {
		display: inline-block;
		padding: 0.1rem 0.55rem;
		border-radius: 999px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: capitalize;
		background: var(--badge-neutral-bg);
		color: var(--badge-neutral-fg);
		white-space: nowrap;
	}
	.ready,
	.done {
		background: var(--badge-ok-bg);
		color: var(--badge-ok-fg);
	}
	.processing,
	.running,
	.uploading {
		background: var(--badge-busy-bg);
		color: var(--badge-busy-fg);
	}
	.failed {
		background: var(--badge-err-bg);
		color: var(--badge-err-fg);
	}
</style>
