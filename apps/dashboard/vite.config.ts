import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [sveltekit()],
	// Unit tests of the rules and state the pages show (docs/adr/0037).
	test: { include: ['src/**/*.test.ts'] }
});
