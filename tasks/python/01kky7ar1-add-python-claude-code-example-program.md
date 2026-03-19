---
title: "Add Python Claude Code example program"
id: "01kky7ar1"
status: completed
priority: medium
type: feature
tags: ["python", "claudecode", "example"]
dependencies: ["01kkx3f67"]
created: "2026-03-17"
---

# Add Python Claude Code example program

## Objective

Add a working example program that demonstrates how to use the Python Claude Code runner. The example should show both `run` and `run_stream` usage with asyncio.

## Tasks

- [x] Create `examples/python/claudecode/main.py` with a working example
- [x] Include `run` and `run_stream` usage with asyncio
- [x] Add a `pyproject.toml` for the example

## Acceptance Criteria

- Example runs against a local Claude Code CLI installation
- Example demonstrates both synchronous and streaming usage
- Example is self-contained with its own `pyproject.toml`
