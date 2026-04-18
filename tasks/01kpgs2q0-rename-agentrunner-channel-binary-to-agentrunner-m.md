---
title: "Rename agentrunner-channel binary to agentrunner-mcp"
id: "01kpgs2q0"
status: pending
priority: high
type: chore
tags: ["go", "mcp"]
effort: small
created: "2026-04-18"
---

# Rename agentrunner-channel binary to agentrunner-mcp

## Description

The `agentrunner-channel` binary is evolving from a channel-only MCP server into a general-purpose MCP bridge (channels + custom tools). Rename it to `agentrunner-mcp` to reflect its broader role.

This is a prerequisite for the custom tools feature (`01kpgqegk`).

## Tasks

- [ ] Rename `go/cmd/agentrunner-channel/` directory to `go/cmd/agentrunner-mcp/`
- [ ] Update binary name in `go/cmd/agentrunner-mcp/main.go` (comment, error messages)
- [ ] Update `channel.ServerName` constant in `go/channel/mcp.go`
- [ ] Update `channel.BinaryPath()` in `go/channel/binary.go` — LookPath name, build path, env var (`AGENTRUNNER_CHANNEL_BIN` → `AGENTRUNNER_MCP_BIN`)
- [ ] Update `go/channel/binary_test.go` and `go/channel/smoke_test.go`
- [ ] Update MCP config key in `go/claudecode/channel.go` (`"agentrunner-channel"` → `"agentrunner-mcp"`)
- [ ] Update `--channels` and `--dangerously-load-development-channels` flags in `go/claudecode/runner.go`
- [ ] Update `channelReplyToolName` constant in `go/claudecode/types.go` (`mcp__agentrunner-channel__reply` → `mcp__agentrunner-mcp__reply`)
- [ ] Update `go/claudecode/channel_test.go` and `go/claudecode/runner_test.go`
- [ ] Update Makefile build and dist targets
- [ ] Run `make check-go` and ensure all checks pass

## Acceptance Criteria

- Binary is named `agentrunner-mcp` everywhere (source, build output, MCP config, CLI flags)
- No remaining references to `agentrunner-channel` as a binary name (the `channel` Go package name stays — it describes the feature, not the binary)
- All existing tests pass with the new name
- `make check-go` passes
