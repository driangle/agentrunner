---
title: "Implement Go Gemini CLI runner"
id: "01kkx4rk2"
status: completed
priority: medium
type: feature
tags: ["go", "gemini"]
created: "2026-03-17"
parent: 01kkx3f5s
dependencies: ["01kkx7v98"]
completed_at: 2026-04-24
---

# Implement Go Gemini CLI runner

## Objective

Implement a Gemini CLI runner in Go that satisfies the common Runner interface. Research the Gemini CLI's flags and output format, then implement process spawning, output parsing, and streaming.

## Tasks

- [x] Research Gemini CLI flags and output format (non-interactive/programmatic mode)
- [x] Define Gemini-specific option extensions
- [x] Define Gemini-specific message/output types
- [x] Implement `Run` and `RunStream` for the Gemini CLI
- [x] Add tests with mock command builder

## Acceptance Criteria

- Gemini runner implements the common Runner interface
- Callers can swap between Claude and Gemini runners without code changes
- Tests pass without requiring the real `gemini` binary
