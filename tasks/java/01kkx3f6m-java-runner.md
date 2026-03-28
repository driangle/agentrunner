---
title: "Create Java agentrunner library"
id: "01kkx3f6m"
status: pending
priority: low
type: feature
tags: ["java"]
created: "2026-03-17"
phase: java
effort: large
dependencies: ["01kkx3f67"]
---

# Create Java CLI runner library

## Objective

Build the Java implementation of `claude-code-cli-runner` in the `java/` directory. The library provides an idiomatic Java interface for invoking and interacting with the Claude Code CLI.

## Tasks

- [ ] Initialize the project (Maven/Gradle build, package structure)
- [ ] Implement core runner that spawns and manages the CLI process
- [ ] Define Java types for CLI inputs and outputs
- [ ] Handle streaming and non-streaming responses
- [ ] Add error handling and result parsing
- [ ] Write tests
- [ ] Add package README

## Acceptance Criteria

- Library can invoke `claude` CLI and return parsed results
- Supports both streaming and non-streaming modes
- Exposes well-typed interfaces for all inputs/outputs
- Has passing tests covering core functionality
