---
title: "Implement Claude Code core runner (process spawning and result collection)"
id: "01kkx4368"
status: completed
priority: high
type: feature
tags: ["go"]
created: "2026-03-17"
dependencies: ["01kkx4367"]
parent: 01kkx4yva
---

# Implement core runner (process spawning and result collection)

## Objective

Implement the core `Run` function that spawns the `claude` CLI as a subprocess, collects stream-json output, and returns a typed result. This is the primary entry point of the library. Reference pons `claude.go` for the basic run-and-collect pattern, and modpol `invoke.go` for error handling and `CommandBuilder` testability.

## Tasks

- [x] Implement `Run(ctx context.Context, prompt string, opts ...Option) (*Result, error)` using functional options pattern
- [x] Build CLI arguments from options (`--print`, `--output-format stream-json`, `--model`, etc.)
- [x] Spawn process with `exec.CommandContext`, pipe stdout, capture stderr
- [x] Scan stdout line by line, parse each as a stream-json message
- [x] Extract the final `result` message for the return value
- [x] Handle context cancellation (deadline → `ErrTimeout`)
- [x] Handle non-zero exit codes with stderr in error message
- [x] Support `CommandBuilder` injection for testability (like modpol)
- [x] Default binary to `"claude"`, allow override via option

## Acceptance Criteria

- `Run` returns a `*Result` with text, usage, cost, duration, session ID
- Context cancellation kills the subprocess and returns `ErrTimeout`
- Non-zero exit returns `ErrNonZeroExit` with stderr
- Missing binary returns `ErrNotFound`
- CLI arguments correctly reflect all provided options
- Tests pass using a mock command builder (no real `claude` binary needed)
