---
title: "Stream-JSON input for client-to-session messaging"
id: "01kp0r2af"
status: pending
priority: high
type: feature
tags: ["channels", "stream-json"]
created: "2026-04-12"
depends-on: ["01kma0s35"]
---

# Stream-JSON input for client-to-session messaging

## Objective

Add support for `--input-format stream-json` as the primary transport for `session.send()`, replacing the current MCP channel notification approach (`notifications/claude/channel`). The MCP-based channel feature is gated behind a server-side feature flag (`tengu_harbor`) that is not enabled in `-p` (print) mode, making it unusable for agentrunner. Stream-JSON input bypasses this limitation entirely by writing messages directly to Claude's stdin.

Reference: https://github.com/anthropics/claude-code/issues/24594

## Background

The current channel architecture:
```
session.send() → Unix socket → agentrunner-channel MCP server → notifications/claude/channel → Claude
```

This fails in `-p` mode because Claude Code's `--dangerously-load-development-channels` bypass only runs in interactive mode. The `tengu_harbor` feature flag gates all channel registrations via `nH_()`, which returns `{action: "skip"}` when the flag is off.

The proposed architecture:
```
session.send() → stdin (stream-json) → Claude
```

Claude Code supports `--input-format stream-json` in `-p` mode, allowing JSON events to be written to stdin while the session is running. This requires `--output-format stream-json` as well (which we already use).

## Tasks

### Research
- [ ] Document the exact stream-json input message format Claude accepts (message types, required fields, envelope structure)
- [ ] Determine how user-injected messages appear in Claude's conversation context (as user messages? system messages? tool results?)
- [ ] Verify `--input-format stream-json` works with `--output-format stream-json` and `--print` simultaneously
- [ ] Test manually: start Claude with `--input-format stream-json --output-format stream-json -p` and write JSON to stdin to confirm messages are received mid-session

### Implementation — TypeScript
- [ ] Add `inputFormat: "stream-json"` option to `ClaudeRunOptions`
- [ ] When `channelEnabled` (or a new `sendEnabled`) is set, pass `--input-format stream-json` to the CLI
- [ ] Implement `session.send()` by writing the appropriate JSON envelope to the child process's stdin
- [ ] Keep the existing `ChannelMessage` interface (or a compatible one) so the API surface doesn't change
- [ ] Handle backpressure on stdin writes
- [ ] Update `session.send()` to reject if stdin is closed or the process has exited

### Implementation — Go
- [ ] Same changes as TypeScript: add input-format flag, implement `session.Send()` via stdin
- [ ] Ensure `Send()` is safe to call concurrently (stdin writes must be serialized)

### Migration
- [ ] Decide whether to keep the MCP channel path as a fallback or remove it entirely
- [ ] If keeping both: add a `channelTransport: "stream-json" | "mcp"` option with `"stream-json"` as default
- [ ] Update examples (`examples/ts/channel/main.ts`, `examples/go/channel/main.go`) to use the new transport
- [ ] Update `docs/guide/channels.md` to document the new default and remove the experimental warning once stable

### Tests
- [ ] Unit test: verify `--input-format stream-json` is added to CLI args when enabled
- [ ] Unit test: verify `session.send()` writes correct JSON to stdin
- [ ] Integration test: fake CLI that reads stdin stream-json and echoes messages back
- [ ] Update existing channel tests to cover the new transport

## Acceptance Criteria

- `session.send()` delivers messages to Claude via stdin stream-json without requiring the `tengu_harbor` feature flag
- The API surface (`session.send(msg)`, `ChannelMessage` type) remains the same or compatible
- Works reliably in `-p` mode with `--output-format stream-json`
- Messages sent via `session.send()` appear in Claude's conversation context and Claude can act on them
- Go and TypeScript implementations are both updated
- Examples and docs are updated
- Unit and integration tests pass
