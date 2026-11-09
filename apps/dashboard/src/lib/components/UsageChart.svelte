<script lang="ts">
	import type { AIUsageGroup } from '$lib/api';
	import { formatTokens, formatUsd } from '$lib/format';

	/** Tokens per day, one bar per day (days without calls included, at zero). */
	let { days }: { days: AIUsageGroup[] } = $props();

	let width = $state(600);
	const height = 190;
	const m = { top: 10, right: 8, bottom: 24, left: 44 };
	let plotW = $derived(Math.max(width - m.left - m.right, 10));
	const plotH = height - m.top - m.bottom;

	const tokens = (d: AIUsageGroup) => d.promptTokens + d.completionTokens;
	/** A clean top for the axis: 1, 2 or 5 times a power of ten. */
	function niceMax(v: number): number {
		if (v <= 0) return 1;
		const p = 10 ** Math.floor(Math.log10(v));
		return ([1, 2, 5, 10].find((s) => s * p >= v) ?? 10) * p;
	}
	let top = $derived(niceMax(Math.max(0, ...days.map(tokens))));
	let ticks = $derived([0, top / 2, top]);
	let slot = $derived(plotW / Math.max(days.length, 1));
	let barW = $derived(Math.max(Math.min(24, slot - 2), 1));
	const y = (v: number) => m.top + plotH - (v / top) * plotH;

	/** A column with a 4px rounded top, square at the baseline. */
	function bar(x: number, h: number): string {
		const r = Math.min(4, h, barW / 2);
		const y0 = m.top + plotH;
		const y1 = y0 - h;
		return `M${x},${y0}V${y1 + r}Q${x},${y1} ${x + r},${y1}H${x + barW - r}Q${x + barW},${y1} ${x + barW},${y1 + r}V${y0}Z`;
	}

	const dayLabel = (key: string) =>
		new Date(key + 'T00:00:00Z').toLocaleDateString(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' });
	let labelled = $derived(
		days.length <= 2 ? days.map((_, i) => i) : [0, Math.floor((days.length - 1) / 2), days.length - 1]
	);

	let active = $state<number | null>(null);
</script>

<div class="chart" bind:clientWidth={width}>
	<svg {width} {height} role="img" aria-label="Tokens per day">
		{#each ticks as t (t)}
			<line class="grid" x1={m.left} x2={m.left + plotW} y1={y(t)} y2={y(t)} />
			<text class="axis" x={m.left - 6} y={y(t)} text-anchor="end" dominant-baseline="middle">{formatTokens(t)}</text>
		{/each}
		{#each days as d, i (d.key)}
			{@const h = (tokens(d) / top) * plotH}
			{#if h > 0}
				<path class="bar" class:dim={active !== null && active !== i} d={bar(m.left + i * slot + (slot - barW) / 2, h)} />
			{/if}
			<!-- The whole column is the hover and focus target. -->
			<rect
				class="hit"
				x={m.left + i * slot}
				y={m.top}
				width={slot}
				height={plotH}
				tabindex="0"
				role="button"
				aria-label="{dayLabel(d.key)}: {formatTokens(tokens(d))} tokens, {formatUsd(d.costUsd)}"
				onpointerenter={() => (active = i)}
				onpointerleave={() => (active = null)}
				onfocus={() => (active = i)}
				onblur={() => (active = null)}
			/>
		{/each}
		{#each labelled as i (i)}
			<text
				class="axis"
				x={m.left + i * slot + slot / 2}
				y={height - 6}
				text-anchor={labelled.length > 1 && i === days.length - 1 ? 'end' : i === 0 && labelled.length > 1 ? 'start' : 'middle'}
				>{dayLabel(days[i].key)}</text
			>
		{/each}
	</svg>
	{#if active !== null}
		{@const d = days[active]}
		<div
			class="tip"
			style="left: {Math.min(Math.max(m.left + active * slot + slot / 2, 90), width - 90)}px"
			data-testid="usage-tip"
		>
			<strong>{dayLabel(d.key)}</strong>
			<span>{formatTokens(d.promptTokens)} in · {formatTokens(d.completionTokens)} out</span>
			<span>{d.calls} {d.calls === 1 ? 'call' : 'calls'} · {formatUsd(d.costUsd)}</span>
		</div>
	{/if}
</div>

<style>
	.chart {
		position: relative;
		width: 100%;
		min-width: 0; /* as a grid item, follow the container, not the drawing */
	}
	svg {
		display: block;
		overflow: visible;
	}
	.grid {
		stroke: var(--border);
		stroke-width: 1;
	}
	.axis {
		fill: var(--muted);
		font-size: 11px;
		font-variant-numeric: tabular-nums;
	}
	.bar {
		fill: var(--chart-bar);
		transition: opacity 0.15s;
	}
	.bar.dim {
		opacity: 0.45;
	}
	.hit {
		fill: transparent;
		cursor: default;
		outline: none;
	}
	.hit:focus-visible {
		stroke: var(--accent);
		stroke-width: 1;
	}
	.tip {
		position: absolute;
		top: -6px;
		transform: translate(-50%, -100%);
		display: grid;
		gap: 0.1rem;
		padding: 0.4rem 0.6rem;
		border-radius: 8px;
		border: 1px solid var(--border);
		background: var(--surface);
		box-shadow: 0 6px 18px rgba(0, 0, 0, 0.12);
		font-size: 0.8rem;
		white-space: nowrap;
		pointer-events: none;
		font-variant-numeric: tabular-nums;
	}
	@media (prefers-reduced-motion: reduce) {
		.bar {
			transition: none;
		}
	}
</style>
