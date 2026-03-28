---
id: "01kmse9cq"
title: "Channel server binary (Go MCP server over stdio)"
status: completed
priority: high
effort: large
parent: "01kma0s35"
dependencies: []
tags: ["channels", "mcp", "go"]
created: 2026-03-28
---

# Channel server binary (Go MCP server over stdio)

## Objective

Build the `agentrunner-channel` Go binary — a lightweight MCP server that bridges Unix socket IPC from the agentrunner libraries to Claude Code's channel system via stdio JSON-RPC.

## Architecture

```
App (via Unix socket) ──► agentrunner-channel (stdio MCP server) ──► Claude CLI
```

- **Stdio side**: JSON-RPC over stdin/stdout, implementing MCP protocol (initialize handshake, tools, notifications)
- **Socket side**: Unix socket listener, receives `ChannelMessage` structs from library clients
- **Path**: Socket path from `AGENTRUNNER_CHANNEL_SOCK` environment variable

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

- [x] Create `cmd/agentrunner-channel/` Go binary with main entry point
- [x] Implement MCP protocol over stdio by hand (no MCP library imports — keep the binary lightweight). Reference: https://github.com/modelcontextprotocol/go-sdk for protocol details (JSON-RPC framing, capability negotiation, notification/tool schemas)
- [x] Handle `initialize` handshake, advertising `experimental: { 'claude/channel': {} }` capability and `tools` capability
- [x] Implement Unix socket listener (path from `AGENTRUNNER_CHANNEL_SOCK`)
- [x] Forward socket messages as `notifications/claude/channel` notifications with content + meta
- [x] Handle `tools/list` and `tools/call` JSON-RPC requests for the `reply` tool
- [x] Add `instructions` string telling Claude how to use the channel and reply tool
- [x] Unit tests for the channel server (JSON-RPC framing, notification encoding, tool dispatch)

## Acceptance Criteria

- Go channel server binary builds and runs as MCP server over stdio
- Channel server receives messages on Unix socket and pushes to Claude via `notifications/claude/channel`
- Channel server exposes `reply` MCP tool for Claude to respond back
- Unit tests cover JSON-RPC framing, notification encoding, and tool dispatch
