# nanaoki plugin v0

The plugin turns your own Claude Code and Codex history into lessons, and brings them back at the right moment. Everything runs on your machine. It uses no LLM and no network.

It has four parts:

1. **Lessons store.** `nanaoki lessons build` reads the event store and builds the lessons: an error cause or exact error signature that a later command in the same session fixed, and an instruction you typed in 2+ sessions. `nanaoki lessons export` writes one markdown file per confirmed lesson.
2. **Hooks.** `nanaoki hook EVENT` reads the hook JSON on stdin and prints the reply for Claude Code or Codex:
   - before a shell command: small guards that catch known-bad commands and suggest the fixed command;
   - after a failed tool call: up to 3 lessons that match the error;
   - at the first prompt of a session: up to 2 rules that match the prompt and this repo (strict match).
3. **Session search.** `nanaoki search "QUERY"` is full-text search over all prompts, assistant messages and tool results, with citations. The `past-sessions` skill tells agents when and how to use it.
4. **Rule suggestions.** `nanaoki rules suggest` proposes lines for `~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md` or a project `AGENTS.md`, as a diff. It never writes the file.


## Install

1. Build the binary and put it on your PATH (Go 1.26+):

   ```sh
   cd cli
   go install ./cmd/nanaoki        # or: go build -o ~/.local/bin/nanaoki ./cmd/nanaoki
   ```

2. Read your history and build the lessons (about 30 s the first time):

   ```sh
   nanaoki ingest
   nanaoki lessons build
   ```

   Run both again from time to time (for example once a day) to add new sessions. The hooks read the lesson index that `lessons build` writes; they do not rebuild it.

3. Optional: export the lessons as markdown.

   ```sh
   nanaoki lessons export              # global lessons -> ~/.local/share/nanaoki/lessons/
   cd ~/work/projects/my-app && nanaoki lessons export --project   # this project's lessons -> ./.lessons/
   ```

### Claude Code

The plugin is in `cli/plugin/claude-code/` (manifest, `hooks/hooks.json`, and the `past-sessions` skill). `cli/plugin/` is a local marketplace.

```sh
claude plugin marketplace add /path/to/github.com/JustinyAhin/nanaoki/plugin
claude plugin install nanaoki@nanaoki-local --scope user
```

To try it for one session without installing: `claude --plugin-dir /path/to/github.com/JustinyAhin/nanaoki/plugin/claude-code`.

`skills` in the plugin folder is a symlink to `cli/plugin/skills`. If your Claude Code version does not follow it, replace the link with a copy of that folder.

Alternative without the plugin: copy the `"hooks"` object of `cli/plugin/claude-code/hooks/hooks.json` into `~/.claude/settings.json`, and copy `cli/plugin/skills/past-sessions/` to `~/.claude/skills/`.

Hooks used: `SessionStart`, `UserPromptSubmit`, `PreToolUse` (matcher `Bash`), `PostToolUseFailure` (all tools).

### Codex

Codex hooks are on by default (`[features] hooks = false` in `config.toml` turns them off). Copy `cli/plugin/codex/hooks.json` to `~/.codex/hooks.json` (or merge its `"hooks"` object into an existing file). Then open `/hooks` in Codex and trust the new hooks: Codex skips hooks it has not reviewed.

For the skill, copy `cli/plugin/skills/past-sessions/` to `~/.codex/skills/`.

Hooks used: `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`. Codex has no failure event, so the `PostToolUse` handler finds a failure from the tool response (a non-zero `exit_code` field, or an "Exit code: N" / "Process exited with code N" line).

## Uninstall

```sh
claude plugin uninstall nanaoki@nanaoki-local
claude plugin marketplace remove nanaoki-local
```

Codex: remove the nanaoki entries from `~/.codex/hooks.json` (or delete the file if it holds nothing else) and remove `~/.codex/skills/past-sessions/`.

Data: `rm -rf ~/.local/share/nanaoki` removes the event store, the lesson index, the search index, the hook state and the logs. Remove `.lessons/` folders in repos by hand if you exported them.

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
| `zsh_glob` | `zsh_glob_nomatch` | Your shell is zsh and an unquoted word has `*`, `?` or `[..]` that matches no file in the cwd (URLs with `?`, `[slug]` routes, `--include=*.md`). | warn |
| `format_before_commit` | `prettier_check` | `git commit` in a repo whose `package.json` has a `format` script, and no formatter ran earlier in the session. | warn |
| `node_version` | `node_version` | npm, npx, pnpm, yarn, node or corepack in a repo that pins a Node major (`.nvmrc`, `.node-version`, `.tool-versions`, `engines.node`) that differs from the node on PATH. | warn |
| `ssh_user` | `ssh_vps_user` | `ssh HOST`, `scp HOST:`, `rsync HOST:` with no user, when an earlier session fixed `ssh HOST` with `ssh USER@HOST`. | warn |

The zsh guards run only when `$SHELL` (or `NANAOKI_SHELL`) is zsh.

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

- Local only. No network calls. No LLM. The hooks read the lesson index and write only under `~/.local/share/nanaoki/` (hook state, `shown.jsonl`, `hook.log`, `node-version.json`).
- Read-only on `~/.claude` and `~/.codex`. `rules suggest` reads `CLAUDE.md` / `AGENTS.md` to compare and never writes them. The plugin files are snippets: nothing here installs itself.
- What lessons hold: the cause text, the normalized error signature (paths, numbers, quoted strings replaced) and the fix command, cut to 160 characters and redacted (secrets, tokens, emails, phone numbers, IP addresses, handles, your home folder, your login and git user name, and the names in `~/.local/share/nanaoki/private_names.txt`). Evidence has project names, dates, session ids and file pointers, not prompts or outputs.
- The lesson index and the search index hold your history on your disk, like the event store. The search index keeps the full text of messages and tool results so it can show snippets. Snippets can hold secrets from old sessions; the skill tells agents not to copy them into commits or files.
- `.lessons/` files are meant for git. Read them before you commit them: they name projects and contain fix commands.
