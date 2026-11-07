<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>UltraMedia</title>
</svelte:head>

<header>
	<a class="brand" href="/assets">UltraMedia</a>
	<nav>
		<a href="/assets" aria-current={page.url.pathname.startsWith('/assets') ? 'page' : undefined}>Videos</a>
		<a class="button" href="/upload">Upload</a>
	</nav>
</header>

<main>
	{@render children()}
</main>

<style>
	:global(:root) {
		--bg: #f6f7f9;
		--surface: #ffffff;
		--text: #16181d;
		--muted: #6b7280;
		--border: #e3e5ea;
		--accent: #4f46e5;
		--accent-fg: #ffffff;
		--badge-neutral-bg: #eef0f3;
		--badge-neutral-fg: #4b5563;
		--badge-ok-bg: #dcfce7;
		--badge-ok-fg: #166534;
		--badge-busy-bg: #fef3c7;
		--badge-busy-fg: #92400e;
		--badge-err-bg: #fee2e2;
		--badge-err-fg: #b91c1c;
		color-scheme: light dark;
	}
	@media (prefers-color-scheme: dark) {
		:global(:root) {
			--bg: #0f1115;
			--surface: #171a21;
			--text: #e7e9ee;
			--muted: #9aa1ad;
			--border: #2a2f3a;
			--accent: #818cf8;
			--accent-fg: #0f1115;
			--badge-neutral-bg: #2a2f3a;
			--badge-neutral-fg: #c8ccd4;
			--badge-ok-bg: #12351f;
			--badge-ok-fg: #86efac;
			--badge-busy-bg: #3b2c0a;
			--badge-busy-fg: #fcd34d;
			--badge-err-bg: #3d1414;
			--badge-err-fg: #fca5a5;
		}
	}
	:global(body) {
		margin: 0;
		background: var(--bg);
		color: var(--text);
		font-family:
			system-ui,
			-apple-system,
			'Segoe UI',
			Roboto,
			sans-serif;
		line-height: 1.5;
	}
	:global(a) {
		color: var(--accent);
	}
	:global(h1) {
		font-size: 1.35rem;
		margin: 0;
	}
	:global(h2) {
		font-size: 1rem;
		margin: 0 0 0.75rem;
	}
	:global(.button),
	:global(button.primary) {
		display: inline-block;
		background: var(--accent);
		color: var(--accent-fg);
		border: 0;
		border-radius: 8px;
		padding: 0.5rem 1rem;
		font: inherit;
		font-weight: 600;
		text-decoration: none;
		cursor: pointer;
	}
	:global(button.primary:disabled) {
		opacity: 0.5;
		cursor: default;
	}
	:global(button.secondary) {
		font: inherit;
		font-size: 0.85rem;
		font-weight: 600;
		padding: 0.3rem 0.8rem;
		border-radius: 6px;
		border: 1px solid var(--accent);
		background: transparent;
		color: var(--accent);
		cursor: pointer;
	}
	:global(button.secondary:disabled) {
		opacity: 0.5;
		cursor: default;
	}
	:global(.card) {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		padding: 1rem;
	}
	:global(.muted) {
		color: var(--muted);
	}
	:global(.error) {
		color: var(--badge-err-fg);
	}
	header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		padding: 0.75rem 1rem;
		background: var(--surface);
		border-bottom: 1px solid var(--border);
	}
	.brand {
		font-weight: 700;
		color: var(--text);
		text-decoration: none;
	}
	nav {
		display: flex;
		align-items: center;
		gap: 1rem;
	}
	nav a:not(.button) {
		color: var(--muted);
		text-decoration: none;
	}
	nav a[aria-current='page'] {
		color: var(--text);
		font-weight: 600;
	}
	main {
		max-width: 960px;
		margin: 0 auto;
		padding: 1.25rem 1rem 3rem;
	}
</style>
