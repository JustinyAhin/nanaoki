---
name: past-sessions
description: Search the user's past Claude Code and Codex sessions on this machine with the local `nanaoki search` CLI. Use when the user asks to check, find, or recall something from earlier sessions or conversations ("check my past sessions", "what did we decide about X", "we already fixed this", "find the conversation where", "continue from where we left off", "resume the X work"), or when a task clearly continues earlier work and the current context does not hold it.
---

# Past sessions

`nanaoki search` runs full-text search (SQLite FTS5, BM25) over every prompt, assistant message, and tool call with its output in the local Claude Code and Codex history. It is local only and makes no network calls.

## How to search

1. Pick 3 to 6 specific words from the request: project names, feature names, file names, error text, tool names. Leave out filler words.
2. Search the current project first, then all projects if the results are thin:

   ```sh
   nanaoki search --paths --project . "stripe billing portal configuration"
   nanaoki search --paths "stripe billing portal configuration"
   ```

3. Narrow with flags when needed:
   - `--since YYYY-MM-DD`, `--until YYYY-MM-DD` for a time range ("last week", "in October").
   - `--agent claude` or `--agent codex` when the user names the agent.
   - `--raw` for FTS5 syntax: a phrase `"weekly report"`, `AND`, `NOT`, a prefix `deploy*`.
   - `--limit N` (default 8, one result per session with the count of its other hits) and `--format json` for the full fields.
   - The current session is left out. `--include-current` keeps it.
4. Run 2 or 3 searches with other words before you conclude that nothing exists.

## How to answer

- Answer from the results. Cite each fact with its source: agent, date, project, and session id (the first 8 characters are enough), for example "(codex, 2026-11-03, tangledplot, session 01a1a374)".
- Results are snippets. To read more, open the cited file line that `--paths` shows (`~/.codex/...jsonl:LINE` or `~/.claude/projects/...jsonl:LINE`) with a read-only command. Codex files that end in `.zst` need `zstd -dc FILE | sed -n 'LINE p'`.
- If the results do not answer the question, say so. Do not fill gaps with guesses.
- Snippets can hold secrets or personal data from old sessions. Do not copy them into commits, PRs, or files unless the user asks.

## Notes

- The index follows the event store. If a recent session is missing, run `nanaoki ingest` first (a few seconds when little changed).
- Claude Code deletes transcripts after `cleanupPeriodDays` (30 by default). Older Claude sessions may exist only as prompts.
