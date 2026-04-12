// [Experimental] This example demonstrates two-way channel communication with Claude Code.
// It starts a session with channels enabled, sends a CI notification to Claude
// via the channel, and prints any channel replies from the stream.
//
// IMPORTANT: The channels feature in Claude Code is gated behind a server-side
// feature flag. In -p (print) mode — which agentrunner uses — this flag must be
// enabled on your account. If the MCP server logs show "forwarding channel
// message" but Claude never acts on it, the feature flag is likely not enabled.
// Channels work in interactive mode (no -p) regardless of the flag.
// See docs/guide/channels.md for details.
//
// Prerequisites:
//   - Claude Code CLI installed (>= 1.0.12): https://docs.anthropic.com/en/docs/claude-code
//   - Authenticated with `claude login`
//   - agentrunner-channel binary available (via npm optional dep or $PATH)
//
// Run:
//   npx tsx main.ts
//   npx tsx main.ts --binary /path/to/claude

import { parseArgs } from "node:util";
import {
  createClaudeRunner,
  messageTextDelta,
  messageChannelReplyContent,
  messageChannelReplyDestination,
} from "agentrunner/claudecode";
import type {
  SystemStreamMessage,
  ResultStreamMessage,
} from "agentrunner/claudecode";
import type { ChannelMessage } from "agentrunner/channel";

const { values } = parseArgs({
  options: {
    binary: { type: "string", default: "claude" },
    verbose: { type: "boolean", default: false },
    "debug-file": { type: "string" },
  },
});

const logger = values.verbose
  ? {
      debug: (msg: string, ...args: unknown[]) =>
        console.error("[debug]", msg, ...args),
      error: (msg: string, ...args: unknown[]) =>
        console.error("[error]", msg, ...args),
    }
  : undefined;

const runner = createClaudeRunner({
  binary: values.binary,
  logger,
});

async function main() {
  // Scenario: Claude reviews files while a CI build failure notification
  // arrives via the channel. The prompt gives Claude multi-turn work
  // so the channel message arrives between turns.
  const prompt =
    "Read main.ts and package.json, then give a one-paragraph summary of each. " +
    "Also respond to any incoming channel notifications. Be brief.";
  console.log(`Prompt: ${prompt}`);
  console.log("---");

  const session = runner.start(prompt, {
    channelEnabled: true,
    skipPermissions: true,
    maxTurns: 10,
    timeout: 60_000,
    includePartialMessages: true,
    debugFile: values["debug-file"],
  });

  // Send the CI notification on the first system init message. session.send()
  // retries on ENOENT while the MCP server creates its socket, so we don't
  // need an explicit delay here.
  let sentNotification = false;

  for await (const msg of session.messages) {
    if (!sentNotification && msg.type === "system") {
      sentNotification = true;
      const notification: ChannelMessage = {
        content:
          "Build #1234 failed on main.\n" +
          "Stage: test\n" +
          "Failed: test_auth_flow (expected 200, got 401)\n" +
          'Commit: abc123f by @alice — "refactor: extract token validation"',
        sourceId: "ci-build-1234",
        sourceName: "GitHub Actions",
      };
      console.log(
        `[Channel] sending CI notification: ${notification.sourceId}`,
      );
      session
        .send(notification)
        .then(() => console.log("[Channel] send succeeded"))
        .catch((err) => console.error("[Channel] send error:", err));
    }

    switch (msg.type) {
      case "system": {
        const sys = msg.data as SystemStreamMessage;
        const toolNames = (sys.tools as Array<{name?: string}> ?? [])
          .map((t) => t.name)
          .filter(Boolean);
        console.log(
          `[System] model=${sys.model ?? "unknown"} session=${sys.session_id ?? ""} tools=${toolNames.length > 0 ? toolNames.join(",") : "none"}`,
        );
        break;
      }
      case "channel_reply": {
        const content = messageChannelReplyContent(msg);
        const dest = messageChannelReplyDestination(msg);
        console.log(`\n[Channel Reply -> ${dest}]\n${content}`);
        break;
      }
      case "stream_event": {
        const text = messageTextDelta(msg);
        if (text) process.stdout.write(text);
        break;
      }
      case "result": {
        const result = msg.data as ResultStreamMessage;
        console.log("\n---");
        console.log(`Cost:     $${(result.total_cost_usd ?? 0).toFixed(4)}`);
        console.log(`Duration: ${result.duration_ms ?? 0}ms`);
        console.log(`Session:  ${result.session_id ?? ""}`);
        break;
      }
    }
  }

  const result = await session.result;
  console.log(`\nFinal: ${result.text.slice(0, 100)}...`);
}

main().catch((err) => {
  console.error("error:", err);
  process.exit(1);
});
