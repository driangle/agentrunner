---
title: "Implement Claude Code stream-json parser"
id: "01kkx4367"
status: completed
priority: high
type: feature
tags: ["go"]
created: "2026-03-17"
dependencies: ["01kkx4366"]
parent: 01kkx4yva
---

# Implement stream-json parser

## Objective

Implement a parser that reads newline-delimited JSON from the CLI's `stream-json` output and produces typed Go values. Reference the doer `fmt/parser.go` for approach — parse the envelope, then lift nested content for assistant messages and stream events.

## Tasks

- [x] Implement `Parse(line string) (Message, error)` — single line to typed message
- [x] Handle assistant message content lifting (content blocks from nested `message` wrapper)
- [x] Handle stream_event inner event parsing
- [x] Silently skip unknown message types for forward compatibility
- [x] Add table-driven tests for each message type (system/init, assistant/text, assistant/thinking, assistant/tool_use, result/success, result/error, stream_event/*, rate_limit_event)
- [x] Add test with real session fixture (JSONL file) if available

## Acceptance Criteria

- `Parse` correctly deserializes all known message types
- Unknown types are parsed without error (forward compatible)
- Invalid JSON returns an error
- Tests cover all message types with realistic payloads
