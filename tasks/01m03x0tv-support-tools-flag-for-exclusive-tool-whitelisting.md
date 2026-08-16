---
id: "01m03x0tv"
title: "Support --tools flag for exclusive tool whitelisting in Claude Code runners"
status: completed
priority: high
effort: medium
dependencies: []
tags: ["claudecode", "tools", "security"]
created_at: 2026-08-16
completed_at: 2026-08-16
---

# Support --tools flag for exclusive tool whitelisting in Claude Code runners

## Objective

Expose the Claude Code CLI `--tools` flag through the Claude Code runners so callers
can restrict a session to an **exclusive** set of built-in tools. Unlike
`--allowedTools` (which only pre-approves and is *not* exclusive — unlisted built-ins
still execute), `--tools` replaces the built-in tool set, so every unlisted built-in
is denied at registration and future built-ins are denied automatically.

CLI reference (`claude --help`): `--tools <tools...>` — *"Specify the list of
available tools from the built-in set. Use `""` to disable all tools, `"default"` to
use all tools, or specify tool names (e.g. `Bash,Edit,Read`)."*

## Motivation

Downstream (skival) needs a variant's `allowed_tools` to behave as a structural
whitelist that denies unlisted built-ins — verified empirically that `--tools
"Read,Grep"` yields an exclusive set (`system/init.tools == ["Grep","Read"]`) and
rejects a hallucinated `Write` call with *"No such tool available… in subagents as
well as here"*, whereas `--allowedTools` alone lets `Bash` run. The runner currently
emits only `--allowedTools`/`--disallowedTools`, so there is no way to enforce this
without a hand-maintained complement of built-in names. See the skival spec
`docs/specs/tool-deny-enforcement.md` (task `01m03awyn`).

## Design notes

- Go: add `Tools []string` to `ClaudeOptions` and a `WithTools(tools ...string)`
  option (`go/claudecode/options.go`); in `buildArgs` (`go/claudecode/runner.go`)
  append `--tools`, `strings.Join(co.Tools, ",")` when non-empty. `--tools` takes a
  single comma/space-separated value, not a repeated flag like `--allowedTools`.
- Mirror in the TypeScript (`ts/src/claudecode`) and Python
  (`python/src/agentrunner/claudecode`) runners.
- Semantics to preserve: empty string `""` disables all tools; omitting the option
  leaves the CLI default (all tools). Only emit `--tools` when the caller sets it.
- `--tools` acts at tool registration and is independent of the permission system, so
  it composes with `--dangerously-skip-permissions` (denied tools stay denied).
- Version floor: `--tools` requires a newer CLI than the current
  `MinCLIVersion = 1.0.12`. Bump the minimum (or make it injectable) so the version
  gate does not silently pass while the flag is ignored on old CLIs.

## Tasks

- [x] Go: add `Tools` option + `WithTools(...)` and emit `--tools "<joined>"` in `buildArgs`
- [x] TypeScript: add the equivalent option and arg building
- [x] Python: add the equivalent option and arg building
- [x] Unit tests in each language's arg builder (set, empty-string-disables-all, unset)
- [x] Reconcile `MinCLIVersion` with the CLI version that introduced `--tools`
- [x] Document `--tools` in `CLAUDE.md` under the Claude Code CLI flags section

## Acceptance Criteria

- Each runner accepts a tools option and, when set, passes `--tools "<comma-joined>"`
  to the CLI subprocess (single value, not repeated flags)
- `""` disables all tools; unset leaves the CLI default; only emitted when set
- New unit tests cover the flag; existing tests continue to pass
- `CLAUDE.md` lists `--tools` as a supported flag so future language libraries include it
</content>
