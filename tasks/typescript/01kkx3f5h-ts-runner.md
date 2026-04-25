---
title: "Create TypeScript agentrunner library"
id: "01kkx3f5h"
status: completed
priority: high
type: feature
tags: ["typescript"]
created: "2026-03-17"
phase: typescript
effort: large
completed_at: 2026-04-25
---

# Create TypeScript agentrunner library

## Objective

Build the TypeScript implementation of `agentrunner` in the `ts/` directory. The library provides a typed, idiomatic Node.js/TypeScript interface for invoking AI coding agents through a common Runner interface.

## Children

- `01kkx7v5m` — Implement TypeScript Claude Code CLI runner
- `01kkx7v98` — Implement TypeScript Ollama runner
- `01kkx7v9v` — Implement TypeScript Codex CLI runner
- `01kkx7vaa` — Implement TypeScript Gemini CLI runner

## Acceptance Criteria

- All four runners implement the common `Runner` interface
- Supports both streaming and non-streaming modes
- Exposes well-typed interfaces for all inputs/outputs
- Has passing tests covering core functionality
