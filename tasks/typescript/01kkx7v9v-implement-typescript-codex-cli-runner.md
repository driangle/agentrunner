---
id: "01kkx7v9v"
title: "Implement TypeScript Codex CLI runner"
status: completed
priority: medium
phase: typescript
dependencies: ["01kkx7v98", "01kkx4rkd"]
parent: 01kkx3f5h
tags: ["typescript", "codex"]
created: 2026-03-17
completed_at: 2026-04-24
---

# Implement TypeScript Codex CLI runner

## Objective

Implement the Codex CLI runner in TypeScript, following the Go Codex implementation as a reference.

## Tasks

- [x] Define Codex-specific option extensions and message types
- [x] Implement JSON streaming event parser for Codex output
- [x] Implement `run()` and `runStream()` for the Codex CLI
- [x] Add tests with mock subprocess

## Acceptance Criteria

- Codex runner implements the common Runner interface
- Callers can swap between runners without code changes
- Tests pass without requiring the real `codex` binary
