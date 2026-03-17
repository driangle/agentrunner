---
title: "Add Claude Code session management (continue, resume)"
id: "01kkx436b"
status: completed
priority: medium
type: feature
tags: ["go"]
created: "2026-03-17"
dependencies: ["01kkx4368"]
parent: 01kkx4yva
---

# Add session management (continue, resume)

## Objective

Add options for session management — continuing the most recent conversation or resuming a specific session by ID. These map to the `--continue` and `--resume` CLI flags.

## Tasks

- [x] Add `WithContinue()` option that adds `--continue` flag
- [x] Add `WithResume(sessionID string)` option that adds `--resume <id>` flag
- [x] Add `WithSessionID(id string)` option for `--session-id`
- [x] Ensure session ID is captured from the `system/init` message in results
- [x] Add tests verifying correct CLI argument construction

## Acceptance Criteria

- `WithContinue()` produces `--continue` in the CLI args
- `WithResume("abc")` produces `--resume abc` in the CLI args
- Session ID from init message is available in the result
