---
title: "Publish Go library as Go module"
id: "01kky7q1z"
status: completed
priority: low
type: chore
tags: ["release", "gomod"]
created: "2026-03-17"
---

# Publish Go library as Go module

## Objective

Publish the Go agentrunner library as a Go module so users can import it via `go get`. Go modules are published by tagging a release in the Git repository with a semver tag. Ensure the module path, versioning, and directory structure are correct for consumption.

## Tasks

- [ ] Verify `go.mod` has the correct module path (e.g., `github.com/<org>/agentrunner/go` or a subdirectory-based path)
- [ ] Ensure all exported types, functions, and packages have GoDoc comments
- [ ] Run `go vet` and `go mod tidy` to ensure the module is clean
- [ ] Tag the release with the appropriate semver tag (e.g., `go/v0.1.0` for subdirectory modules or `v0.1.0`)
- [ ] Push the tag to the remote repository
- [ ] Verify the module appears on pkg.go.dev after tagging
- [ ] Verify `go get` works from a fresh project

## Acceptance Criteria

- Module is importable via `go get <module-path>`
- Module appears on pkg.go.dev with documentation
- `go mod tidy` in a consuming project resolves the dependency cleanly
- All exported symbols have GoDoc comments
