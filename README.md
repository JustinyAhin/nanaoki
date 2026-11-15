# nanaoki

Your coding agent fixed this error last month. In another project. Then it forgot.

`nanaoki` reads the Claude Code and Codex history that is already on your machine, finds the mistakes your agents keep repeating, and shows the fix that worked last time, at the moment the same error comes back.

Everything runs locally. No LLM, no network, no account.

## Install

Go 1.26+:

```sh
go install github.com/JustinyAhin/nanaoki/cmd/nanaoki@latest
```

## See your numbers

```sh
nanaoki                 # read your history, then print the report
nanaoki --split auto    # strict number: errors in the last third of your history
                        # whose cause was already fixed in the first two thirds
nanaoki --since 30d     # only the last 30 days (also 2w, 3m, 1y)
```

The report shows:

1. **Coverage**: which agents and sessions it found, and from when.
2. **Repeated errors**: the error causes that come back across sessions, days and projects.
3. **Repeat-after-fix**: errors whose cause an earlier session already fixed, and in which project.
4. **Time lost**: wall time from each repeated error to the next command that worked (an estimate).
5. **Restated rules**: instructions you keep typing in new sessions.
6. **Monthly trend**.
7. **Suggested fixes**: a rule, a hook, or an environment fact for each top item.

All counts come from regex rules over your logs. Treat them as estimates. `--format json` gives the evidence (file and line) for each number.

One tip: Claude Code deletes transcripts after 30 days by default. Set `"cleanupPeriodDays": 3650` in `~/.claude/settings.json` to keep them.

## Stop the repeats (plugin)

```sh
nanaoki lessons build   # build lessons from your history (run it once a day)
claude --plugin-dir "$(go env GOPATH)/pkg/mod/github.com/!justiny!ahin/nanaoki@*/plugin/claude-code"
```

Or clone this repo and use `claude --plugin-dir ./plugin/claude-code`. Codex hooks are in `plugin/codex/`. See [PLUGIN.md](PLUGIN.md).

The plugin:

- warns before a few known-bad shell commands and suggests the fixed command (zsh quoting first);
- after a failed command, shows up to 3 lessons from your history that match the error, with where it was fixed;
- at the first prompt of a session, shows 1-2 rules that match the repo;
- stays quiet on probes (`cat x 2>/dev/null`, `test -f`, a grep with no match).

## Search your past sessions

```sh
nanaoki search "502 prerender" --since 3m
```

Full-text search over all your prompts, agent messages and tool output, with the session and time for each hit. The `past-sessions` skill tells agents to use it when you ask "did we fix this before?".

## Rules you keep typing

```sh
nanaoki rules suggest
```

Prints lines for your `CLAUDE.md` / `AGENTS.md` based on what you restate across sessions. It never writes the file.

## Privacy

- Reads `~/.claude` and `~/.codex` (or `$CLAUDE_CONFIG_DIR`, `$CODEX_HOME`) read-only.
- Writes only to `~/.local/share/nanaoki/`. Delete that folder to remove everything.
- Makes no network calls. Sends nothing anywhere.

## License

MIT
