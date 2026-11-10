<script lang="ts">
	import { getUsage, usagePeriods, type AIUsageGroup, type UsageSummary } from '$lib/api';
	import { formatTokens, formatUsd } from '$lib/format';
	import UsageChart from '$lib/components/UsageChart.svelte';

	let days = $state(30);
	let usage = $state<UsageSummary | null>(null);
	let error = $state('');

	$effect(() => {
		const period = days;
		getUsage(period)
			.then((u) => {
				if (period === days) {
					usage = u;
					error = '';
				}
			})
			.catch((e) => (error = (e as Error).message));
	});

	/** The cost, "—" when no call had a price, with "+" when some did not. */
	const cost = (g: AIUsageGroup) =>
		g.unpricedCalls === g.calls ? '—' : formatUsd(g.costUsd) + (g.unpricedCalls > 0 ? '+' : '');
	const periodLabel = (d: number) => (d === 0 ? 'All time' : d === 1 ? 'Today' : `${d} days`);

	/** Every day of the period, with the days without calls at zero. */
	let chartDays = $derived.by(() => {
		if (!usage) return [];
		const byKey = new Map(usage.byDay.map((d) => [d.key, d]));
		const first = usage.since ?? usage.byDay[0]?.key;
		if (!first) return [];
		const out: AIUsageGroup[] = [];
		const end = new Date(new Date().toISOString().slice(0, 10) + 'T00:00:00Z');
		for (let t = new Date(first.slice(0, 10) + 'T00:00:00Z'); t <= end; t.setUTCDate(t.getUTCDate() + 1)) {
			const key = t.toISOString().slice(0, 10);
			out.push(
				byKey.get(key) ?? {
					key,
					calls: 0,
					promptTokens: 0,
					completionTokens: 0,
					costUsd: 0,
					unpricedCalls: 0,
					fromLogCalls: 0
				}
			);
		}
		return out;
	});

	const tables = (u: UsageSummary) => [
		{ title: 'By action', rows: u.byAction, name: (g: AIUsageGroup) => g.label || g.key, link: () => '' },
		{ title: 'By model', rows: u.byModel, name: (g: AIUsageGroup) => g.key, link: () => '' },
		{
			title: 'By video',
			rows: u.byVideo,
			name: (g: AIUsageGroup) => (g.key ? g.label || g.key : 'Deleted videos'),
			link: (g: AIUsageGroup) => (g.key ? `/assets/${g.key}` : '')
		}
	];
</script>

<svelte:head><title>AI usage · UltraMedia</title></svelte:head>

