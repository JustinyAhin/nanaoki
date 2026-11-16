<script module lang="ts">
	// Terminal bodies of the cases, shared by the cases page and the blog hook demo.
	export {
		zshGlob,
		zshEquals,
		nodeVersion,
		missingScript,
		sveltekitResolve,
		gitNoCommits,
		notJson,
		dontChangeYet,
		wrongLesson,
		codeComments
	};
</script>

<!-- The terminal keeps whitespace, so each body line must stay as written. -->
<!-- prettier-ignore -->
{#snippet zshGlob()}<span class="blk"><span class="g">●</span> Bash(<span class="c">ls -a ../web/.dev.vars*</span>)</span><span class="ind"><span class="d">└</span> <span class="y">nanaoki guard:</span> [zsh_glob] zsh fails with "no matches found" when an unquoted glob matches no file: `../web/.dev.vars*`. Quote words with [ ] * ? (URLs, [slug] routes, --include=*.md). Suggested command: <span class="c">ls -a '../web/.dev.vars*'</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet zshEquals()}<span class="blk"><span class="g">●</span> Bash(<span class="c">ax "https://jobs.example.com/listing/1234" --md --budget 1500; echo =====; ax "https://jobs.example.com/listing/5678" --md</span>)</span><span class="ind"><span class="d">└</span> <span class="r">Blocked by hook.</span> <span class="y">nanaoki guard:</span> [zsh_eq] In zsh an unquoted word that starts with '=' expands to a command path, so `=====` fails with "not found". Quote it. Suggested command: <span class="c">ax "https://jobs.example.com/listing/1234" --md --budget 1500; echo '====='; ax "https://jobs.example.com/listing/5678" --md</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet nodeVersion()}<span class="blk"><span class="g">●</span> Bash(<span class="c">npm run build</span>)</span><span class="ind"><span class="d">└</span> <span class="y">nanaoki guard:</span> [node_version] The repo pins Node 22 (.nvmrc) but node on PATH is 26. Switch first. Suggested command: <span class="c">nvm use 22 &amp;&amp; &lt;command&gt;</span></span><span class="blk gap"><span class="d">If it runs anyway:</span></span><span class="ind"><span class="d">└</span> <span class="r">Toolchain check failed: expected Node 22.x for this project.</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] The project pins a Node major version. Use the pinned version (fnm/nvm use) before install/build. Fix seen: <span class="c">APP_NODE=~/.nvm/versions/node/v22.22.0/bin/node npm …</span> <span class="d">(project-d, 2026-09)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet missingScript()}<span class="blk"><span class="g">●</span> Bash(<span class="c">bun run gen</span>)</span><span class="ind"><span class="d">└</span> <span class="r">error: Script not found "gen"</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] The package.json script does not exist in this workspace. Read package.json scripts first. Fix seen: <span class="c">bun run format:check</span> <span class="d">(project-e, 2026-04)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet sveltekitResolve()}<span class="blk"><span class="g">●</span> Bash(<span class="c">bun run lint</span>)</span><span class="ind"><span class="d">└</span> <span class="r">24:25  error  Unexpected href link without resolve()  svelte/no-navigation-without-resolve</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] SvelteKit lint: wrap internal hrefs/goto with resolve() from $app/paths (svelte/no-navigation-without-resolve). Fix seen: <span class="c">bunx eslint &lt;5 files&gt;</span> <span class="d">(project-g, 2026-05)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet gitNoCommits()}<span class="blk"><span class="g">●</span> Bash(<span class="c">git status --short --branch &amp;&amp; git diff --stat &amp;&amp; git log -1 --oneline</span>)</span><span class="ind"><span class="d">└</span> <span class="r">fatal: your current branch 'main' does not have any commits yet</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] Error `fatal: your current branch &lt;q&gt; does not have any commits yet` after `git status --short &amp;&amp; git log --oneline -8 &amp;&amp; if test -f web/package.json; then sed -n '1,240p' web/package.json; fi` was fixed by: <span class="c">git status --short; git diff --stat; git diff -- &lt;4 files&gt; …</span> <span class="d">(project-i, 2026-08)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet notJson()}<span class="blk"><span class="g">●</span> Bash(<span class="c">python3 -I -c 'import json,glob; [json.load(open(f)) for f in sorted(glob.glob("data/q*.json"))]'</span>)</span><span class="ind"><span class="d">└</span> <span class="r">json.decoder.JSONDecodeError: Extra data: line 2 column 1 (char 2817)</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] CLI output was not JSON (banner, warning, or error text). Check the exit code and strip non-JSON lines before parsing. Fix seen: <span class="c">seo-cli keywords "…" | jq -r …</span> <span class="d">(project-k, 2026-04)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet dontChangeYet()}<span class="blk"><span class="d">&gt;</span> <span class="h">how could this be fixed? just asking, don't make any changes yet</span></span><span class="ind"><span class="d">└</span> Lessons from your earlier sessions (nanaoki):</span><span class="ind">{"  [L1] don't make any changes yet"}</span><span class="ind">{"  [L2] but don't make any changes yet"}</span>{/snippet}

<!-- prettier-ignore -->
{#snippet wrongLesson()}<span class="blk"><span class="g">●</span> Bash(<span class="c">rg -n "deploy" apps/cms/CLAUDE.md apps/web/CLAUDE.md</span>)</span><span class="ind"><span class="d">└</span> <span class="r">rg: apps/cms/CLAUDE.md: No such file or directory (os error 2)</span></span><span class="ind"><span class="d">└</span> This error matches lessons from your earlier sessions (nanaoki): [L1] A CLI was not installed or has another name. Check with 'command -v' once, then use the right name. Fix seen: <span class="c">rg -n "…" …</span> <span class="d">(project-n, 2026-02)</span></span>{/snippet}

<!-- prettier-ignore -->
{#snippet codeComments()}<span class="blk"><span class="d">&gt;</span> <span class="h">add the helper for the share card and comment the code</span></span><span class="ind"><span class="d">└ (nanaoki printed nothing)</span></span>{/snippet}
