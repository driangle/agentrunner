---
title: "Fix Go Claude Code CLI version parser (falsely rejects 1.0.x)"
id: "01kz3bppj"
status: pending
priority: high
type: bug
tags: ["go", "claudecode"]
created: "2026-08-03"
phase: critical-feedback
---

# Fix Go Claude Code CLI version parser (falsely rejects 1.0.x)

## Steps to Reproduce

1. Install Claude Code CLI (`claude --version` prints e.g. `1.0.12 (Claude Code)` — note the parenthetical suffix).
2. Set `MinCLIVersion` to `1.0.12` (the current value) in `go/claudecode/runner.go`.
3. Construct a real `Runner` (no injected command builder) and call `Start`, which runs `checkVersion`.

## Expected Behavior

A CLI reporting version `1.0.12` (the exact minimum) or any `1.0.x` should pass the version check.

## Actual Behavior

`isVersionAtLeast` (`go/claudecode/runner.go:93`) splits the raw `--version` output on `.` and `strconv.Atoi`s each part, discarding the error. For `1.0.12 (Claude Code)` the third part `"12 (Claude Code)"` parses to `0`, yielding `[1,0,0]`, which compares as **below** `[1,0,12]` — so a valid CLI is falsely rejected with `ErrNotFound`.

This is currently masked only because the installed CLI is `2.x` (major version clears the floor before the patch is examined). Bumping `MinCLIVersion` into the `2.x` range, or running against any `1.0.x` CLI, exposes the bug. There are also **zero tests** for `isVersionAtLeast` / `checkVersion`.

## Environment

- OS: any
- Version: `go/claudecode/runner.go` @ current `main`; observed `claude --version` = `2.1.220 (Claude Code)`

## Tasks

- [ ] Extract the semver token with a regex (`(\d+\.\d+\.\d+)`) before comparison — mirror the working Python approach in `python/src/agentrunner/claudecode/version.py`.
- [ ] Return a clear parse error when no version token is found, instead of silently treating it as `0.0.0`.
- [ ] Add table-driven unit tests covering: exact minimum, below minimum, above minimum, real output with `(Claude Code)` suffix, and unparseable output.

## Acceptance Criteria

- `1.0.12 (Claude Code)` passes the `>= 1.0.12` check.
- `1.0.11 (Claude Code)` fails with a clear error.
- Setting `MinCLIVersion` to `2.1.0` and testing `2.0.5 (Claude Code)` correctly fails.
- `go/claudecode` has unit tests for the version comparison logic.
- `make check-go` passes.
