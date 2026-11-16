# nanaoki.com

SvelteKit 3 app with Svelte 5 and TypeScript, prerendered and served by Cloudflare Workers. Bun is the package manager.

## Code

- Use Svelte 5 runes. Use kebab-case filenames and named-object parameters.
- Import from `src/lib` with `#lib/...`. Add `.ts` to TypeScript module imports, for example `#lib/site.ts`.
- Keep component CSS in its Svelte file. Put only shared tokens, base elements and utilities in `src/lib/styles.css`.
- Extract a section into a component when this makes the page easier to read or reuse.
- Keep links, install commands and the social card data in `src/lib/site.ts`.
- Prerender all routes. URLs end with `/`. A `+server.ts` route needs its own `export const prerender = true`.
- Use `resolve()` from `$app/paths` for internal links. For an external URL or an asset URL from a variable, add an `eslint-disable-next-line` comment with the reason.
- Put files that need a fixed URL in `static/`. Import other images and fonts from `src/lib/assets/`, so Vite adds a content hash.
- Never edit `worker-configuration.d.ts`. `check` and `build` generate it, and Git ignores it.

## Whitespace

- Terminal and error blocks use `white-space: pre-wrap`. Keep each terminal body on one line under `<!-- prettier-ignore -->`.
- Use `{' '}` or `{'\n'}` where Svelte would trim a space or a line break.
- Do not put a line break between an inline tag and the punctuation after it. It renders as a space before the punctuation.
- After a text change, compare the rendered text with the live page.

## Content

- Use only numbers from the research files. Say "on my own data" for eval numbers.
- Screenshots must come from `nanaoki --anon`. Never show real project names, paths or session content.
- When `static/assets/og.png` changes, bump `?v=N` in `src/lib/site.ts`. The card source is `og/v2-ab.html`, rendered with `og/render-v2.sh ab`.
- The footer font is a 2 KB subset of Shippori Mincho Medium (SIL OFL 1.1) with only 七転び八起き. To change the characters, make it again:

  ```sh
  pyftsubset ShipporiMincho-Medium.ttf --text="七転び八起き" --flavor=woff2 \
    --layout-features='vkna,vert,vrt2' --output-file=src/lib/assets/fonts/shippori-mincho-500-subset.woff2
  ```

## Checks

Run `bun run lint`, `bun run check` and `bun run build` before each commit. `check` deletes the build output first, because svelte-check reports errors in compiled files. Stop temporary servers after verification.

## Deploy

Workers Builds deploys `nanaoki-site` on each push to `main` of the public repo that changes `site/`. Settings: root directory `/site/`, build command `bun install --frozen-lockfile && bun run build`, deploy command `bunx wrangler deploy`. Manual fallback: `bun run build && bunx wrangler deploy`.
