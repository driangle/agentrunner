---
title: "Add --include-partial-messages example to Claude Code examples"
id: "01kky8wdz"
status: pending
priority: medium
type: feature
tags: ["claudecode", "example", "streaming"]
dependencies: ["01kky8ved"]
created: "2026-03-17"
---

# Add --include-partial-messages example to Claude Code examples

## Objective

Update all existing Claude Code example programs (Go, TypeScript) to demonstrate usage of `--include-partial-messages` for real-time incremental streaming. Also update CLAUDE.md so that any future language example tasks include this flag by default.

## Tasks

- [ ] Add a streaming example with `includePartialMessages: true` to `examples/ts/claudecode/main.ts`
- [ ] Add a streaming example with `IncludePartialMessages: true` to `examples/go/claudecode/main.go`
- [ ] Update `CLAUDE.md` example task descriptions to note that examples should demonstrate `--include-partial-messages`

## Acceptance Criteria

- Go and TypeScript examples each have a working streaming example that uses `includePartialMessages`
- Examples compile and type-check cleanly
- `CLAUDE.md` documents that future example tasks should include `--include-partial-messages` usage
- `make check` passes
