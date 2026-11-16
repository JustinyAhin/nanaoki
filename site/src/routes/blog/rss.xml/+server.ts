import { toDate } from '#lib/blog/format.ts';
import { canonicalUrl } from '#lib/seo.ts';
import { listPosts } from '#lib/server/blog/posts.ts';
import { SITE } from '#lib/site.ts';

export const prerender = true;

const escape = (text: string) =>
	text
		.replaceAll('&', '&amp;')
		.replaceAll('<', '&lt;')
		.replaceAll('>', '&gt;')
		.replaceAll('"', '&quot;');

export const GET = () => {
	const posts = listPosts();
	const home = canonicalUrl({ path: '/blog/' });
	const items = posts.map((post) => {
		const url = canonicalUrl({ path: `/blog/${post.slug}/` });
		return [
			'<item>',
			`<title>${escape(post.title)}</title>`,
			`<link>${url}</link>`,
			`<guid isPermaLink="true">${url}</guid>`,
			`<description>${escape(post.description)}</description>`,
			`<pubDate>${toDate(post.date).toUTCString()}</pubDate>`,
			...post.tags.map((tag) => `<category>${escape(tag)}</category>`),
			'</item>'
		].join('');
	});
	const updated = posts[0] ? toDate(posts[0].date) : new Date(0);

	const xml = [
		'<?xml version="1.0" encoding="UTF-8"?>',
		'<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom"><channel>',
		`<title>${SITE.name} blog</title>`,
		`<link>${home}</link>`,
		`<atom:link href="${canonicalUrl({ path: '/blog/rss.xml' }).replace(/\/$/, '')}" rel="self" type="application/rss+xml"/>`,
		'<description>Notes from building nanaoki.</description>',
		'<language>en</language>',
		`<lastBuildDate>${updated.toUTCString()}</lastBuildDate>`,
		...items,
		'</channel></rss>'
	].join('\n');

	return new Response(xml, { headers: { 'content-type': 'application/rss+xml; charset=utf-8' } });
};
