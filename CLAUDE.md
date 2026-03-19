# agentrunner

Monorepo of language-native libraries for programmatically invoking AI coding agents (Claude Code, Gemini CLI, Codex CLI, Ollama).

## Project Structure

```
Makefile      # Top-level build: `make check` verifies all libraries
INTERFACE.md  # Language-agnostic Runner interface specification
ts/           # TypeScript library
go/           # Go library
python/       # Python library
java/         # Java library
tasks/        # taskmd task files
```

## Build & Verification

`make check` builds, lints, and tests every library. A pre-commit git hook runs it automatically.

When adding a new language library:
1. Add `check-<lang>`, `build-<lang>`, `lint-<lang>`, and `test-<lang>` targets to the Makefile.
2. Add `check-<lang>` as a dependency of the top-level `check` target.

This ensures every library is verified uniformly and the pre-commit hook catches regressions across all languages.

## Design Principles

### Common Runner interface

All libraries implement a shared, language-agnostic Runner interface. The interface should feel native in each language but follow the same conceptual shape:

- **Run(prompt, options) → Result** — execute a prompt and return the final result
- **RunStream(prompt, options) → Stream<Message>** — execute and stream messages as they arrive
- **Options** — common options (model, system prompt, working directory, timeout, env) plus runner-specific options
- **Result** — common result fields (text, usage, cost, duration, session ID, is_error) plus runner-specific fields
- **Message** — common message envelope (type, content) with runner-specific subtypes

Each runner (Claude Code, Gemini, Codex) implements this interface, so callers can swap runners without changing their code.

### Thin and transparent

Each library should be as thin as possible — a minimal layer over the underlying CLI or API. Avoid inventing abstractions, hiding behavior, or adding logic that isn't directly required by the interface. Callers should be able to predict what the library does by knowing what the underlying tool does. Pass options through, surface errors directly, and keep the code auditable at a glance. The goal is a convenience wrapper, not a framework.

### CLI version compatibility

Each library must explicitly declare which versions of each CLI it supports. This serves two purposes: informing users which CLI versions to install, and enabling the library to detect incompatible versions at runtime.

- **Documentation** — each library's README (or equivalent) must list the supported CLI version range for every runner it implements (e.g., "Claude Code CLI >= 1.0.12").
- **Code metadata** — each runner implementation must define its supported version range as a constant or configuration value in code. This enables runtime version checks and makes the compatibility contract grep-able and auditable.
- **Runtime check** — when a runner starts, it should verify the installed CLI version falls within the supported range and return a clear error if not. This prevents confusing failures from breaking CLI changes.

### Language-native conventions

Each library must feel native to its language. Follow the idiomatic structure, naming, error handling, packaging, and testing conventions of that language. Do not impose patterns from one language onto another.

- **Go**: modules, exported types, `error` returns, `context.Context`, table-driven tests
- **TypeScript**: npm package, async iterators, typed interfaces, Jest/Vitest tests
- **Python**: pyproject.toml, dataclasses/Pydantic, async/await, pytest
- **Java**: Maven/Gradle, builder pattern, CompletableFuture, JUnit tests

### Logging

Each library must allow the user to configure debug logging using the language's conventional logging approach. Logging should be opt-in — disabled by default, with no output unless the user explicitly provides a logger. At minimum, log the exact CLI command (binary, args, working directory) before execution so users can reproduce issues outside the library.

- **Go**: accept a `*slog.Logger` via an option; use `DebugContext` for command details
- **TypeScript**: TBD
- **Python**: use the standard `logging` module with a named logger (e.g. `logging.getLogger("agentrunner.claudecode")`)
- **Java**: use SLF4J

### Supported runners

#### Claude Code CLI (`claude`)

Interacts via `claude -p` (print/non-interactive mode) with `--output-format stream-json`. Key CLI flags:

- `--print` / `-p` — non-interactive mode
- `--output-format stream-json` — newline-delimited JSON events
- `--output-format json` — single JSON result
- `--model` — model selection
- `--system-prompt` / `--append-system-prompt` — prompt customization
- `--allowedTools` / `--disallowedTools` — tool permissions
- `--dangerously-skip-permissions` — skip permission prompts
- `--max-turns` — limit agentic turns
- `--max-budget-usd` — cost limit
- `--continue` / `--resume` — session management
- `--mcp-config` — MCP server configuration
- `--json-schema` — structured output
- `--include-partial-messages` — stream partial/incremental messages

Stream-JSON message types: `system` (init), `assistant` (text/thinking/tool_use), `user` (tool results), `result` (success/error with cost/usage/duration), `stream_event` (raw API events), `rate_limit_event`.

#### Gemini CLI (`gemini`)

TBD — research CLI flags and output format.

#### Codex CLI (`codex`)

TBD — research CLI flags and output format.

#### Ollama (local, `ollama`)

Local runner that talks to the Ollama HTTP API rather than shelling out to a CLI. Enables fully offline agent execution with locally-hosted models. TBD — research API endpoints and streaming format.

### Example programs

