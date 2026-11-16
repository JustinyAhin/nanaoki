import { canonicalUrl } from '#lib/seo.ts';
import { listPosts } from '#lib/server/blog/posts.ts';

export const prerender = true;

export const GET = () => {
	const posts = listPosts();
	const pages = [
		{ path: '/' },
		{ path: '/cases/' },
		...(posts.length ? [{ path: '/blog/', lastmod: posts[0].date }] : []),
		...posts.map((post) => ({ path: `/blog/${post.slug}/`, lastmod: post.date }))
	];
	const urls = pages.map(
		({ path, lastmod }: { path: string; lastmod?: string }) =>
			`<url><loc>${canonicalUrl({ path })}</loc>${lastmod ? `<lastmod>${lastmod}</lastmod>` : ''}</url>`
	);

	return new Response(
		`<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.join('')}</urlset>`,
		{ headers: { 'content-type': 'application/xml; charset=utf-8' } }
	);
};
