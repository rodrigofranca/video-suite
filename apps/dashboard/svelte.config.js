import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		// The dashboard is a single-page app (ssr = false): every route falls
		// back to index.html, so any static file server can host the build.
		// PUBLIC_API_URL is read when building.
		adapter: adapter({ fallback: 'index.html' })
	}
};

export default config;
