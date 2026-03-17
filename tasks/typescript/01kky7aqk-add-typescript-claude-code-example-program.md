---
title: "Add TypeScript Claude Code example program"
id: "01kky7aqk"
status: completed
priority: medium
type: feature
tags: ["typescript", "claudecode", "example"]
dependencies: ["01kkx7v5m"]
created: "2026-03-17"
---

# Add TypeScript Claude Code example program

## Objective

Add a working example program that demonstrates how to use the TypeScript Claude Code runner. The example should show both `run` and `runStream` usage with async/await and async iterators.

## Tasks

- [x] Create `examples/ts/claudecode/main.ts` with a working example
- [x] Include `run` and `runStream` usage with async/await and async iterators
- [x] Add a `package.json` and `tsconfig.json` for the example

## Acceptance Criteria

- Example compiles and runs against a local Claude Code CLI installation
- Example demonstrates both synchronous and streaming usage
- Example is self-contained with its own `package.json` and `tsconfig.json`
