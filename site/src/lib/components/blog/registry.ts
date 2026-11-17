import Art from './art.svelte';
import Callout from './callout.svelte';
import Chart from './chart.svelte';
import CodeBlock from './code-block.svelte';
import ComparePane from './compare-pane.svelte';
import Compare from './compare.svelte';
import Episodes from './episodes.svelte';
import EvalExplorer from './eval-explorer.svelte';
import Figure from './figure.svelte';
import HookDemo from './hook-demo.svelte';
import InstallBlock from './install-block.svelte';
import Missing from './missing.svelte';
import Proverb from './proverb.svelte';
import Reveal from './reveal.svelte';
import Sidenote from './sidenote.svelte';
import Stat from './stat.svelte';
import Stats from './stats.svelte';
import Step from './step.svelte';
import Steps from './steps.svelte';
import TableWrap from './table-wrap.svelte';
import TerminalBlock from './terminal-block.svelte';
import Video from './video.svelte';

import type { Component } from 'svelte';

// Maps the render names in src/lib/server/blog/schema.ts to components.
// The props come from Markdoc attributes, so the renderer cannot type them.
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- Props differ per component.
const COMPONENTS: Record<string, Component<any>> = {
	Art,
	Callout,
	Chart,
	CodeBlock,
	Compare,
	ComparePane,
	Episodes,
	EvalExplorer,
	Figure,
	HookDemo,
	InstallBlock,
	Missing,
	Proverb,
	Reveal,
	Sidenote,
	Stat,
	Stats,
	Step,
	Steps,
	TableWrap,
	TerminalBlock,
	Video
};

export { COMPONENTS };
