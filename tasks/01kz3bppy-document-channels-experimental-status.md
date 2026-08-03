---
title: "Document channels as experimental/account-gated; reassess vs. thinness"
id: "01kz3bppy"
status: pending
priority: medium
type: chore
tags: ["docs", "channel"]
created: "2026-08-03"
---

# Document channels as experimental/account-gated; reassess vs. thinness

## Description

The channels feature is the most complex and most-distributed part of the codebase: a standalone MCP server binary (`go/cmd/agentrunner-mcp`), a unix-socket JSON-RPC server, cross-compilation to five platforms, and per-platform npm distribution packages (`npm/channel-*`). By the runner's own comment (`go/claudecode/runner.go:411-425`), channels are gated behind a **server-side account feature flag** and require two `--dangerously-*` flags in `-p` mode — so for most users this code path may not function at all.

This sits in tension with the project's stated "thin and transparent" principle ("as thin as possible… a convenience wrapper, not a framework"). Two things are needed: make the experimental/account-gated status obvious to users, and take a deliberate position on whether channels belong in a thin wrapper library.

## Tasks

- [ ] Add a clear "experimental — requires an account-level feature flag, may not work in `-p` mode" banner to the channels documentation (`docs/guide/channels.md`) and each library README that exposes channel APIs.
- [ ] Record an explicit decision on channels vs. the thinness principle: keep as clearly-labeled experimental, split into a separate package, or otherwise scope it.
- [ ] Ensure examples that use channels state the prerequisite up front so they don't appear broken to users without the flag.

## Tasks (checklist)

- [ ] Experimental/account-gated banner added to docs and relevant READMEs
- [ ] Positioning decision recorded (keep-experimental / split / scope)
- [ ] Channel examples note the feature-flag prerequisite
