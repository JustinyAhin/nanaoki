import { canonicalUrl } from '#lib/seo.ts';

export const prerender = true;

const PATHS = ['/', '/cases/'];

export const GET = () => {
	const urls = PATHS.map((path) => `<url><loc>${canonicalUrl({ path })}</loc></url>`);

	return new Response(
		`<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.join('')}</urlset>`,
		{ headers: { 'content-type': 'application/xml; charset=utf-8' } }
	);
};
