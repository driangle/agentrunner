# Channel Communication Example

[Experimental] Demonstrates two-way channel communication with Claude Code using the agentrunner TypeScript library, covering:

1. **Channel Session** — start a session with channels enabled and stream messages
2. **Send Channel Message** — send a CI build failure notification to Claude via the channel while it works
3. **Channel Replies** — receive and print Claude's channel reply messages from the stream

> **Note:** The channels feature in Claude Code is gated behind a server-side feature flag.
> In `-p` (print) mode, this flag must be enabled on your account.
> See `docs/guide/channels.md` for details.

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed (>= 1.0.12)
- Authenticated with `claude login`
- Node.js >= 18

## Setup

```sh
npm install
```

## Run

```sh
npx tsx main.ts
```

### Options

```
--binary <path>       Path to the Claude Code CLI binary (default: "claude")
--verbose             Enable debug logging
--debug-file <path>   Write debug output to a file
```

### Examples

```sh
# Use a custom binary path
npx tsx main.ts --binary /usr/local/bin/claude

# Enable debug logging to see the exact CLI command
npx tsx main.ts --verbose
```
