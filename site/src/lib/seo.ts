import { SITE } from './site';

type StructuredData = Record<string, unknown>;

const absoluteUrl = ({ path, origin = SITE.origin }: { path: string; origin?: string }) =>
	new URL(path, origin).href;

const canonicalUrl = ({ path }: { path: string }) => {
	const url = new URL(path, SITE.origin);

	url.search = '';
	url.hash = '';
	url.pathname = `${url.pathname.replace(/\/+$/, '')}/`;

	return url.href;
};

// Escaping angle brackets prevents JSON values from closing the HTML script element.
const serializeJsonLd = (data: StructuredData) =>
	JSON.stringify(data).replaceAll('<', '\\u003c').replaceAll('>', '\\u003e');

export { absoluteUrl, canonicalUrl, serializeJsonLd };
export type { StructuredData };
