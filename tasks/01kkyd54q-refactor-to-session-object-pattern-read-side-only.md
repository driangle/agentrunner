---
title: "Refactor to Session object pattern (read side only)"
id: "01kkyd54q"
status: pending
priority: high
type: feature
effort: large
tags: ["api", "refactor"]
created: "2026-03-17"
---

# Refactor to Session object pattern (read side only)

## Objective

Replace the current `Run`/`RunStream` dual-function API with a unified **Session** object that encapsulates a running agent process. The Session exposes the read side (messages iterable, result promise/future, abort) while `Run` and `RunStream` become thin convenience wrappers over it. The write side (`.send()`) is **out of scope** — reserve the shape but do not implement it until Claude Code supports mid-turn input.

### Session shape per language

| Language   | Messages (read)                        | Result                        | Abort               |
|------------|----------------------------------------|-------------------------------|----------------------|
| TypeScript | `AsyncIterable<Message>`               | `Promise<Result>`             | `abort(): void`      |
| Go         | `<-chan Message` or `Next()` iterator  | blocks or returns `Result`    | `context.Cancel()`   |
| Python     | `async for msg in session`             | `await session.result`        | `session.abort()`    |
| Java       | `Iterator<Message>` / `Flow.Publisher` | `CompletableFuture<Result>`   | `session.abort()`    |

### Convenience wrappers

- **`Run(prompt, opts)`** — start a session, drain messages internally, return result.
- **`RunStream(prompt, opts)`** — start a session, return messages iterable (result accessible via session).

## Tasks

- [ ] Define the `Session` type/interface for each language with: messages, result, abort, and a placeholder/reserved `send` method signature
- [ ] **Go**: refactor `claudecode` runner to return a `Session`; `Run` and `RunStream` wrap it
- [ ] **TypeScript**: refactor `claudecode` runner to return a `Session`; `Run` and `RunStream` wrap it
- [ ] **Python**: refactor `claudecode` runner to return a `Session`; `run` and `run_stream` wrap it
- [ ] **Java**: refactor `claudecode` runner to return a `Session`; `run` and `runStream` wrap it
- [ ] Update unit tests in all languages to test the Session API directly
- [ ] Update integration tests (fake CLI) to exercise Session lifecycle (messages → result, abort mid-stream)
- [ ] Update example programs to use the new API
- [ ] Update INTERFACE.md to document the Session pattern

## Acceptance Criteria

- A `Session` type exists in every language library with `.messages`, `.result`, `.abort()`, and a reserved `.send()` signature that throws/errors "not yet supported"
- `Run` and `RunStream` still exist as convenience functions that delegate to `Session`
- All existing tests pass (updated to new API)
- New tests cover: creating a session, iterating messages, awaiting result, aborting mid-stream
- Example programs work with the new API
- `make check` passes across all languages
- INTERFACE.md reflects the Session-based design
