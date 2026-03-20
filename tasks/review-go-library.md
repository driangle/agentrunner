# Go Library Review

Overall, this is a **well-designed, idiomatic Go library**. The architecture is clean, the test coverage is excellent, and the code follows Go conventions closely. Below are the issues found, organized by severity.

## Issues

### 1. Timeout cancel func is leaked

**Files:** `go/claudecode/runner.go:66-69`, `go/ollama/runner.go:67-70`

```go
if options.Timeout > 0 {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(ctx, options.Timeout)
    _ = cancel  // leaked
}
```

The timeout's `cancel` is deliberately discarded with `_ = cancel`. This leaks the timer goroutine created by `context.WithTimeout` until the timeout fires naturally. The `sessionCancel` on line 72 cancels the child context but does **not** release the timeout timer on the parent. Fix: defer the timeout cancel inside the goroutine, or combine it with `sessionCancel`.

### 2. `Message` lacks typed accessors (spec gap)

INTERFACE.md specifies typed accessors (`Text()`, `Thinking()`, `ToolName()`, etc.), but the Go `Message` struct only has `Type` and `Raw`. Callers must re-parse `Raw` themselves (as the example does repeatedly with `claudecode.Parse(string(msg.Raw))`). This makes the streaming example verbose and forces users to know about `StreamMessage` internals.

Consider adding accessor methods to `Message` or a helper like `claudecode.ParseMessage(msg) (StreamMessage, error)` that accepts a `Message` directly instead of requiring `string(msg.Raw)`.

### 3. `Session` is not safe to abandon (spec violation)

INTERFACE.md says: *"Abandoning a session (not draining messages, not awaiting result) should terminate the underlying process."*

Currently, if a caller calls `Start()` and discards the session without draining `Messages` or calling `Abort()`, the goroutine blocks forever on `msgCh <- msg` (`go/claudecode/runner.go:160`). The CLI process keeps running. There's no finalizer or GC-triggered cleanup.

### 4. `Extra map[any]any` — untyped extension mechanism

`Options.Extra` is `map[any]any`, which bypasses Go's type system. The key-type trick (`claudeOptsKey{}`, `onMessageKey{}`) is clever but fragile — a type assertion panic at `go/claudecode/options.go:132` (`v.(OnMessageFunc)`) is possible if someone stores the wrong type under that key. Consider using a checked type assertion with `ok`.

### 5. `mapStatusError` uses wrong sentinel for HTTP errors

**File:** `go/ollama/runner.go:297`

```go
return fmt.Errorf("%w: HTTP %d", agentrunner.ErrNonZeroExit, code)
```

`ErrNonZeroExit` is documented as "CLI process exited with a non-zero code" — it doesn't apply to HTTP status codes from the Ollama API. Consider a new `ErrHTTPError` sentinel or a more generic wrapping.

### 6. `Runner` doesn't verify it implements the interface

Neither `claudecode.Runner` nor `ollama.Runner` has a compile-time interface assertion:

```go
var _ agentrunner.Runner = (*Runner)(nil)
```

This is standard Go practice to catch drift between the interface and implementation at compile time.

### 7. No CLI version check

CLAUDE.md requires: *"when a runner starts, it should verify the installed CLI version falls within the supported range."* The README documents `>= 1.0.12` but the code never checks. A `claude --version` check in `Start` (or a lazy once-check) would satisfy this.

### 8. `scanner.Err()` checked too late

**File:** `go/claudecode/runner.go:205`

`scanner.Err()` is checked after `cmd.Wait()` and after the result-message check. If the scanner hits an I/O error, the loop exits, `resultMsg` is nil, `cmd.Wait()` may succeed (process already done), and we get `ErrNoResult` instead of the actual scanner error. Move the scanner error check before the `waitErr` check.

### 9. Error matching uses string comparison instead of `errors.Is`

**File:** `go/claudecode/runner_test.go:138` and several other tests

```go
if !strings.Contains(err.Error(), agentrunner.ErrNonZeroExit.Error()) {
```

This should use `errors.Is(err, agentrunner.ErrNonZeroExit)` since the errors are wrapped with `%w`. Multiple tests have this pattern.

## Minor / Style

### 10. `Parse` takes `string` but callers always have `[]byte`

`Parse(line string)` forces an extra allocation on `json.Unmarshal([]byte(line), ...)`. Since `msg.Raw` stores `[]byte`, accepting `[]byte` would be more natural and avoid one copy.

### 11. `--verbose` is always passed

`buildArgs` unconditionally adds `--verbose` (line 271 of `go/claudecode/runner.go`). This should be opt-in or at least documented. Users who don't want verbose CLI output have no way to suppress it.

### 12. Redundant option application in test

**File:** `go/claudecode/runner_test.go:248`

```go
WithAllowedTools("Read", "Write")(&agentrunner.Options{})  // thrown away
WithAllowedTools("Read", "Write")(opts)
```

The first line applies options to a throwaway `Options{}` and does nothing.

### 13. `DurationMs` type mismatch

`StreamMessage.DurationMs` is `float64` but `Result.DurationMs` is `int64`. The conversion at `go/claudecode/runner.go:328` truncates silently. Either make both the same type or use explicit rounding.

### 14. Example re-parses raw JSON unnecessarily

The streaming example (`examples/go/claudecode/main.go:133`) re-parses every message through `claudecode.Parse(string(msg.Raw))`. This is a consequence of issue #2 — if `Message` had typed accessors or carried parsed data, the example would be much cleaner.

### 15. `doc.go` files are nearly empty

Both `claudecode/doc.go` and `ollama/doc.go` contain just a package comment. These could live as the doc comment on the primary file (`runner.go`), which is the more common Go convention for small packages.

## What's done well

- **Functional options pattern** for both runner construction (`RunnerOption`) and invocation (`Option`) — idiomatic and extensible
- **Helper-process test pattern** for Claude — excellent way to test subprocess behavior without a real CLI
- **`httptest` for Ollama** — proper HTTP testing
- **Sentinel errors with `%w` wrapping** — correct use of Go error conventions
- **`Session` as the primary primitive** with `Run`/`RunStream` as convenience wrappers — clean layered design
- **Thorough test coverage** — happy path, errors, timeouts, cancellation, message ordering, callbacks, raw JSON preservation
- **`slog` integration** — opt-in structured logging is exactly right for Go
- **Content lifting in parser** — lifting `AssistantMsg.Content` to `StreamMessage.Content` is a nice ergonomic touch
- **Clean separation** between common interface (`runner.go`) and runner-specific packages (`claudecode/`, `ollama/`)
