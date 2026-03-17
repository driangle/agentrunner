---
title: "Support --include-partial-messages flag in Claude Code runners"
id: "01kky8ved"
status: pending
priority: high
type: feature
tags: ["claudecode", "streaming"]
created: "2026-03-17"
---

# Support --include-partial-messages flag in Claude Code runners

## Objective

Add support for the `--include-partial-messages` CLI flag across all existing Claude Code runner implementations (Go, TypeScript) and document it as a required flag for any future language libraries. This flag enables streaming of partial/incremental messages from the Claude Code CLI, which is needed for real-time text output in streaming mode.

## Tasks

- [ ] Add `includePartialMessages` option to Go `ClaudeRunOptions` and pass `--include-partial-messages` in argument building
- [ ] Add `includePartialMessages` option to TypeScript `ClaudeRunOptions` and pass `--include-partial-messages` in argument building
- [ ] Add unit tests for the new flag in both Go and TypeScript arg builders
- [ ] Update Go example to use `--include-partial-messages` in the streaming example
- [ ] Update TypeScript example to use `--include-partial-messages` in the streaming example
- [ ] Document `--include-partial-messages` in `CLAUDE.md` under the Claude Code CLI flags section

## Acceptance Criteria

- Go and TypeScript runners accept an `includePartialMessages` boolean option
- When enabled, `--include-partial-messages` is passed to the Claude Code CLI subprocess
- Existing unit tests continue to pass; new tests cover the flag
- Both example programs demonstrate streaming with partial messages
- `CLAUDE.md` lists `--include-partial-messages` as a supported CLI flag so future libraries include it
