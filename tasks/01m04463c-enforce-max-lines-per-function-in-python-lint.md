---
title: "Enforce max lines per function in Python lint"
id: "01m04463c"
status: pending
priority: low
type: chore
tags: ["lint", "tooling"]
created: "2026-08-16"
phase: python
---

# Enforce max lines per function in Python lint

## Description

Python lint (`ruff check src/ tests/`) currently selects `["E", "F", "I", "W"]`,
none of which limit function length. Add a 30-line-per-function limit so the
"keep functions small and focused" principle in `CLAUDE.md` is enforced, matching
the limit being applied to Go and TypeScript.

Ruff has no exact lines-per-function rule. The closest built-in is pylint's
`PLR0915` (`too-many-statements`, configurable via
`[tool.ruff.lint.pylint] max-statements`), which counts statements rather than
lines. Decide between approximating with `PLR0915` or a small check script run
from `lint-python`; note the chosen semantics in the docs so the limit isn't
misread as a literal line count.

## Tasks

- [ ] Pick the enforcement mechanism (ruff `PLR0915` with a tuned
      `max-statements`, or a small script run from `lint-python`)
- [ ] Target a 30-line-per-function limit; if using `PLR0915`, record the
      statement-count equivalent chosen and why
- [ ] Exempt `python/tests/`, consistent with the other languages
- [ ] Wire the check into the `lint-python` Makefile target so it runs as part of
      `make check-lite` and therefore on the pre-commit hook
- [ ] Refactor any source function that exceeds the limit
- [ ] Document the limit in `python/README.md`

## Acceptance Criteria

- `make lint-python` fails when a function in `python/src/` exceeds the limit
- `make check-lite` runs the check (verify by temporarily inflating a function and
  confirming the pre-commit hook rejects the commit)
- Test files are exempt
- `make check` passes on a clean tree
