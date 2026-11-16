import { dev } from '$app/env';
import Markdoc from '@markdoc/markdoc';
import { parse as parseYaml } from 'yaml';

import { createConfig } from './schema.ts';

import type { PostMeta, RenderNode, RenderTag, TocItem } from '#lib/blog/types.ts';
import type { Node, ValidateError } from '@markdoc/markdoc';

// The blog-posts plugin in vite.config.ts replaces a draft with null in a production build,
// so draft text never reaches the build output.
const FILES = import.meta.glob<string | null>('/src/content/blog/*.mdoc', {
	eager: true,
	import: 'default'
});

const WORDS_PER_MINUTE = 230;

type Post = { meta: PostMeta; source: string; file: string };

const fail = ({ file, message }: { file: string; message: string }): never => {
	throw new Error(`${file}: ${message}`);
};

const readMeta = ({ file, ast }: { file: string; ast: Node }) => {
	const slug = file
		.split('/')
		.pop()!
		.replace(/\.mdoc$/, '');
	const raw = parseYaml(String(ast.attributes.frontmatter ?? '')) ?? {};
	const date = raw.date instanceof Date ? raw.date.toISOString().slice(0, 10) : raw.date;

	if (!/^[a-z0-9]+(-[a-z0-9]+)*$/.test(slug)) fail({ file, message: 'use a kebab-case file name' });
	if (typeof raw.title !== 'string' || !raw.title) fail({ file, message: 'title is required' });
	if (typeof raw.description !== 'string') fail({ file, message: 'description is required' });
	if (typeof date !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(date)) {
		fail({ file, message: 'date must be YYYY-MM-DD' });
	}
	if (raw.draft !== undefined && typeof raw.draft !== 'boolean') {
		fail({ file, message: 'draft must be true or false' });
	}

	let words = 0;
	for (const node of ast.walk()) {
		if (node.type === 'text' || node.type === 'code' || node.type === 'fence') {
			words += String(node.attributes.content).split(/\s+/).filter(Boolean).length;
		}
	}

	const meta: PostMeta = {
		slug,
		title: raw.title,
		description: raw.description,
		date,
		draft: raw.draft === true,
		og: typeof raw.og === 'string' ? raw.og : undefined,
		tags: Array.isArray(raw.tags) ? raw.tags.map(String) : [],
		minutes: Math.max(1, Math.round(words / WORDS_PER_MINUTE))
	};
	return meta;
};

let cache: Post[] | undefined;

const loadPosts = () =>
	(cache ??= Object.entries(FILES)
		.flatMap(([file, source]) => {
			if (source === null) return [];
			const ast = Markdoc.parse(source);
			const meta = readMeta({ file, ast });
			if (meta.draft && !dev) return [];
			return [{ meta, source, file }];
		})
		.sort(
			(a, b) => b.meta.date.localeCompare(a.meta.date) || a.meta.slug.localeCompare(b.meta.slug)
		));

const listPosts = () => loadPosts().map((post) => post.meta);

const textOf = (node: RenderNode): string => {
	if (node === null || node === undefined || typeof node === 'boolean') return '';
	if (typeof node !== 'object') return String(node);
	return node.children.map(textOf).join('');
};

const describe = ({ file, error }: { file: string; error: ValidateError }) =>
	`${file}:${error.lines[0] + 1} ${error.error.message}`;

const renderPost = ({ slug }: { slug: string }) => {
	const post = loadPosts().find((item) => item.meta.slug === slug);
	if (!post) return undefined;

	// Parse again for each render: the dev pass below changes the tree.
	const ast = Markdoc.parse(post.source);
	const config = createConfig({ dev });
	const errors = Markdoc.validate(ast, config).filter(
		(error) => error.error.level === 'error' || error.error.level === 'critical'
	);

	// A broken post fails the build. In dev it renders with the errors on top.
	if (errors.length && !dev) {
		throw new Error(errors.map((error) => describe({ file: post.file, error })).join('\n'));
	}

	if (dev) {
		for (const node of ast.walk()) {
			if (node.type === 'tag' && node.tag && !config.tags?.[node.tag]) {
				node.attributes = { name: node.tag };
				node.tag = 'missing';
			}
		}
	}

	const root = JSON.parse(JSON.stringify(Markdoc.transform(ast, config))) as RenderTag;
	const toc: TocItem[] = root.children.flatMap((child) =>
		child && typeof child === 'object' && (child.name === 'h2' || child.name === 'h3')
			? [
					{
						id: String(child.attributes.id),
						text: textOf(child),
						level: child.name === 'h2' ? 2 : 3
					}
				]
			: []
	);

	return {
		meta: post.meta,
		nodes: root.children,
		toc,
		errors: errors.map((error) => describe({ file: post.file, error }))
	};
};

export { listPosts, renderPost };
