// This example demonstrates two-way channel communication with Claude Code.
// It starts a session with channels enabled, sends a CI notification to Claude
// via the channel, and prints any channel replies from the stream.
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
  });

  // Wait for the system init message before sending — that confirms
  // the MCP server is connected and the socket is ready.
  let sentNotification = false;

  for await (const msg of session.messages) {
    // Send a CI notification once the system init arrives.
    if (!sentNotification && msg.type === "system") {
      sentNotification = true;
      // Longer delay to let the MCP handshake complete after system init.
      setTimeout(async () => {
        const notification: ChannelMessage = {
          content:
            "Build #1234 failed on main.\n" +
            "Stage: test\n" +
            'Failed: test_auth_flow (expected 200, got 401)\n' +
            'Commit: abc123f by @alice — "refactor: extract token validation"',
          sourceId: "ci-build-1234",
          sourceName: "GitHub Actions",
        };
        console.log(`[Channel] sending CI notification: ${notification.sourceId}`);
        try {
          await session.send(notification);
          console.log("[Channel] send succeeded");
        } catch (err) {
          console.error("[Channel] send error:", err);
        }
      }, 5000);
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
