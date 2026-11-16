// A Markdoc render tree after JSON serialization: tags lose their class, so they are plain objects.
type RenderTag = {
	$$mdtype: 'Tag';
	name: string;
	attributes: Record<string, unknown>;
	children: RenderNode[];
};

type RenderNode = string | number | boolean | null | undefined | RenderTag;

type PostMeta = {
	slug: string;
	title: string;
	description: string;
	date: string;
	draft: boolean;
	og?: string;
	tags: string[];
	minutes: number;
};

type TocItem = { id: string; text: string; level: 2 | 3 };

type ChartDatum = { label: string; value?: number; values?: number[] };

export type { ChartDatum, PostMeta, RenderNode, RenderTag, TocItem };
