# nanaoki CLI

Go CLI and agent plugin. It reads Claude Code and Codex history on the user's machine, builds lessons, and serves them through hooks, search and a report.

## Rules for the product

- Local only. The CLI makes no network calls and uses no LLM.
- Read `~/.claude` and `~/.codex` (or `$CLAUDE_CONFIG_DIR`, `$CODEX_HOME`) only. Write only to `~/.local/share/nanaoki/`.
- Hooks fail open. On any error, print nothing, exit 0, and write to `hook.log`. Never return an explicit "allow".
- Hooks must stay fast: under 50 ms at p95. Read the cached index. Never rebuild it in a hook.
- Guards warn and suggest the fixed command by default. Block only when the guard is safe. Never change a command silently.
- Stay quiet on probes (`rules.IsProbe`). A noisy hook gets uninstalled.
- Every number in the report must link back to evidence (file and line) in `--format json`.
- `rules suggest` prints a diff. It never writes the user's files.

## Code

- Pure Go. No cgo: `modernc.org/sqlite`, `github.com/klauspost/compress/zstd`. Cross-builds must work with `CGO_ENABLED=0`.
- Layout: `cmd/nanaoki` (commands), `internal/ingest` (parsers, store), `internal/rules` (regex rules, causes, redaction, probes), `internal/analyze` (report), `internal/lessons` (lesson store, index, export), `internal/hook`, `internal/guards`, `internal/search`, `internal/rulesuggest`.
- Stream log files. Never load a whole session file into memory.
- Keep regex rules general. Do not add rules for one person's tools, hosts or projects.
- Run `gofmt`, `go vet ./...` and `go test ./...` before each commit.

## Tests

- Use small synthetic fixtures in `testdata/`. Never copy real session content into the repo.
- Add a test for each new parser format, guard, cause and filter.
- A change to counting logic must keep the documented numbers reachable, or explain the change.

## Releases

- `nanaoki version` prints the version that GoReleaser sets with `-X main.version`.
- A `v*` tag runs `.github/workflows/release.yml`. It builds macOS, Linux and Windows binaries, checksums and the GitHub Release.
- `install.sh` checks the SHA-256 checksum before it installs.
