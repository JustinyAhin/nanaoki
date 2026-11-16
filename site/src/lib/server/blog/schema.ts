import Markdoc from '@markdoc/markdoc';

import { ART } from '#lib/blog/art/index.ts';
import { highlight } from './highlight.ts';
import { POST_TAGS } from './post-tags.ts';

import type { Config, Node, RenderableTreeNode, Schema, ValidationError } from '@markdoc/markdoc';

const { Tag } = Markdoc;

const WIDTHS = ['text', 'wide', 'full'];
const width = (fallback = 'text') => ({ type: String, default: fallback, matches: WIDTHS });

const problem = ({ id, message }: { id: string; message: string }): ValidationError => ({
	id,
	level: 'error',
	message
});

const textOf = (node: Node) => {
	let text = '';
	for (const child of node.walk()) {
		if (child.type === 'text' || child.type === 'code') text += String(child.attributes.content);
	}
	return text;
};

const slugify = (text: string) =>
	text
		.toLowerCase()
		.replace(/[^a-z0-9\s-]/g, '')
		.trim()
		.replace(/[\s-]+/g, '-');

const findChild = ({ node, type }: { node: Node; type: string }) => {
	for (const child of node.walk()) if (child.type === type) return child;
	return undefined;
};

const isTag = (child: RenderableTreeNode, name: string): child is InstanceType<typeof Tag> =>
	Tag.isTag(child) && child.name === name;

// A sidenote written on its own line gets a paragraph. Unwrap it, because the note renders inline.
const unwrapParagraphs = (children: RenderableTreeNode[]) =>
	children.flatMap((child) => (isTag(child, 'p') ? child.children : [child]));

const validateChart = (node: Node): ValidationError[] => {
	const { type, data, series } = node.attributes as {
		type: string;
		data: unknown;
		series?: unknown;
	};
	if (!Array.isArray(data) || data.length === 0) {
		return [problem({ id: 'chart-data', message: 'chart: data must be a non-empty array' })];
	}

	const errors: ValidationError[] = [];
	if (Array.isArray(series) && series.length > 4) {
		errors.push(problem({ id: 'chart-series', message: 'chart: use 4 series or fewer' }));
	}
	const multi = type === 'stacked' || Array.isArray(series);
	if (type === 'stacked' && !Array.isArray(series)) {
		errors.push(problem({ id: 'chart-series', message: 'chart: stacked needs series=[...]' }));
	}

	data.forEach((datum, index) => {
		const row = datum as { label?: unknown; value?: unknown; values?: unknown };
		const at = `chart: data[${index}]`;
		if (typeof row?.label !== 'string') {
			errors.push(problem({ id: 'chart-label', message: `${at} needs a string label` }));
		}
		if (!multi && typeof row?.value !== 'number') {
			errors.push(problem({ id: 'chart-value', message: `${at} needs a number value` }));
		}
		if (multi) {
			const values = row?.values;
			const ok =
				Array.isArray(values) &&
				values.every((value) => typeof value === 'number') &&
				(!Array.isArray(series) || values.length === series.length);
			if (!ok) {
				errors.push(
					problem({
						id: 'chart-values',
						message: `${at} needs values=[...] with one number for each series`
					})
				);
			}
		}
	});

	return errors;
};

