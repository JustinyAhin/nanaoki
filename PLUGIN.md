# nanaoki plugin v0

The plugin turns your own Claude Code and Codex history into lessons, and brings them back at the right moment. Everything runs on your machine. It uses no LLM and no network.

It has five parts:

1. **Lessons store.** `nanaoki lessons build` reads the event store and builds the lessons: an error cause or exact error signature that a later command in the same session fixed, and an instruction you typed in 2+ sessions. `nanaoki lessons export` writes one markdown file per confirmed lesson.
2. **Hooks.** `nanaoki hook EVENT` reads the hook JSON on stdin and prints the reply for Claude Code or Codex:
   - before a shell command: small guards that catch known-bad commands and suggest the fixed command;
   - after a failed tool call: up to 3 lessons that match the error;
   - at the first prompt of a session: up to 2 rules that match the prompt and this repo (strict match).
3. **Session search.** `nanaoki search "QUERY"` is full-text search over all prompts, assistant messages and tool results, with citations. The `past-sessions` skill tells agents when and how to use it.
4. **Rule suggestions.** `nanaoki rules suggest` proposes lines for `~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md` or a project `AGENTS.md`, as a diff. It never writes the file.
5. **Stats and notes.** The hooks log each lesson shown (`shown.jsonl`), each guard that fired (`guards.jsonl`, with the redacted command and the suggested command) and each hook call with its duration (`calls.jsonl`). `nanaoki stats --since 7d` reads these logs and the event store: whether the same failure came back after a lesson, whether the next command used a guard's fix, noise candidates, repeats that got no lesson, hook health, and a rough before/after of the repeat-after-fix rate. `nanaoki note "text" --noise|--useful|--wrong|--missing [--lesson ID|--guard ID]` saves your feedback (`notes.jsonl`); the `nanaoki-feedback` skill lets you say "note that this lesson was noise" to the agent.


## Install

1. Install the binary (see README) and check it: `nanaoki version`.
2. Read your history and build the lessons (about 30 s the first time):

   ```sh
   nanaoki ingest && nanaoki lessons build
   ```

   After that, the SessionStart hook keeps them fresh: when the lesson index is older than a day, it starts `nanaoki refresh` (ingest + lessons build) in the background and returns at once. The log is `~/.local/share/nanaoki/refresh.log`. `NANAOKI_AUTO_REFRESH=0` turns it off.

### Claude Code

```sh
nanaoki setup claude
```

This runs `claude plugin marketplace add JustinyAhin/nanaoki` and `claude plugin install nanaoki@nanaoki --scope user`. Start a new session to load the hooks. The marketplace file is `.claude-plugin/marketplace.json` at the repo root; the plugin is `plugin/claude-code/` (manifest, `hooks/hooks.json`, and the `past-sessions` and `nanaoki-feedback` skills).

From a clone: `nanaoki setup claude --from /path/to/nanaoki`. For one session only: `claude --plugin-dir /path/to/nanaoki/plugin/claude-code`.

Hooks used: `SessionStart`, `UserPromptSubmit`, `PreToolUse` (matcher `Bash`), `PostToolUseFailure` (all tools).

### Codex

```sh
nanaoki setup codex
```

This adds the nanaoki hooks to `~/.codex/hooks.json` (your other hooks stay; the old file goes to `hooks.json.bak`) and copies the skills to `~/.codex/skills/`. Run it again after an upgrade. Then open `/hooks` in Codex and trust the new hooks: Codex skips hooks it has not reviewed. Codex hooks are on by default (`[features] hooks = false` in `config.toml` turns them off).

Hooks used: `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`. Codex has no failure event, so the `PostToolUse` handler finds a failure from the tool response (a non-zero `exit_code` field, or an "Exit code: N" / "Process exited with code N" line).

### Optional: lesson files

```sh
nanaoki lessons export              # global lessons -> ~/.local/share/nanaoki/lessons/
cd ~/work/projects/my-app && nanaoki lessons export --project   # this project's lessons -> ./.lessons/
```

