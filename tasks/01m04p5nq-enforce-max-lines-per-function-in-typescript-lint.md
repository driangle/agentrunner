---
title: "Enforce max lines per function in TypeScript lint"
id: "01m04p5nq"
status: pending
priority: low
type: chore
tags: ["lint", "tooling"]
created: "2026-08-16"
phase: typescript
---

# Enforce max lines per function in TypeScript lint

## Description

`ts/eslint.config.js` caps files at 200 lines but places no limit on individual
function length. Add a 30-line-per-function limit via ESLint's
`max-lines-per-function` rule, matching the "keep functions small and focused"
principle in `CLAUDE.md`. The same limit is being applied to Go and Python so the
rule is uniform across languages.

## Tasks

- [ ] Add `max-lines-per-function` to `ts/eslint.config.js` with `max: 30`,
      counting blank lines and comments (consistent with the existing `max-lines`
      config)
- [ ] Disable the rule for `test/**/*.ts`, matching how `max-lines` is handled
- [ ] Refactor any source function that exceeds 30 lines by extracting helpers
- [ ] Confirm the rule runs via `npm run lint` → `make lint-ts` → `make check-lite`
      → pre-commit hook
- [ ] Document the limit in `ts/README.md` (or `ts/CLAUDE.md`)

## Acceptance Criteria

- `make lint-ts` fails when a function in `ts/src/` exceeds 30 lines
- `make check-lite` runs the check (verify by temporarily inflating a function and
  confirming the pre-commit hook rejects the commit)
- Test files are exempt
- `make check` passes on a clean tree
