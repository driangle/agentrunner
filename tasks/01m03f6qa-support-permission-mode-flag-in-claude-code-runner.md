---
id: "01m03f6qa"
title: "Support --permission-mode flag in Claude Code runners"
status: completed
priority: medium
effort: medium
dependencies: []
tags: ["claudecode", "permissions"]
created_at: 2026-08-16
completed_at: 2026-08-16
---

# Support --permission-mode flag in Claude Code runners

## Objective

Expose the Claude Code CLI `--permission-mode` flag through the Claude Code runners.
Today the only permission lever the runner offers is `WithSkipPermissions()` →
`--dangerously-skip-permissions`, which is all-or-nothing (bypass everything). The CLI
supports a graduated set of modes; exposing it lets callers pick a mode that fits a
restricted run instead of reaching for the blunt bypass.

CLI reference (`claude --help`): `--permission-mode <mode>` — choices:
`acceptEdits`, `auto`, `bypassPermissions`, `manual`, `dontAsk`, `plan`.

## Motivation

Downstream (skival) always adds `--dangerously-skip-permissions` for automated runs.
For restricted variants a narrower mode is preferable, but the runner gives no way to
set one. See skival spec `docs/specs/tool-deny-enforcement.md` (task `01m03awyn`).

## Design notes

- Go: add `PermissionMode string` to `ClaudeOptions` and `WithPermissionMode(mode string)`
  (`go/claudecode/options.go`); in `buildArgs` (`go/claudecode/runner.go`) append
  `--permission-mode`, `co.PermissionMode` when non-empty.
- Mirror in TypeScript (`ts/src/claudecode`) and Python (`python/src/agentrunner/claudecode`).
- Keep `WithSkipPermissions()` as-is (sugar for the `bypassPermissions` behavior). If
  both are set, document precedence — recommend `--permission-mode` wins, or validate
  and reject the conflicting combination.
- Consider validating against the known mode set, or pass through as a plain string to
  avoid coupling to a CLI enum that may grow.

## Tasks

- [x] Go: add `PermissionMode` option + `WithPermissionMode(...)` and emit `--permission-mode` (already landed via task `01kpj3xkq`)
- [x] TypeScript: add the equivalent option and arg building
- [x] Python: add the equivalent option and arg building
- [x] Define and test the interaction with `WithSkipPermissions()` — permission mode wins, skip flag omitted
- [x] Unit tests in each language's arg builder (set / unset)
- [x] Document `--permission-mode` in `CLAUDE.md` under the Claude Code CLI flags section

## Acceptance Criteria

- Each runner accepts a permission-mode option and, when set, passes `--permission-mode <mode>`
- Interaction with `--dangerously-skip-permissions` is defined and tested
- New unit tests cover the flag; existing tests continue to pass
- `CLAUDE.md` lists `--permission-mode` as a supported flag so future language libraries include it
</content>
