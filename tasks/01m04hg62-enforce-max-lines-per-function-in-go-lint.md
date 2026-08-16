---
title: "Enforce max lines per function in Go lint"
id: "01m04hg62"
status: pending
priority: low
type: chore
tags: ["lint", "tooling"]
created: "2026-08-16"
phase: go
---

# Enforce max lines per function in Go lint

## Description

Go lint is currently just `go vet ./...`, which places no limit on function
length. Add a 30-line-per-function limit so the "keep functions small and
focused" principle in `CLAUDE.md` is enforced, matching the limit being applied
to TypeScript and Python.

Options are `golangci-lint` with the `funlen` linter (which supports both a line
and statement limit), or a small script invoked from `lint-go`. If the
file-length task (`01m0411w3`) settles on golangci-lint, reuse that setup here
rather than introducing a second mechanism.

## Tasks

- [ ] Pick the enforcement mechanism (golangci-lint `funlen`, or a small script
      run from `lint-go`); coordinate with `01m0411w3`
- [ ] Set the limit to 30 lines per function
- [ ] Exempt `_test.go` files, consistent with the other languages
- [ ] Wire the check into the `lint-go` Makefile target so it runs as part of
      `make check-lite` and therefore on the pre-commit hook
- [ ] Refactor any source function that exceeds the limit
- [ ] Document the limit in `go/README.md` (and `go/CLAUDE.md` if relevant)

## Acceptance Criteria

- `make lint-go` fails when a function in a non-test `.go` file exceeds 30 lines
- `make check-lite` runs the check (verify by temporarily inflating a function and
  confirming the pre-commit hook rejects the commit)
- `_test.go` files are exempt
- `make check` passes on a clean tree
