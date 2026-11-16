import { createHighlighterCoreSync } from 'shiki/core';
import { createJavaScriptRegexEngine } from 'shiki/engine/javascript';
import diff from 'shiki/langs/diff.mjs';
import go from 'shiki/langs/go.mjs';
import javascript from 'shiki/langs/javascript.mjs';
import json from 'shiki/langs/json.mjs';
import shellscript from 'shiki/langs/shellscript.mjs';
import toml from 'shiki/langs/toml.mjs';
import typescript from 'shiki/langs/typescript.mjs';
import yaml from 'shiki/langs/yaml.mjs';
import githubDark from 'shiki/themes/github-dark.mjs';

import type { HighlighterCore } from 'shiki/core';

// A fixed, small language set keeps the build fast. Other languages render as plain text.
// The JavaScript regex engine avoids the WASM build of Oniguruma.
let highlighter: HighlighterCore | undefined;

const getHighlighter = () =>
	(highlighter ??= createHighlighterCoreSync({
		themes: [githubDark],
		langs: [diff, go, javascript, json, shellscript, toml, typescript, yaml],
		engine: createJavaScriptRegexEngine()
	}));

const highlight = ({ code, language }: { code: string; language?: string }) => {
	const shiki = getHighlighter();
	const lang = language && shiki.getLoadedLanguages().includes(language) ? language : 'text';

	return shiki.codeToHtml(code.replace(/\n$/, ''), { lang, theme: 'github-dark' });
};

export { highlight };