Each language library should include example programs under `examples/<lang>/claudecode/` that demonstrate the full API surface. At minimum, every example program should cover:

1. **Simple run** — single prompt, print result fields
2. **Streaming with `--include-partial-messages`** — use `includePartialMessages: true` to demonstrate real-time incremental streaming output
3. **Session resume** — multi-turn conversation via session IDs

When adding a new language library, include these examples from the start.

### Testing strategy

Every task must include unit tests for the code it introduces. Tests live alongside the code following each language's conventions.

Three levels of testing:

1. **Unit tests** — test individual functions, types, and logic in isolation. Pure functions, parsers, option builders, type conversions, error handling. No external processes or network calls. Every task must have unit tests.

2. **Integration tests** — test runner behavior using fake/stub CLIs. Build lightweight scripts that mimic the real CLI's output format (e.g., a shell script that prints stream-json lines to stdout) and verify the runner correctly builds commands, passes arguments, parses output, and handles errors. These validate the full Run/RunStream path without requiring the real CLI to be installed.

3. **E2E tests** — test against the real CLIs (`claude`, `gemini`, `codex`, `ollama`). These invoke the actual CLI binaries and verify real-world behavior. E2E tests should be gated behind a build tag or environment variable (e.g., `//go:build e2e`, `@pytest.mark.e2e`, `describe.skip`) so they don't run in CI by default, since they require CLI installation and may incur costs. Keep E2E tests minimal — a simple prompt that validates the round-trip works.

### Reference implementations (Go, Claude Code)

Three prior implementations exist as reference (none are perfect):

1. **doer** (`/Users/driangle/workplace/gg/doer/doer-1/apps/cli/internal/agent/claudecode/`)
   - Strengths: clean type system for stream-json parsing (`fmt/types.go`, `fmt/parser.go`), good test coverage, formatter for human-readable output
   - Weaknesses: tightly coupled to a specific `agent.Agent` interface, not a standalone library

2. **pons** (`/Users/driangle/workplace/gg/pons/pons-1/apps/eval/internal/runner/`)
   - Strengths: simple and focused, clean `Run()` function, good event collection
   - Weaknesses: minimal type safety for events, no streaming callback, eval-specific

3. **modpol** (`/Users/driangle/workplace/gg/modpol/modpol-1/apps/cli/internal/claude/`)
   - Strengths: sentinel errors, `CommandBuilder` for testability, `ReadStreamResult` helper, progress hooks
   - Weaknesses: limited event type coverage, app-specific concerns mixed in

# Coding Principles

This document defines the core principles and preferences for code written in this environment.

---

## File Organization

**Give each file one clear responsibility.**
Files that mix concerns become harder to reason about and riskier to modify.

**Split by concept, not size.**
Extract cohesive modules (features, behaviors, domains), not arbitrary chunks.

**Optimize for readability.**
A file should be scannable in a minute or two. If you're constantly scrolling, it's too big.

**Prefer composition over growth.**
Add new functionality with new modules instead of piling onto existing ones.

**Keep boundaries explicit.**
Small public interfaces and hidden internals reduce coupling and make changes safer.

**Group by feature, not "utils."**
Related code together, unrelated code apart improves discoverability.

**Refactor early and often.**
Split files when they start feeling heavy—before they become "god files."

---

## Core Design Principles

### 1. Clarity over cleverness
Code is read far more than written. Favor straightforward solutions, obvious naming, and simple control flow. If something needs a comment to explain "how it works," it's probably too complex.

### 2. Keep it simple (KISS)
Start with the simplest design that solves today's problem. Extra abstractions, layers, or patterns "just in case" usually become liabilities.

### 3. Design for change
Requirements evolve. Structure code so changes are localized:
- Small modules
- Clear boundaries
- Minimal coupling

Aim to modify one place, not five.

### 4. Single responsibility
Each module/class/function should do one thing well. Mixed responsibilities create ripple effects and fragile code.

### 5. Prefer composition over inheritance
Compose behavior from small pieces instead of deep hierarchies. It's easier to extend, test, and reason about.

### 6. Explicit interfaces and contracts
Make inputs, outputs, and assumptions clear. Avoid hidden side effects. Strong typing, validation, and documented invariants reduce bugs dramatically.

### 7. Test what matters
Test behavior, not implementation. Focus on:
- Critical paths
- Edge cases
- Regressions

A few meaningful tests beat many brittle ones.

### 8. Automate everything repeatable
Builds, tests, linting, formatting, deployments. Manual steps create inconsistency and slow teams down.

### 9. Measure before optimizing
Premature optimization adds complexity. Profile first, then fix the real bottlenecks.

### 10. Leave the codebase better than you found it
Small, continuous improvements (naming, refactoring, cleanup) compound into long-term maintainability.

---

## Implementation Guidelines

When writing or modifying code:
- Start with the simplest solution
- Keep functions small and focused
- Use clear, descriptive names
- Avoid premature abstractions
- Make dependencies explicit
- Validate at system boundaries only
- Don't add "just in case" code
- Refactor when things feel heavy, not after they're unwieldy
