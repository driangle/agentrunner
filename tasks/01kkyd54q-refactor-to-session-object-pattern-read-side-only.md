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

### TypeScript reference API

The Runner gains a `start()` primitive; `run` and `runStream` delegate to it:

```typescript
interface Session {
  messages: AsyncIterable<Message>;
  result: Promise<Result>;
  abort(): void;
  send(input: unknown): void; // reserved — throws "not yet supported"
}

interface Runner {
  start(prompt: string, options?: RunOptions): Session;
  run(prompt: string, options?: RunOptions): Promise<Result>;
  runStream(prompt: string, options?: RunOptions): AsyncIterable<Message>;
}
```

Usage:

```typescript
const claude = createClaudeRunner();

// simple run (unchanged)
const result = await claude.run("explain this repo");

// streaming (unchanged)
for await (const msg of claude.runStream("refactor auth")) { ... }

// session: full control
const session = claude.start("fix the tests", { model: "opus" });
for await (const msg of session.messages) { console.log(msg.type); }
const result = await session.result;

// session: abort mid-stream
const session = claude.start("long task");
for await (const msg of session.messages) {
  if (tooExpensive(msg)) { session.abort(); break; }
}
```

### Go reference API

```go
type Session struct {
    Messages <-chan Message
}

func (s *Session) Result() (Result, error)  // blocks until done
func (s *Session) Abort()                   // cancels context + kills process
func (s *Session) Send(input any) error     // returns ErrNotSupported

type Runner struct { /* config, binary, logger */ }

func (r *Runner) Start(ctx context.Context, prompt string, opts ...Option) *Session
func (r *Runner) Run(ctx context.Context, prompt string, opts ...Option) (Result, error)
func (r *Runner) RunStream(ctx context.Context, prompt string, opts ...Option) <-chan Message
```

Usage:

```go
claude := claudecode.New()

// simple run
result, err := claude.Run(ctx, "explain this repo")

// streaming
for msg := range claude.RunStream(ctx, "refactor auth") {
    fmt.Println(msg.Type)
}

// session: full control
session := claude.Start(ctx, "fix the tests", claudecode.WithModel("opus"))
for msg := range session.Messages {
    fmt.Println(msg.Type)
}
result, err := session.Result()

// session: abort mid-stream
session := claude.Start(ctx, "long task")
for msg := range session.Messages {
    if tooExpensive(msg) {
        session.Abort()
        break
    }
}

// future (not yet): send mid-turn (e.g. permission responses)
session := claude.Start(ctx, "delete unused test files")
for msg := range session.Messages {
    if msg.Type == "permission_request" {
        var req PermissionRequest
        json.Unmarshal(msg.Raw, &req)
        allowed := reviewPermission(req.Tool, req.Args)
        session.Send(PermissionResponse{
            RequestID: req.RequestID,
            Allow:     allowed,
        })
    }
}
result, err := session.Result()
```

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
