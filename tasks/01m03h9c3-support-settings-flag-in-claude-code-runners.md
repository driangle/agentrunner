---
id: "01m03h9c3"
title: "Support --settings flag in Claude Code runners"
status: completed
priority: medium
effort: medium
dependencies: []
tags: ["claudecode", "config"]
created_at: 2026-08-16
completed_at: 2026-08-16
---

# Support --settings flag in Claude Code runners

## Objective

Expose the Claude Code CLI `--settings` flag through the Claude Code runners so
callers can pass additional settings per run — as a file path or an inline JSON
string — without staging a whole `CLAUDE_CONFIG_DIR`.

CLI reference (`claude --help`): `--settings <file-or-json>` — *"Path to a settings
JSON file or a JSON string to load additional settings from."*

## Motivation

Downstream (skival) currently can only supply richer config (permission blocks,
hooks, env) via `CLAUDE_CONFIG_DIR` gymnastics. `--settings` lets a caller pass
generated config inline per-run, which is a better fit for per-sample/per-variant
configuration. See skival spec `docs/specs/tool-deny-enforcement.md` (task `01m03awyn`).

## Design notes

- Go: add `Settings string` to `ClaudeOptions` and `WithSettings(fileOrJSON string)`
  (`go/claudecode/options.go`); in `buildArgs` (`go/claudecode/runner.go`) append
  `--settings`, `co.Settings` when non-empty. The value may be a path or a raw JSON
  string — pass it through verbatim; the CLI disambiguates.
- Mirror in TypeScript (`ts/src/claudecode`) and Python (`python/src/agentrunner/claudecode`).
- Note the CLI caveat: in `-p`/non-interactive mode, settings files that fail
  validation are silently ignored (no error). Consider documenting this so callers do
  not assume a malformed value will surface an error.

## Tasks

- [x] Go: add `Settings` option + `WithSettings(...)` and emit `--settings <value>`
- [x] TypeScript: add the equivalent option and arg building
- [x] Python: add the equivalent option and arg building
- [x] Unit tests in each language's arg builder (path value, inline-JSON value, unset)
- [x] Document `--settings` (and the silent-validation caveat) in `CLAUDE.md`

## Acceptance Criteria

- Each runner accepts a settings option and, when set, passes `--settings <value>` verbatim
- Both a file path and an inline JSON string are accepted and passed through unchanged
- New unit tests cover the flag; existing tests continue to pass
- `CLAUDE.md` lists `--settings` as a supported flag so future language libraries include it
</content>
