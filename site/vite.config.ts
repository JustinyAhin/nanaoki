import adapter from '@sveltejs/adapter-cloudflare';
import { sveltekit } from '@sveltejs/kit/vite';
import { readFileSync } from 'node:fs';
import { defineConfig } from 'vite';
import { parse as parseYaml } from 'yaml';

import type { Plugin } from 'vite';

// Loads blog posts (.mdoc) as raw text. The site deploys on every push, so a production build
// replaces each draft with null: draft text never reaches the client, the worker or a prerendered page.
const blogPosts = (): Plugin => {
	let build = false;

	return {
		name: 'blog-posts',
		enforce: 'pre',
		configResolved(config) {
			build = config.command === 'build';
		},
		load(id) {
			if (!id.endsWith('.mdoc')) return;

			const source = readFileSync(id, 'utf8');
			const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/.exec(source)?.[1] ?? '';
			const draft = parseYaml(frontmatter)?.draft === true;

			return build && draft ? 'export default null;' : `export default ${JSON.stringify(source)};`;
		}
	};
};

export default defineConfig({
	plugins: [
		blogPosts(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter(),
			// No page links to the sitemap or the feed, and the header hides Blog until a post is published.
			prerender: {
				entries: ['*', '/sitemap.xml', '/blog/', '/blog/rss.xml'],
				// With every post still a draft, no post page exists yet. Any other unseen route fails.
				handleUnseenRoutes: ({ routes, message }) => {
					if (routes.some((route) => route !== '/blog/[slug]')) throw new Error(message);
				}
			}
		})
	]
});
