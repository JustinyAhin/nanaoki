---
name: nanaoki-feedback
description: Save the user's feedback about a nanaoki lesson or guard with the local `nanaoki note` CLI. Use when the user comments on a lesson or guard that the nanaoki hooks showed in this session ("note that this lesson was noise", "that guard was wrong", "this tip helped", "nanaoki should have warned me about this", "log feedback for nanaoki"), or asks to see their notes or the plugin stats.
---

# nanaoki feedback

The nanaoki hooks show lessons (`[L1] ...` after "lessons from your earlier sessions (nanaoki)") and guard warnings (`nanaoki guard: [GUARD_ID] ...`). The user can rate them. `nanaoki note` saves each rating on this machine (`~/.local/share/nanaoki/notes.jsonl`), and `nanaoki stats` counts them.

## Save a note

1. Find what the user talks about, from this session's context:
   - a lesson: its number in the newest hook reply (`[L1]`, `[L2]`, ...). `--lesson L1` saves the id of that lesson. For a lesson from an older reply, use its id from `nanaoki stats --since 1d --format json` (`lessons.shows[].lesson`, for example `cause:5663deba89`).
   - a guard: the name in brackets after `nanaoki guard:`, for example `zsh_glob`.
2. Pick one tag:
   - `--useful`: it helped.
   - `--noise`: it showed up but did not help here.
   - `--wrong`: the advice was wrong.
   - `--missing`: a lesson or warning should have shown up and did not.
3. Run it, with the user's words in short form as the text:

   ```sh
   nanaoki note "glob warning on a quoted path, not needed" --noise --guard zsh_glob
   nanaoki note "the resolve() lesson fixed the lint error" --useful --lesson L1
   nanaoki note "should have warned about node 18 before the build" --missing
   ```

   The CLI adds the time, the project and the current session. Leave out `--lesson`/`--guard` when you cannot find the id; the text is enough.
4. Tell the user the note id that the CLI printed. Do not change any other file.

## Read notes and stats

- `nanaoki note --list --since 7d` lists the notes.
- `nanaoki stats --since 7d` shows lessons shown, guards fired, what the agent did next, noise candidates, misses and hook health. Run `nanaoki ingest` first so the store holds the recent sessions.

## Rules

- Do not put secrets, tokens or other people's personal data in a note. The CLI redacts common secrets, but write the note without them.
- Save a note only when the user asks for it or clearly gives feedback on nanaoki. Do not rate lessons on your own.
