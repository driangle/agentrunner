---
title: "Write tests and add Go package README"
id: "01kkx436c"
status: completed
priority: medium
type: chore
tags: ["go"]
created: "2026-03-17"
dependencies: ["01kkx4368", "01kkx4369"]
parent: 01kkx4yva
---

# Write tests and add Go package README

## Description

Ensure comprehensive test coverage and add a package-level README for the Go library.

## Tasks

- [x] Add integration-style tests using `CommandBuilder` with mock subprocess
- [x] Add table-driven tests for CLI argument building from all option combinations
- [x] Add parser tests with a real session fixture (JSONL file from a `claude -p --output-format stream-json` invocation)
- [x] Ensure `go test ./...` passes with no race conditions (`-race`)
- [x] Add `go/README.md` with installation, quick start, and API overview
- [x] Add usage examples in README (basic run, streaming, session resume)
