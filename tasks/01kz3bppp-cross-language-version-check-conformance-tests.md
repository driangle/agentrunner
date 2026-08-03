---
title: "Add cross-language version-check conformance tests"
id: "01kz3bppp"
status: pending
priority: medium
type: feature
tags: ["foundation", "testing"]
created: "2026-08-03"
dependencies: ["01kz3bppj", "01kz3bppm"]
---

# Add cross-language version-check conformance tests

## Objective

The version-compatibility principle is implemented three different ways today: Python parses robustly with a regex, Go has a buggy hand-rolled parser with no tests, and TypeScript has no check at all. That divergence is the symptom of tasks built in isolation with no shared conformance guarantee.

Define a small, shared set of version-check cases and assert every language library behaves identically, so "the interface is the same everywhere" is verified rather than asserted.

## Tasks

- [ ] Define a canonical table of cases: exact minimum, one patch below, one above, real `X.Y.Z (Claude Code)` suffixed output, whitespace, and unparseable output — each with the expected pass/fail outcome.
- [ ] Add a test in each library (Go, TypeScript, Python) driven by that same table.
- [ ] Keep the canonical `MIN_VERSION` value consistent across the three libraries (single documented source of truth).
- [ ] Ensure the tests run under each library's existing `make check-<lang>` target.

## Acceptance Criteria

- All three libraries agree on every case in the shared table.
- The suffixed-output case (`1.0.12 (Claude Code)`) passes in all three.
- `make check` runs the conformance tests for every implemented language.
