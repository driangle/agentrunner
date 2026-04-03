# Channels

Channels enable two-way communication with a running Claude Code session. While the agent is working, you can send it messages from external sources (CI notifications, webhook events, user input) and receive structured replies.

## How It Works

When you enable channels, agentrunner:

1. Starts an **agentrunner-channel** MCP server alongside the Claude CLI process
2. Creates a Unix socket for your code to send messages through
3. Forwards your messages to Claude as MCP channel notifications
4. Claude can reply using the `reply` tool, which surfaces as `channel_reply` messages in the stream

```
Your code ──send()──► Unix socket ──► MCP server ──► Claude Code
                                                        │
Your code ◄──stream── channel_reply ◄── reply tool ◄────┘
```

## Enabling Channels

::: code-group

```go [Go]
session, err := runner.Start(ctx, "Review this PR",
    claudecode.WithChannelEnabled(),
    agentrunner.WithSkipPermissions(),
)
```

```ts [TypeScript]
const session = runner.start("Review this PR", {
  channelEnabled: true,
  skipPermissions: true,
});
```

:::

## Sending Messages

Wait for the `system` init message before sending — this confirms the MCP server is connected.

::: code-group

```go [Go]
import "github.com/anthropics/agentrunner/go/channel"

for msg := range session.Messages {
    if msg.Type == agentrunner.MessageTypeSystem {
        // MCP server is ready — send a message.
        err := session.Send(channel.ChannelMessage{
            Content:    "Build #1234 failed on main",
            SourceID:   "ci-build-1234",
            SourceName: "GitHub Actions",
        })
        break
    }
}
```

```ts [TypeScript]
for await (const msg of session.messages) {
  if (msg.type === "system") {
    await session.send({
      content: "Build #1234 failed on main",
      sourceId: "ci-build-1234",
      sourceName: "GitHub Actions",
    });
    break;
  }
}
```

:::

### ChannelMessage Fields

| Field | Type | Description |
|-------|------|-------------|
| `content` | `string` | Message body that Claude reads |
| `sourceId` | `string` | Caller-defined correlation ID |
| `sourceName` | `string` | Human-readable origin (e.g. "GitHub Actions") |
| `replyTo` | `string?` | Optional reference to a prior message's sourceId |

## Receiving Replies

When Claude replies to a channel message, it emits a `channel_reply` message in the stream:

::: code-group

```go [Go]
for msg := range session.Messages {
    if msg.Type == agentrunner.MessageTypeChannelReply {
        fmt.Printf("Reply to %s: %s\n",
            msg.ChannelReplyDestination(),
            msg.ChannelReplyContent(),
        )
    }
}
```

```ts [TypeScript]
import {
  messageChannelReplyContent,
  messageChannelReplyDestination,
} from "agentrunner/claudecode";

for await (const msg of session.messages) {
  if (msg.type === "channel_reply") {
    const content = messageChannelReplyContent(msg);
    const dest = messageChannelReplyDestination(msg);
    console.log(`Reply to ${dest}: ${content}`);
  }
}
```

:::

## Channel Server Logging

The channel MCP server runs as a child process of Claude Code. By default it produces no log output. To enable logging for debugging, specify a log file and level:

::: code-group

```go [Go]
session, err := runner.Start(ctx, "Review this PR",
    claudecode.WithChannelEnabled(),
    claudecode.WithChannelLogFile("/tmp/channel.log"),
    claudecode.WithChannelLogLevel("debug"),
)
```

```ts [TypeScript]
const session = runner.start("Review this PR", {
  channelEnabled: true,
  channelLogFile: "/tmp/channel.log",
  channelLogLevel: "debug",
});
```

:::

Then tail the log file to see the MCP server's activity:

```sh
tail -f /tmp/channel.log
```

### Log Levels

| Level | What it logs |
|-------|-------------|
| `debug` | All of the below, plus every JSON-RPC request/response and notification delivery confirmations |
| `info` | Server start/stop, MCP handshake, channel messages forwarded, tool calls (default) |
| `warn` | Malformed input, unknown methods/tools |
| `error` | Failed writes, unrecoverable errors |

### What the Logs Show

The channel server logs the full lifecycle of a channel session:

- **Server start** — socket path being listened on
- **MCP handshake** — `initialize` and `initialized` messages from Claude Code
- **Channel messages** — each message forwarded from your code to Claude, with source_id and content length
- **Tool calls** — when Claude invokes the `reply` tool, with the full arguments
- **Shutdown** — clean shutdown or errors

This is useful for diagnosing issues like:
- The MCP server not starting (binary not found)
- Messages being sent but not reaching Claude (socket connected but notification not forwarded)
- Claude receiving messages but not replying (check if `reply` tool calls appear)

### Environment Variables

When running the channel binary directly (outside of agentrunner), logging is controlled via environment variables:

| Variable | Description |
|----------|-------------|
| `AGENTRUNNER_CHANNEL_LOG` | File path to write logs to. No logging when unset. |
| `AGENTRUNNER_CHANNEL_LOG_LEVEL` | Log level: `debug`, `info` (default), `warn`, `error`. |

## Prerequisites

The channel feature requires the `agentrunner-channel` binary. It is resolved in this order:

1. `AGENTRUNNER_CHANNEL_BIN` environment variable
2. Platform-specific npm optional dependency (TypeScript only)
3. System `PATH`
4. Built from source into user cache (Go only)

## Full Example

See [`examples/ts/channel/main.ts`](https://github.com/anthropics/agentrunner/blob/main/examples/ts/channel/main.ts) for a complete working example that sends a CI notification and prints Claude's reply.