// Builds a fresh config for each render, because heading ids and sidenote numbers count per post.
const createConfig = ({ dev }: { dev: boolean }): Config => {
	const ids = new Map<string, number>();
	let sidenotes = 0;

	const nodes: Record<string, Schema> = {
		heading: {
			children: ['inline'],
			attributes: { id: { type: String }, level: { type: Number, required: true } },
			transform(node, config) {
				const attributes = node.transformAttributes(config);
				const base = (attributes.id as string | undefined) ?? (slugify(textOf(node)) || 'section');
				const count = ids.get(base) ?? 0;
				ids.set(base, count + 1);

				const id = count ? `${base}-${count}` : base;
				return new Tag(`h${node.attributes.level}`, { id }, node.transformChildren(config));
			}
		},
		fence: {
			attributes: { content: { type: String }, language: { type: String }, process: {} },
			transform(node) {
				const { content, language } = node.attributes as { content: string; language?: string };
				return new Tag('CodeBlock', {
					code: content.replace(/\n$/, ''),
					language: language ?? '',
					html: highlight({ code: content, language })
				});
			}
		},
		table: {
			transform(node, config) {
				return new Tag('TableWrap', {}, [new Tag('table', {}, node.transformChildren(config))]);
			}
		}
	};

	const tags: Record<string, Schema> = {
		callout: {
			render: 'Callout',
			attributes: {
				type: { type: String, default: 'note', matches: ['note', 'warn', 'quote'] },
				title: { type: String },
				cite: { type: String }
			}
		},
		sidenote: {
			inline: true,
			transform(node, config) {
				sidenotes += 1;
				const children = unwrapParagraphs(node.transformChildren(config));
				return new Tag('Sidenote', { n: sidenotes }, children);
			}
		},
		figure: {
			render: 'Figure',
			attributes: {
				src: { type: String },
				alt: { type: String },
				caption: { type: String },
				width: width()
			},
			validate(node) {
				const { src, alt } = node.attributes;
				if (src && !alt) return [problem({ id: 'figure-alt', message: 'figure: src needs alt' })];
				if (!src && node.children.length === 0) {
					return [problem({ id: 'figure-empty', message: 'figure: needs src or children' })];
				}
				return [];
			}
		},
		terminal: {
			attributes: {
				title: { type: String, default: 'terminal' },
				label: { type: String },
				width: width()
			},
			validate(node) {
				return findChild({ node, type: 'fence' })
					? []
					: [
							problem({ id: 'terminal-fence', message: 'terminal: put the output in a code fence' })
						];
			},
			transform(node, config) {
				const fence = findChild({ node, type: 'fence' });
				const attributes = node.transformAttributes(config);
				const text = String(fence?.attributes.content ?? '').replace(/\n$/, '');
				return new Tag('TerminalBlock', { ...attributes, text });
			}
		},
		stats: {
			attributes: { width: width() },
			transform(node, config) {
				const children = node.transformChildren(config).filter((child) => isTag(child, 'Stat'));
				const sources: string[] = [];
				for (const child of children) {
					if (!isTag(child, 'Stat')) continue;
					child.attributes.standalone = false;
					if (child.attributes.source) {
						sources.push(String(child.attributes.source));
						child.attributes.note = sources.length;
					}
				}
				return new Tag('Stats', { ...node.transformAttributes(config), sources }, children);
			}
		},
		stat: {
			selfClosing: true,
			attributes: {
				value: { type: [String, Number], required: true },
				label: { type: String, required: true },
				source: { type: String }
			},
			transform(node, config) {
				return new Tag('Stat', { ...node.transformAttributes(config), standalone: true });
			}
		},
		chart: {
			render: 'Chart',
			selfClosing: true,
			attributes: {
				type: { type: String, required: true, matches: ['bar', 'line', 'stacked'] },
				data: { type: Array, required: true },
				series: { type: Array },
				title: { type: String },
				caption: { type: String },
				unit: { type: String, default: '' },
				source: { type: String },
				highlight: { type: String },
				width: width()
			},
			validate: validateChart
		},
		compare: {
			attributes: { label: { type: String, default: 'Compare' }, width: width() },
			validate(node) {
				const panes = node.children.filter((child) => child.tag === 'pane');
				return panes.length === 2
					? []
					: [problem({ id: 'compare-panes', message: 'compare: needs exactly two pane tags' })];
			},
			transform(node, config) {
				const panes = node.children.filter((child) => child.tag === 'pane');
				const labels = panes.map((pane) => String(pane.attributes.label));
				return new Tag(
					'Compare',
					{ ...node.transformAttributes(config), labels },
					panes.map(
						(pane, index) =>
							new Tag('ComparePane', { label: labels[index] }, pane.transformChildren(config))
					)
				);
			}
		},
		pane: {
			attributes: { label: { type: String, required: true } }
		},
		steps: {
			validate(node) {
				return findChild({ node, type: 'list' })
					? []
					: [problem({ id: 'steps-list', message: 'steps: write the steps as a list' })];
			},
			transform(node, config) {
				const list = findChild({ node, type: 'list' });
				const items = list?.children.filter((child) => child.type === 'item') ?? [];
				return new Tag(
					'Steps',
					{},
					items.map(
						(item, index) => new Tag('Step', { n: index + 1 }, item.transformChildren(config))
					)
				);
			}
		},
		art: {
			render: 'Art',
			selfClosing: true,
			attributes: {
				name: { type: String, required: true, matches: Object.keys(ART) },
				height: { type: [Number, String], default: 320 },
				caption: { type: String },
				seed: { type: Number, default: 1 },
				width: width()
			}
		},
		...POST_TAGS,
		reveal: {
			render: 'Reveal'
		},
		// Dev only: an unknown tag shows as a placeholder, so a draft can use a component before it exists.
		missing: {
			attributes: { name: { type: String } },
			transform(node) {
				return new Tag('Missing', { name: node.attributes.name, dev });
			}
		}
	};

	return { nodes, tags };
};

export { createConfig, slugify };
