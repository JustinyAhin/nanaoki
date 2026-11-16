/** @type {import("prettier").Config} */
const config = {
	useTabs: true,
	singleQuote: true,
	trailingComma: 'none',
	printWidth: 100,
	plugins: ['prettier-plugin-svelte'],
	// Keep the default whitespace handling: the terminal blocks use pre-wrap.
	overrides: [{ files: '*.svelte', options: { parser: 'svelte' } }]
};

export default config;
