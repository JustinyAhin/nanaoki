import { CASES } from '#lib/cases.ts';

import type { Node, Schema, ValidationError } from '@markdoc/markdoc';

// Custom tags of the announcement post. Each component holds no numbers of its own: the data is in
// the post, and these checks fail the build when its shape is wrong.

const problem = (message: string): ValidationError => ({ id: 'post-tag', level: 'error', message });

const isNumber = (value: unknown) => typeof value === 'number' && Number.isFinite(value);

const validateHookDemo = (node: Node) => {
	const cases = node.attributes.cases as unknown[];
	const known = new Set(CASES.map((item) => item.id));
	return cases
		.filter((id) => typeof id !== 'string' || !known.has(id))
		.map((id) => problem(`hook-demo: unknown case "${String(id)}"`));
};

const validateEvalExplorer = (node: Node) => {
	const { setups, tokens, slices } = node.attributes as {
		setups: unknown[];
		tokens?: unknown[];
		slices: unknown[];
	};
	const errors: ValidationError[] = [];
	const count = setups.length;

	if (!setups.every((name) => typeof name === 'string')) {
		errors.push(problem('eval-explorer: setups must be strings'));
	}
	if (tokens && (tokens.length !== count || !tokens.every(isNumber))) {
		errors.push(problem('eval-explorer: tokens needs one number for each setup'));
	}

	slices.forEach((raw, index) => {
		const slice = raw as Record<string, unknown>;
		const at = `eval-explorer: slices[${index}]`;
		const values = slice.values as unknown[] | undefined;
		const ci = slice.ci as unknown[] | undefined;
		const extra = slice.extra as Record<string, unknown> | undefined;

		if (typeof slice.id !== 'string' || typeof slice.label !== 'string') {
			errors.push(problem(`${at} needs a string id and label`));
		}
		if (typeof slice.judge !== 'string' || !isNumber(slice.n)) {
			errors.push(problem(`${at} needs a string judge and a number n`));
		}
		if (
			!Array.isArray(values) ||
			values.length !== count ||
			!values.every((value) => value === null || isNumber(value))
		) {
			errors.push(problem(`${at} needs values with one number or null for each setup`));
		}
		const isPair = (pair: unknown) =>
			pair === null || (Array.isArray(pair) && pair.length === 2 && pair.every(isNumber));
		if (ci !== undefined && (!Array.isArray(ci) || ci.length !== count || !ci.every(isPair))) {
			errors.push(problem(`${at} needs ci with one [low, high] or null for each setup`));
		}
		if (extra !== undefined && (typeof extra.label !== 'string' || !isNumber(extra.value))) {
			errors.push(problem(`${at} needs extra as {label, value}`));
		}
		if (slice.caption !== undefined && typeof slice.caption !== 'string') {
			errors.push(problem(`${at} caption must be a string`));
		}
	});

	return errors;
};

const validateEpisodes = (node: Node) => {
	const { total, repeats, crossProject, sameProject } = node.attributes as Record<string, number>;
	const errors: ValidationError[] = [];
	if (repeats > total || crossProject > repeats) {
		errors.push(problem('episodes: needs crossProject <= repeats <= total'));
	}
	if (sameProject !== undefined && crossProject + sameProject !== repeats) {
		errors.push(problem('episodes: crossProject + sameProject must equal repeats'));
	}
	return errors;
};

const POST_TAGS: Record<string, Schema> = {
	'hook-demo': {
		render: 'HookDemo',
		selfClosing: true,
		attributes: { cases: { type: Array, required: true } },
		validate: validateHookDemo
	},
	'eval-explorer': {
		render: 'EvalExplorer',
		selfClosing: true,
		attributes: {
			setups: { type: Array, required: true },
			tokens: { type: Array, default: [] },
			slices: { type: Array, required: true }
		},
		validate: validateEvalExplorer
	},
	episodes: {
		render: 'Episodes',
		selfClosing: true,
		attributes: {
			total: { type: Number, required: true },
			repeats: { type: Number, required: true },
			crossProject: { type: Number, required: true },
			sameProject: { type: Number }
		},
		validate: validateEpisodes
	},
	install: {
		render: 'InstallBlock',
		selfClosing: true
	},
	proverb: {
		render: 'Proverb',
		selfClosing: true
	}
};

export { POST_TAGS };
