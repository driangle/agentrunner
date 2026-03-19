---
title: "Create Python agentrunner library"
id: "01kkx3f67"
status: in-progress
priority: high
type: feature
tags: ["python"]
created: "2026-03-17"
phase: python
effort: large
dependencies: ["01kkx7v5m"]
---

# Create Python CLI runner library

## Objective

Build the Python implementation of `claude-code-cli-runner` in the `python/` directory. The library provides an idiomatic Python interface for invoking and interacting with the Claude Code CLI.

## Tasks

- [ ] Initialize the package (`pyproject.toml`, package structure)
- [ ] Implement core runner that spawns and manages the CLI process
- [ ] Define Python types/dataclasses for CLI inputs and outputs
- [ ] Handle streaming and non-streaming responses
- [ ] Add error handling and result parsing
- [ ] Write tests
- [ ] Add package README

## Acceptance Criteria

- Library can invoke `claude` CLI and return parsed results
- Supports both streaming and non-streaming modes (sync and async)
- Exposes typed interfaces for all inputs/outputs
- Has passing tests covering core functionality
