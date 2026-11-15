// Package plugin holds the hook and skill files, embedded so `nanaoki setup codex` can install them.
package plugin

import "embed"

//go:embed codex/hooks.json all:claude-code/skills
var Files embed.FS
