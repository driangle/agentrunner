---
title: "Enforce max lines per file in Python lint"
id: "01m04nnap"
status: pending
priority: low
type: chore
tags: ["lint", "tooling"]
created: "2026-08-16"
phase: python
---

# Enforce max lines per file in Python lint

## Description

The TypeScript library caps files at 200 lines via the ESLint `max-lines` rule
(`ts/eslint.config.js`), enforced on every commit through the pre-commit hook.
Python has no equivalent: `ruff` is configured with `line-length = 100` (max
characters per line) and `select = ["E", "F", "I", "W"]`, none of which limit
file length.

Add a 200-line-per-file limit to the Python lint step so the file-organization
principle in `CLAUDE.md` ("a file should be scannable in a minute or two") is
enforced uniformly across languages.

Ruff has no built-in file-length rule, so this needs either a small check script
invoked from `lint-python`, or a flake8/other linter added alongside ruff. Prefer
the simplest option that keeps `make lint-python` a single entry point.

## Tasks

- [ ] Pick the enforcement mechanism (small script run from `lint-python`, or an
      additional linter with a file-length rule)
- [ ] Set the limit to 200 lines, counting blank lines and comments, to match the
      TypeScript rule
- [ ] Exempt `python/tests/` (TypeScript disables `max-lines` for `test/**/*.ts`)
- [ ] Wire the check into the `lint-python` Makefile target so it runs as part of
      `make check-lite` and therefore on the pre-commit hook
- [ ] Split any existing Python source file that exceeds the limit
- [ ] Document the limit in `python/README.md`

## Acceptance Criteria

- `make lint-python` fails when a non-test file under `python/src/` exceeds 200 lines
- `make check-lite` runs the check (verify by temporarily appending filler lines to a
  source file and confirming the pre-commit hook rejects the commit)
- Test files are exempt
- `make check` passes on a clean tree
