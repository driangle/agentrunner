---
title: "Implement a supported CLI version range (upper bound) or align docs"
id: "01kz3bppr"
status: pending
priority: medium
type: feature
tags: ["foundation", "claudecode"]
created: "2026-08-03"
phase: critical-feedback
---

# Implement a supported CLI version range (upper bound) or align docs

## Objective

`CLAUDE.md` states the purpose of the compatibility contract is to declare a supported version *range* and to "detect incompatible versions" — implying both a floor and a ceiling. Every implementation and README only enforces a minimum (`>= 1.0.12`). A breaking change in a future CLI release above the floor passes undetected, so the stated contract over-promises.

Resolve the mismatch: either add an upper bound to the check, or narrow the documented promise to "minimum version" everywhere.

## Tasks

- [ ] Decide: enforce a max supported version (range) vs. document a minimum only.
- [ ] If range: add a `MAX_VERSION` (or supported-range) constant per library and extend each runtime check to warn or error above it; add tests.
- [ ] If minimum-only: update `CLAUDE.md`'s "CLI version compatibility" wording to say "minimum supported version" rather than "range".
- [ ] Ensure Go, TypeScript, Python, and the docs/READMEs all reflect the chosen approach consistently.

## Acceptance Criteria

- `CLAUDE.md`, the code, and the READMEs describe the same compatibility semantics.
- If a range is enforced, a version above the ceiling is detected with a clear message and is covered by a test.
- `make check` passes.