## Uninstall

```sh
nanaoki setup claude --uninstall
nanaoki setup codex --uninstall
rm -rf ~/.local/share/nanaoki      # all data: event store, indexes, logs
```

Remove `.lessons/` folders in repos by hand if you exported them.

## What each hook does

| Event | Claude Code | Codex | Reply |
|---|---|---|---|
| Session start | `SessionStart` | `SessionStart` | Nothing. It only marks the session. The strict rule needs the first message, so it runs at the first prompt. |
| First prompt | `UserPromptSubmit` | `UserPromptSubmit` | Up to 2 rules, only when all hold: the rule's words overlap the prompt (idf overlap >= 0.25), the rule comes from this repo or a repo with the same stack (manifest and framework files; Node alone does not count), and you typed it in 2+ sessions. Later prompts get nothing. |
| Before a shell command | `PreToolUse` (Bash) | `PreToolUse` | The guards. A warning goes to the agent as context with the fixed command; the normal permission flow continues. A block (`permissionDecision: deny`) only for guards marked safe. |
| After a failure | `PostToolUseFailure` | `PostToolUse` (failure detected) | Up to 3 lessons with the same exact error signature or the same cause, ranked by project, stack, command and number of sessions. |

A lesson shows at most once per session. Each shown lesson goes to `~/.local/share/nanaoki/shown.jsonl` (the `shown` and `repeats_after` scores of the export).

### Guards

Each guard is deterministic, small, and has a unit test (`internal/guards`). A guard never changes the command. It warns and suggests the fixed command; it blocks only when the command fails for certain.

| Guard | Lesson (cause) | Fires when | Decision |
|---|---|---|---|
| `zsh_eq` | `zsh_eq_quote` | Your shell is zsh and a word starts with an unquoted `=` that is not a command name (`echo =====`). Skipped inside `[[ ]]`. | **block** (safe: zsh stops with "not found") |
| `zsh_glob` | `zsh_glob_nomatch` | Your shell is zsh and an unquoted word has `*`, `?` or `[..]` that matches no file in the cwd (URLs with `?`, `[slug]` routes, `--include=*.md`). In a chain the other commands still run, so the agent sees exit 0 and misses results. | block |
| `format_before_commit` | `prettier_check` | `git commit` in a repo whose `package.json` has a `format` script, and no formatter ran earlier in the session. | warn |
| `node_version` | `node_version` | npm, npx, pnpm, yarn, node or corepack in a repo that pins a Node major (`.nvmrc`, `.node-version`, `.tool-versions`, `engines.node`) that differs from the node on PATH. Off by default; `NANAOKI_NODE_GUARD=1` turns it on. The hook sees the PATH of the agent process, not the PATH of the shell that runs the command, so it can warn on a correct Node. Quiet when node on PATH is a version-manager shim (vite-plus, volta, mise). | warn |
| `ssh_user` | `ssh_vps_user` | `ssh HOST`, `scp HOST:`, `rsync HOST:` with no user, when an earlier session fixed `ssh HOST` with `ssh USER@HOST`. | warn |

The zsh guards run only when the command runs in zsh: the shell of a Codex argv command (`["bash", "-lc", ...]` is bash), else `$SHELL` (or `NANAOKI_SHELL`).

### Speed and failure

- One hook call takes about 15 ms at p95 on an M-series MacBook (target: under 50 ms). The lesson index is one gob file (`~/.local/share/nanaoki/lessons.gob`, about 3 MB). The node version is cached by PATH.
- Fail open: on any error (no index, bad JSON, a panic), the hook prints nothing and exits 0, so the agent goes on as if there were no hook. It does not print an explicit "allow", because that would skip your own permission prompt. The error goes to `~/.local/share/nanaoki/hook.log`.

## Lesson files

`nanaoki lessons export` writes one file per confirmed lesson. Example (synthetic):

