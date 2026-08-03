---
title: "Add missing TypeScript runtime CLI version check"
id: "01kz3bppm"
status: pending
priority: high
type: bug
tags: ["typescript", "claudecode"]
created: "2026-08-03"
phase: critical-feedback
---

# Add missing TypeScript runtime CLI version check

## Steps to Reproduce

1. `grep -rin version ts/src/` — no version detection or comparison exists.
2. Read `ts/README.md:9` — it advertises "Claude Code CLI >= 1.0.12".

## Expected Behavior

Per the "CLI version compatibility" design principle in `CLAUDE.md`, each runner must, on start, verify the installed CLI version falls within the supported range and return a clear error if not. The TS library documents `>= 1.0.12` and should enforce it — matching Go and Python.

## Actual Behavior

The TypeScript Claude Code runner performs **no runtime version check**. The documented `>= 1.0.12` guarantee is unenforced, so a breaking CLI change produces exactly the confusing failure the principle exists to prevent.

## Environment

- OS: any
- Version: `ts/src/claudecode/` @ current `main`

## Tasks

- [ ] Add a version module (e.g. `ts/src/claudecode/version.ts`) exporting a `MIN_VERSION` constant and a function that runs `claude --version`, extracts the semver token via regex, and compares it.
- [ ] Invoke the check when the runner starts; throw the library's typed `NotFound`-category error when the version is too old or the binary is missing.
- [ ] Keep the `MIN_VERSION` constant grep-able and in sync with `ts/README.md`.
- [ ] Add unit tests (mock the version subprocess) covering pass, fail-too-old, suffixed output, and unparseable output.

## Acceptance Criteria

- Starting the runner against a CLI below `1.0.12` throws a clear, typed error.
- `1.0.12 (Claude Code)`-style output parses correctly.
- The min version is defined as a single exported constant in code.
- `make check-ts` passes.
