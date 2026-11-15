# nanaoki.com

Static landing page. One `index.html` with inline CSS and JS, plus `assets/`. No build step.

## Preview

```sh
python3 -m http.server 8000   # then open http://localhost:8000
```

## Deploy

Push deploys. Cloudflare Workers Builds watches the public repo `JustinyAhin/nanaoki` (root directory `/site/`, deploy command `bunx wrangler deploy`) and deploys `nanaoki-site` on each push to `main` that changes `site/`. In the private repo, edit `site/`, commit, then run `scripts/sync_public.sh "message"`.

`wrangler.toml` serves this folder as static assets on nanaoki.com and www.nanaoki.com, with Workers Logs and traces on. `.assetsignore` keeps the docs, config, card sources and preview screenshots out of the upload.

Manual deploy (fallback): `cd site && bunx wrangler deploy`.

## Notes

- Screenshots in `assets/` are copies of `public/assets/` (already anonymized with `--anon`). Do not add new screenshots of real output without `--anon`.
- `og:image` is `assets/og.png`, drawn in code: `og/v2-ab.html`, rendered with `og/render-v2.sh ab`, then copied to `assets/og.png`. Bump `?v=N` in the meta tags when it changes.
- `preview-desktop.png` and `preview-mobile.png` are review screenshots only.
