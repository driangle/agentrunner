---
title: "Clarify version-check principle scope for HTTP runners (Ollama)"
id: "01kz3bpq0"
status: pending
priority: low
type: chore
tags: ["docs", "ollama"]
created: "2026-08-03"
---

# Clarify version-check principle scope for HTTP runners (Ollama)

## Description

The "CLI version compatibility" principle in `CLAUDE.md` is written as if every runner shells out to a versioned CLI, but Ollama talks to an HTTP API and has no `--version` binary check — so the "runtime version check" requirement quietly doesn't apply to it. Ollama support is also asymmetric (Go and TypeScript implemented, Python "Planned"), which is fine on its own but leaves the principle's scope ambiguous.

State explicitly which runner classes the version-check requirement applies to, so the omission for HTTP/API runners reads as intentional rather than as a gap.

## Tasks

- [ ] Update the `CLAUDE.md` version-compatibility section to scope the runtime version check to CLI-based runners, and describe what (if anything) API-based runners should check instead (e.g. endpoint reachability / API version, or explicitly nothing).
- [ ] Confirm the Ollama READMEs/docs don't imply a CLI version requirement that doesn't exist.
- [ ] Note the Python Ollama runner remains "Planned" so the support matrix stays consistent.

## Tasks (checklist)

- [ ] `CLAUDE.md` scopes the version-check requirement by runner class
- [ ] Ollama docs verified consistent
- [ ] Support matrix (Python "Planned") verified