```markdown
---
id: "cause:1e0211f922"
kind: cause
when:
  event: tool_failure
  cause: zsh_glob_nomatch
  commands: ["ls", "rg"]
applies_to:
  stack: ["node", "sveltekit"]
  project: ["alpha", "beta"]
  files: []
status: confirmed
enforce: hook
guard: zsh_glob
evidence:
  - project: "alpha"
    date: 2026-09-01
    session: "codex:0000aaaa-..."
    pointer: "~/.codex/sessions/2026/09/01/rollout-....jsonl:12"
    error: "(eval):N: no matches found: <path>"
    fix: "ls 'src/routes/[slug]'"
score:
  shown: 3
  repeats_after: 0
  last_seen: 2026-11-02
  sessions: 2
  episodes: 2
---

zsh fails on unmatched globs. Quote paths with [ ] * ? (SvelteKit [slug] routes, --include=*.md, URLs with ?). Fix seen (alpha): `ls 'src/routes/[slug]'`.
```

- `status`: `candidate` (seen in 1 session; not exported), `confirmed` (2+ sessions), `retired` (set by you: edit the file; the next export keeps it as it is).
- `enforce`: `hook` when a guard checks the cause before the command runs, else `reminder`.
- `score`: `shown` = times a hook showed it; `repeats_after` = episodes with this key that started after the first time it was shown; `last_seen` = the last episode.
- Global lessons (an env or stack cause, or a signature or rule seen in 2+ projects) go to `~/.local/share/nanaoki/lessons/`. `--project` writes only the lessons of the current project that are not global, to `./.lessons/`.

## Session search

```sh
nanaoki search "stripe billing portal"                  # all projects
nanaoki search --project . --since 2026-10-01 "deploy hook"
nanaoki search --agent codex --format json "sentry recap"
nanaoki search --raw '"weekly report" AND client'
```

Each result has the agent, time, project, session id, a snippet (64 tokens) and the source file line. At most 2 results per session. The index is `~/.local/share/nanaoki/search.db` (FTS5, BM25). Each search first syncs it with the event store (under 1 s when nothing changed; about 15 s the first time). Use `--no-sync` to skip that.

## Rule suggestions

```sh
nanaoki rules suggest             # global: diff for ~/.claude/CLAUDE.md and ~/.codex/AGENTS.md
nanaoki rules suggest --project   # this project: diff for ./AGENTS.md
```

Sources: the preference rules of the report ("no AI attribution", "commit or push only when told", ...) that you typed in 2+ sessions, and instruction sentences you typed in 2+ sessions. A rule seen in one project goes to that project's file; else it is global. Rules the file already states are left out. `--write` does not exist in v0: copy the lines you want.

Known limit: a pasted log line that holds "remember", "make sure" or "never" can count as a typed instruction. Review the diff.

## Privacy

- Local only. No network calls. No LLM. The hooks read the lesson index and write only under `~/.local/share/nanaoki/` (hook state, `shown.jsonl`, `hook.log`, `node-version.json`). `nanaoki setup` is the only command that writes outside that folder.
- Read-only on `~/.claude` and `~/.codex`. `rules suggest` reads `CLAUDE.md` / `AGENTS.md` to compare and never writes them. Nothing installs itself: you run `nanaoki setup`.
- What lessons hold: the cause text, the normalized error signature (paths, numbers, quoted strings replaced) and the fix command, cut to 160 characters and redacted (secrets, tokens, emails, phone numbers, IP addresses, handles, your home folder, your login and git user name, and the names in `~/.local/share/nanaoki/private_names.txt`). Evidence has project names, dates, session ids and file pointers, not prompts or outputs.
- The lesson index and the search index hold your history on your disk, like the event store. The search index keeps the full text of messages and tool results so it can show snippets. Snippets can hold secrets from old sessions; the skill tells agents not to copy them into commits or files.
- `.lessons/` files are meant for git. Read them before you commit them: they name projects and contain fix commands.
