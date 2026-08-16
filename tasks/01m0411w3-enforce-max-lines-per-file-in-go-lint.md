---
title: "Enforce max lines per file in Go lint"
id: "01m0411w3"
status: pending
priority: low
type: chore
tags: ["lint", "tooling"]
created: "2026-08-16"
phase: go
---

# Enforce max lines per file in Go lint

## Description

The TypeScript library caps files at 200 lines via the ESLint `max-lines` rule
(`ts/eslint.config.js`), enforced on every commit through the pre-commit hook.
Go has no equivalent: `lint-go` is just `go vet ./...`, which has no file-length
check.

Add a 200-line-per-file limit to the Go lint step so the file-organization
principle in `CLAUDE.md` is enforced uniformly across languages. Options are
`golangci-lint` with revive's `file-length-limit` rule, or a small script invoked
from `lint-go`. Adding golangci-lint means a new developer/CI dependency — weigh
that against the script approach.

## Tasks

- [ ] Pick the enforcement mechanism (golangci-lint + revive `file-length-limit`,
      or a small script run from `lint-go`)
- [ ] Set the limit to 200 lines, counting blank lines and comments, to match the
      TypeScript rule
- [ ] Exempt `_test.go` files (TypeScript disables `max-lines` for `test/**/*.ts`)
- [ ] Wire the check into the `lint-go` Makefile target so it runs as part of
      `make check-lite` and therefore on the pre-commit hook
- [ ] Split any existing Go source file that exceeds the limit
- [ ] Document the limit in `go/README.md` (and `go/CLAUDE.md` if relevant)

## Acceptance Criteria

- `make lint-go` fails when a non-test `.go` file exceeds 200 lines
- `make check-lite` runs the check (verify by temporarily appending filler lines to a
  source file and confirming the pre-commit hook rejects the commit)
- `_test.go` files are exempt
- `make check` passes on a clean tree
