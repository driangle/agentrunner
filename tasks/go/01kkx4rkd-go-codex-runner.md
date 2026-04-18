---
title: "Implement Go Codex CLI runner"
id: "01kkx4rkd"
status: completed
priority: medium
type: feature
tags: ["go", "codex"]
created: "2026-03-17"
parent: 01kkx3f5s
dependencies: ["01kkx7v98"]
completed_at: 2026-04-18
---

# Implement Go Codex CLI runner

## Objective

Implement a Codex CLI runner in Go that satisfies the common Runner interface. Research the Codex CLI's flags and output format, then implement process spawning, output parsing, and streaming.

## Tasks

- [x] Research Codex CLI flags and output format (non-interactive/programmatic mode)
- [x] Define Codex-specific option extensions
- [x] Define Codex-specific message/output types
- [x] Implement `Run` and `RunStream` for the Codex CLI
- [x] Add tests with mock command builder

## Acceptance Criteria

- Codex runner implements the common Runner interface
- Callers can swap between Claude, Gemini, and Codex runners without code changes
- Tests pass without requiring the real `codex` binary
