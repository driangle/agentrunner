---
title: "CI workflow to publish Go module on release"
id: "01kky7rk1"
status: completed
priority: low
type: chore
tags: ["ci", "release"]
created: "2026-03-17"
---

# CI workflow to publish Go module on release

## Objective

Create a GitHub Actions workflow that validates the Go module when a Go-specific release tag is pushed. Go modules are published automatically by the Go module proxy when a tag exists, so the CI workflow focuses on validation. Use a tag convention like `go/v*` (e.g., `go/v0.1.0`) to enable independent releases.

## Tasks

- [x] Create `.github/workflows/publish-go.yml` GitHub Actions workflow
- [x] Trigger on push of tags matching `go/v*` pattern
- [x] Add job steps: checkout, setup Go, run `make check-go`
- [x] Verify the module is fetchable via `GOPROXY=https://proxy.golang.org go list -m <module>@<version>`
- [x] Optionally create a GitHub Release from the tag with auto-generated notes
- [x] Document the release process in the Go library README

## Acceptance Criteria

- Pushing a tag like `go/v0.1.0` triggers the validation workflow
- The workflow runs `make check-go` before confirming the release
- Publishing one library does not trigger publishing of other libraries
- The module is resolvable on pkg.go.dev after the tag is pushed
