# Blog posts

Each post is one Markdoc file: `src/content/blog/<slug>.mdoc`. The URL is `/blog/<slug>/`. Use a kebab-case slug.

## Frontmatter

```yaml
---
title: The fix was in the other repo # required
description: One or two sentences. # required, used in the index, SEO and the feed
date: 2026-11-16 # required, YYYY-MM-DD
draft: true # optional, default false
og: /blog/<slug>/card.png # optional, a 1200x630 image in static/
tags: [coding-agents, codex] # optional
---
```

`draft: true` posts render in `bun run dev` only. A production build replaces their text with `null` (see the `blog-posts` plugin in `vite.config.ts`), so they are not in any page, the index, the sitemap, the feed or the worker. In dev, a post shows its validation errors at the top. In a production build, any error fails the build.

## Markdown

Paragraphs, `##` and `###` headings (they make the contents list), lists, links, `inline code`, `> quotes`, tables and fenced code. Code fences are highlighted at build time for `sh`, `ts`, `js`, `json`, `go`, `yaml`, `toml` and `diff`. Other languages render as plain text. Do not use `#`: the title comes from the frontmatter.

## Tags

`width` is `text` (default), `wide` or `full` on `figure`, `video`, `terminal`, `stats`, `chart`, `compare` and `art`.

````md
{% callout type="note" title="Optional title" %}Text.{% /callout %}
{% callout type="warn" %}Text.{% /callout %}
{% callout type="quote" cite="Who said it" %}Text.{% /callout %}

A sentence.{% sidenote %}Margin note on wide screens, tap to open on phones.{% /sidenote %} More text.

{% figure src="/blog/slug/shot.png" alt="Required with src" caption="Optional" width="wide" /%}
{% figure caption="A figure can also wrap other tags." %}...{% /figure %}
{% video src="/blog/slug/film.mp4" label="Required" poster="/blog/slug/poster.jpg" caption="Optional" width="wide" /%}

{% terminal title="after the failure" %}
```
$ a command (cyan)
! an error line (red, the "!" is hidden)
└ a tool result line
● Bash(a tool call)
> a prompt line
# a comment line (dim)
plain output
```
{% /terminal %}

{% stats %}
{% stat value="2,209" label="repeats" source="Short citation, shown on hover and as a footnote" /%}
{% stat value=15 label="hook p95 in ms" /%}
{% /stats %}
{% stat value="40%" label="A stat on its own prints its source under it" source="..." /%}

{% chart type="bar" unit=" sessions" highlight="Second" caption="..." source="..."
  data=[{label: "First", value: 193}, {label: "Second", value: 181}] /%}
{% chart type="bar" series=["Without", "With"] data=[{label: "Shell", values: [40, 12]}] /%}
{% chart type="line" unit="%" series=["A", "B"] data=[{label: "Mar", values: [41, 40]}] /%}
{% chart type="line" title="One series" data=[{label: "W1", value: 3}] /%}
{% chart type="stacked" series=["Guard", "Lesson", "Missed"] data=[{label: "Shell", values: [3, 2, 1]}] /%}

{% compare label="Without and with nanaoki" %}
{% pane label="Without" %}...{% /pane %}
{% pane label="With" %}...{% /pane %}
{% /compare %}

{% steps %}
1. First step.
2. Second step.
{% /steps %}

{% art name="seven-falls" height=360 seed=3 caption="Optional" width="wide" /%}

{% reveal %}Fades in when it scrolls into view.{% /reveal %}
````

Charts: up to 4 series, one axis. A single series is ink, and `highlight` names the one bar to show in the accent. `unit` shows on the first bar value, on the line end labels and in tooltips. Each chart has a "Show the numbers" table.

## Post tags

Tags made for one post. Their data checks are in `src/lib/server/blog/post-tags.ts`.

```md
{% hook-demo cases=["zsh-equals", "missing-script"] /%}
{% episodes total=5290 repeats=2274 crossProject=1578 sameProject=696 /%}
{% eval-explorer setups=["A", "B"] tokens=[0, 36] slices=[{id: "all", label: "All", judge: "claude", n: 10, values: [0, 50], ci: [null, [40, 60]], extra: {label: "B + search", value: 60}, caption: "Text."}] /%}
{% install /%}
{% proverb /%}
```

`hook-demo` reads the cases from `CASES` in `src/lib/cases.ts`.

## Add a component

1. Write the Svelte component in `src/lib/components/blog/`. Respect `prefers-reduced-motion`. Make it work without JavaScript where possible. For scroll effects, use `whenSeen` from `#lib/blog/in-view.ts`: hide content only in the `armed` state.
2. Add the tag to `src/lib/server/blog/schema.ts` (attributes, `render` name, `validate` for data that can be wrong).
3. Add the render name to `src/lib/components/blog/registry.ts`.
4. Use it in `kitchen-sink.mdoc` and add an example here.

## Add an art piece

1. Write `src/lib/blog/art/<name>.ts`. Export an `ArtPiece`: `label` (alt text), `still` (the time of the reduced-motion frame) and `draw({ ctx, width, height, time, seed, colors })`. Use `rng(seed)` from `random.ts` for every random choice, and create it again in each frame, so a seed always draws the same piece.
2. Add it to `ART` in `src/lib/blog/art/index.ts`. The schema reads the names from there.

The host (`art.svelte`) handles the canvas size, device pixel ratio, the colour tokens (`--ink`, `--gold-1..3`), pausing off-screen and reduced motion.
