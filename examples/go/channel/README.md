# Channel Communication Example

[Experimental] Demonstrates two-way channel communication with Claude Code using the agentrunner Go library, covering:

1. **Channel Session** — start a session with channels enabled and stream messages
2. **Send Channel Message** — send a CI build failure notification to Claude via the channel while it works
3. **Channel Replies** — receive and print Claude's channel reply messages from the stream

> **Note:** The channels feature in Claude Code is gated behind a server-side feature flag.
> In `-p` (print) mode, this flag must be enabled on your account.
> See `docs/guide/channels.md` for details.

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed (>= 1.0.12)
- Authenticated with `claude login`
- Go >= 1.22

## Setup

```sh
go mod tidy
```

## Run

```sh
go run .
```

### Options

```
--claude <path>   Path to the Claude Code CLI binary (default: "claude")
--verbose         Enable debug logging
```

### Examples

```sh
# Use a custom binary path
go run . --claude /usr/local/bin/claude

# Enable debug logging to see the exact CLI command
go run . --verbose
```
