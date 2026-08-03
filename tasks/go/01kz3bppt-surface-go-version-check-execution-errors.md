---
title: "Go checkVersion silently swallows --version execution errors"
id: "01kz3bppt"
status: pending
priority: low
type: chore
tags: ["go", "claudecode"]
created: "2026-08-03"
dependencies: ["01kz3bppj"]
---

# Go checkVersion silently swallows --version execution errors

## Description

In `go/claudecode/runner.go`, `checkVersion` runs `claude --version` and, if the command itself fails to execute, returns early and lets the check pass (`runner.go:79`). This makes the Go version gate effectively best-effort, while Python raises a `NotFoundError` on the same condition. The result is inconsistent strictness across languages for the same design principle.

The current behavior is defensible (`Start` re-checks binary existence via `exec.LookPath`), but the divergence should be a deliberate, documented decision rather than an accident. Coordinate the choice with the version-range decision in `01kz3bppr` and the conformance tests in `01kz3bppp`.

## Tasks

- [ ] Decide the intended behavior when `claude --version` fails to execute (pass-through vs. error) and make it consistent with the Python and TypeScript runners.
- [ ] Add a short comment documenting the chosen behavior and why.
- [ ] Add a unit test asserting the chosen behavior (e.g. version probe failure → expected outcome).

## Tasks (checklist)

- [ ] Behavior decided and aligned across languages
- [ ] Comment added
- [ ] Test added
- [ ] `make check-go` passes
