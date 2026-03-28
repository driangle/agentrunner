---
id: "01kmse9da"
title: "Channel server cross-compilation and distribution"
status: completed
priority: high
effort: medium
parent: "01kma0s35"
dependencies: ["01kmse9cq"]
tags: ["channels", "distribution"]
created: 2026-03-28
---

# Channel server cross-compilation and distribution

## Objective

Set up cross-compilation of the `agentrunner-channel` binary for all supported platforms and embed/distribute it within each language's package format (npm, PyPI, Go module).

## Tasks

- [x] Add Makefile targets for cross-compiling to linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- [x] npm: platform-specific `optionalDependencies` packages (esbuild/turbo pattern)
- [x] PyPI: platform wheels with binary included
- [x] Go: `//go:embed` the binary or build as module dependency
- [x] Release automation: cross-compile and place binaries before publishing

## Acceptance Criteria

- Cross-compilation produces binaries for 5 platform targets (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- Binary embedding works for each package format (npm, PyPI, Go)
- Release automation cross-compiles and places binaries before publishing
