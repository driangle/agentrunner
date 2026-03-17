---
title: "Add Claude Code streaming callback support"
id: "01kkx4369"
status: completed
priority: medium
type: feature
tags: ["go"]
created: "2026-03-17"
dependencies: ["01kkx4368"]
parent: 01kkx4yva
---

# Add streaming callback support

## Objective

Add support for receiving parsed messages as they arrive, enabling callers to display real-time progress, stream text deltas, or react to tool use events. Reference the doer `fmt/formatter.go` for how stream events are processed incrementally.

## Tasks

- [x] Implement `RunStream(ctx context.Context, prompt string, opts ...Option) (<-chan Message, <-chan error)` — returns a channel of parsed messages
- [x] Alternatively or additionally, support an `OnMessage` callback option
- [x] Ensure the result message is still the last message on the channel
- [x] Handle channel cleanup on context cancellation
- [x] Add tests verifying message ordering and channel closure

## Acceptance Criteria

- Callers can process messages incrementally as they arrive
- Stream text deltas (`content_block_delta` with `text_delta`) are delivered in order
- Channel is closed after the final `result` message or on error
- Context cancellation properly cleans up