<div class="title">
	<h1>AI usage</h1>
	<div class="periods" role="radiogroup" aria-label="Period">
		{#each usagePeriods as p (p)}
			<button
				class="period"
				class:on={days === p}
				role="radio"
				aria-checked={days === p}
				onclick={() => (days = p)}
				data-testid="period-{p}">{periodLabel(p)}</button
			>
		{/each}
	</div>
</div>
<p class="muted intro">
	Tokens and cost of every call to the language model: summaries, clip suggestions and translations. Transcription
	and face tracking run on this machine and cost nothing.
</p>

{#if error}
	<p class="card error">{error}</p>
{:else if !usage}
	<p class="muted">Loading…</p>
{:else if usage.total.calls === 0}
	<p class="card muted" data-testid="usage-empty">No calls to the language model in this period.</p>
{:else}
	<div class="tiles" data-testid="usage-total">
		<div class="tile"><span class="label">Cost</span><b>{formatUsd(usage.total.costUsd)}</b></div>
		<div class="tile"><span class="label">Tokens in</span><b>{formatTokens(usage.total.promptTokens)}</b></div>
		<div class="tile"><span class="label">Tokens out</span><b>{formatTokens(usage.total.completionTokens)}</b></div>
		<div class="tile"><span class="label">Calls</span><b>{usage.total.calls}</b></div>
	</div>
	{#if usage.total.unpricedCalls > 0}
		{@const fromLog = usage.total.fromLogCalls}
		{@const unknown = usage.total.unpricedCalls - fromLog}
		<p class="muted note" data-testid="usage-note">
			The cost leaves out
			{#if unknown > 0}
				{unknown} {unknown === 1 ? 'call' : 'calls'} the gateway had no price for{fromLog > 0 ? ', and' : ''}
			{/if}
			{#if fromLog > 0}
				{fromLog} {fromLog === 1 ? 'job' : 'jobs'} from before calls were recorded one by one, read back from {fromLog === 1
					? 'its log'
					: 'their logs'} without a cost
			{/if}. The real cost is higher.
		</p>
	{/if}

	<section class="card">
		<h2>Tokens per day</h2>
		<UsageChart days={chartDays} />
		<details>
			<summary>As a table</summary>
			<div class="scroll">
				<table>
					<thead><tr><th>Day (UTC)</th><th>Calls</th><th>Tokens in</th><th>Tokens out</th><th>Cost</th></tr></thead>
					<tbody>
						{#each usage.byDay as d (d.key)}
							<tr>
								<td>{d.key}</td><td>{d.calls}</td><td>{d.promptTokens.toLocaleString()}</td><td
									>{d.completionTokens.toLocaleString()}</td
								><td>{cost(d)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</details>
	</section>

	<div class="groups">
		{#each tables(usage) as t (t.title)}
			<section class="card" data-testid="usage-{t.title.toLowerCase().replace(' ', '-')}">
				<h2>{t.title}</h2>
				<div class="scroll">
					<table>
						<thead><tr><th></th><th>Calls</th><th>Tokens</th><th>Cost</th></tr></thead>
						<tbody>
							{#each t.rows as g (g.key)}
								<tr>
									<td class="name">
										{#if t.link(g)}<a href={t.link(g)}>{t.name(g)}</a>{:else}{t.name(g)}{/if}
									</td>
									<td>{g.calls}</td>
									<td>{formatTokens(g.promptTokens + g.completionTokens)}</td>
									<td title={g.unpricedCalls > 0 ? `${g.unpricedCalls} without a price` : ''}>{cost(g)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</section>
		{/each}
	</div>
{/if}

<style>
	.title {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
	}
	.periods {
		display: flex;
		flex-wrap: wrap;
		gap: 0.3rem;
	}
	.period {
		font: inherit;
		font-size: 0.85rem;
		padding: 0.25rem 0.7rem;
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		cursor: pointer;
	}
	.period.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
		font-weight: 600;
	}
	.intro {
		margin: 0.4rem 0 1rem;
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
		gap: 0.75rem;
		margin-bottom: 0.5rem;
	}
	.tile {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		padding: 0.75rem 0.9rem;
		display: grid;
		gap: 0.15rem;
	}
	.tile .label {
		font-size: 0.8rem;
		color: var(--muted);
	}
	.tile b {
		font-size: clamp(1.1rem, 5vw, 1.4rem);
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}
	.note {
		font-size: 0.85rem;
		margin: 0 0 0.75rem;
	}
	section.card {
		display: grid;
		gap: 0.75rem;
		margin-top: 0.75rem;
		min-width: 0;
	}
	.groups {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 0.75rem;
	}
	.scroll {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.88rem;
		font-variant-numeric: tabular-nums;
	}
	th,
	td {
		text-align: right;
		padding: 0.35rem 0.4rem;
		border-bottom: 1px solid var(--border);
		white-space: nowrap;
	}
	th:first-child,
	td:first-child {
		text-align: left;
	}
	td.name {
		white-space: normal;
		overflow-wrap: anywhere;
	}
	th {
		font-size: 0.75rem;
		color: var(--muted);
		font-weight: 600;
	}
	details summary {
		cursor: pointer;
		font-size: 0.85rem;
		color: var(--muted);
	}
</style>
