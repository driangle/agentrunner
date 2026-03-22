---
title: "Two-way channels support via session.Send()"
id: "01kma0s35"
status: in-progress
priority: high
type: feature
tags: ["channels", "mcp", "cross-platform"]
created: "2026-03-22"
---

# Two-way channels support via session.Send()

## Objective

Enable two-way communication with a running Claude Code session through the `--channels` flag. Clients call `session.Send()` to push messages into an active session and receive Claude's channel replies on the message stream — without needing to configure an MCP server themselves. The library handles channel server lifecycle, IPC, and CLI flag wiring transparently.

## Architecture

```
┌─────────────────────┐
│  App (Go/TS/Py)      │
│                      │
│  session.Send() ─────┼──► Unix Socket ──► ┌──────────────────────┐
│                      │                    │  agentrunner-channel  │
│  session.Messages ◄──┼─── stream-json     │  (Go MCP server)     │
│  (includes replies)  │                    │  stdio ↔ Claude CLI   │
└─────────────────────┘                    └──────────────────────┘
```

- **agentrunner-channel**: Go binary, MCP server over stdio (Claude side), Unix socket listener (library side)
- **IPC**: Unix socket, path from `AGENTRUNNER_CHANNEL_SOCK` env var
- **CLI flags**: `--channels server:agentrunner-channel --dangerously-load-development-channels`

## Message Format

```
ChannelMessage {
  content:     string  // message body Claude reads
  source_id:   string  // caller-defined correlation ID
  source_name: string  // human-readable origin (e.g. "github-webhook")
  reply_to:    string  // optional, references prior message's source_id
}
```

## Tasks

### Channel server binary
- [ ] Create `cmd/agentrunner-channel/` Go binary
- [ ] Implement MCP protocol over stdio by hand (no MCP library imports — keep the binary lightweight). Reference: https://github.com/modelcontextprotocol/go-sdk for protocol details (JSON-RPC framing, capability negotiation, notification/tool schemas)
- [ ] Handle `initialize` handshake, advertising `experimental: { 'claude/channel': {} }` capability and `tools` capability
- [ ] Implement Unix socket listener (path from `AGENTRUNNER_CHANNEL_SOCK`)
- [ ] Forward socket messages as `notifications/claude/channel` notifications with content + meta
- [ ] Handle `tools/list` and `tools/call` JSON-RPC requests for the `reply` tool
- [ ] Add `instructions` string telling Claude how to use the channel and reply tool
- [ ] Unit tests for the channel server (JSON-RPC framing, notification encoding, tool dispatch)

### Cross-compilation and distribution
- [ ] Add Makefile targets for cross-compiling to linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- [ ] npm: platform-specific `optionalDependencies` packages (esbuild/turbo pattern)
- [ ] PyPI: platform wheels with binary included
- [ ] Go: `//go:embed` the binary or build as module dependency
- [ ] Release automation: cross-compile and place binaries before publishing

### Go library integration
- [ ] Add `WithChannelEnabled()` option to ClaudeOptions
- [ ] On start: create temp Unix socket, generate MCP config, merge with user mcpConfig, set env var
- [ ] Wire `--channels server:agentrunner-channel` and `--dangerously-load-development-channels` into arg builder
- [ ] Implement `session.Send(ctx, ChannelMessage)` — connect to socket and write message
- [ ] Parse channel reply messages from stream into `MessageTypeChannelReply`
- [ ] Unit and integration tests

### TypeScript library integration
- [ ] Add `channelEnabled` option to ClaudeRunOptions
- [ ] Same start-time wiring (temp socket, MCP config merge, env var, CLI flags)
- [ ] Implement `session.send(ChannelMessage)`
- [ ] Parse channel replies into typed messages on the stream
- [ ] Unit and integration tests

### Python library integration
- [ ] Add `channel_enabled` option to ClaudeRunOptions
- [ ] Same start-time wiring
- [ ] Implement `session.send(ChannelMessage)`
- [ ] Parse channel replies into typed messages on the async iterator
- [ ] Unit and integration tests

### Examples
- [ ] Go example: two-way channel communication
- [ ] TypeScript example: two-way channel communication
- [ ] Python example: two-way channel communication

## Acceptance Criteria

- Go channel server binary builds and works as MCP server over stdio
- Channel server receives messages on Unix socket and pushes to Claude via `notifications/claude/channel`
- Channel server exposes `reply` MCP tool for Claude to respond back
- Cross-compilation produces binaries for 5 platform targets
- Go library: `session.Send()` delivers messages, channel replies appear on message stream
- TypeScript library: same
- Python library: same
- Binary embedding works for each package format (npm, PyPI, Go)
- Unit tests for channel server, IPC protocol, message parsing
- Integration tests with fake channel server
- Example programs for each language demonstrating two-way channel communication
