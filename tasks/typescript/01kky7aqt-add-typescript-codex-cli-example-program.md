---
title: "Add TypeScript Codex CLI example program"
id: "01kky7aqt"
status: completed
priority: medium
type: feature
tags: ["typescript", "codex", "example"]
dependencies: ["01kkx7v9v"]
created: "2026-03-17"
completed_at: 2026-04-24
---

# Add TypeScript Codex CLI example program

## Objective

Add a working example program that demonstrates how to use the TypeScript Codex CLI runner. The example should show both `run` and `runStream` usage with async/await and async iterators.

## Tasks

- [x] Create `examples/ts/codex/main.ts` with a working example
- [x] Include `run` and `runStream` usage with async/await and async iterators
- [x] Add a `package.json` and `tsconfig.json` for the example

## Acceptance Criteria

- Example compiles and runs against a local Codex CLI installation
- Example demonstrates both synchronous and streaming usage
- Example is self-contained with its own `package.json` and `tsconfig.json`
