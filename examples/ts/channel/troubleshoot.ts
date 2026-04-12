// [Experimental] Minimal channel troubleshooting example.
//
// KNOWN LIMITATION: The channels feature in Claude Code v2.1.84 is gated
// behind a server-side feature flag ("tengu_harbor"). In -p (print) mode,
// the --dangerously-load-development-channels bypass that works in
// interactive mode is NOT applied. This means channels will silently fail
// in -p mode unless the feature flag is enabled on your account.
//
// Symptoms: the channel MCP server starts, handshakes, and successfully
// forwards your message (visible in channel logs), but Claude never sees
// the notification in its conversation context.
//
// To verify this is the cause: run Claude interactively (without -p) with
// the same --channels and --mcp-config flags and send a message via nc.
// If that works but this example doesn't, the feature flag is off.
//
// DEBUGGING INSIGHTS (from prior investigation):
//
//   1. Claude Code in -p mode does NOT wait for external events. It runs
//      through its turns and exits. Send the notification while Claude is
//      blocked inside a tool call.
//   2. Do NOT restrict allowedTools — the reply tool is dynamically loaded
//      via ToolSearchTool and a static allowedTools list blocks that.
//   3. Trigger the send when you observe the first assistant tool_use
//      block, not on the system init message.
//
// Run:
//   npx tsx troubleshoot.ts
//   npx tsx troubleshoot.ts --debug-file ./debug.log
//   npx tsx troubleshoot.ts --sleep-seconds 20
//   npx tsx troubleshoot.ts --send-delay-ms 1000

import { parseArgs } from "node:util";
import {
  createClaudeRunner,
  messageText,
  messageToolName,
  messageChannelReplyContent,
  messageChannelReplyDestination,
} from "agentrunner/claudecode";
import type {
  StreamEventStreamMessage,
} from "agentrunner/claudecode";
import type { ChannelMessage } from "agentrunner/channel";

const { values } = parseArgs({
  options: {
    binary: { type: "string", default: "claude" },
    "debug-file": { type: "string" },
    "sleep-seconds": { type: "string", default: "20" },
    "send-delay-ms": { type: "string", default: "1000" },
  },
});

const sleepSeconds = Number(values["sleep-seconds"]);
const sendDelayMs = Number(values["send-delay-ms"]);

const ts = () => new Date().toISOString();
const log = (msg: string) => console.log(`${ts()} ${msg}`);
const logErr = (msg: string, ...args: unknown[]) =>
  console.error(`${ts()} ${msg}`, ...args);

const logger = {
  debug: (msg: string, ...args: unknown[]) =>
    console.error(`${ts()} [debug]`, msg, ...args),
  error: (msg: string, ...args: unknown[]) =>
    console.error(`${ts()} [error]`, msg, ...args),
};

const runner = createClaudeRunner({ binary: values.binary, logger });

const NOTIFICATION: ChannelMessage = {
  content: "Please ACK this test message.",
  sourceId: "tshoot-1",
  sourceName: "troubleshoot",
};

async function main() {
  // Long-running Bash sleep keeps Claude busy mid-turn so the notification
  // arrives during the tool call, not during reasoning. The phrasing is
  // intentionally loose (matching the working main.ts) so the prompt does
  // not constrain how Claude handles the notification.
  const prompt =
    `Run the shell command \`sleep ${sleepSeconds}\` using the Bash tool, ` +
    "then briefly summarize what you did. " +
    "Also respond to any incoming channel notifications by replying via " +
    "the channel.";

  log(`sleep-seconds: ${sleepSeconds}`);
  log(`send-delay-ms: ${sendDelayMs}`);
  log("---");

  // NOTE: do NOT pass allowedTools — the channel reply tool is registered
  // dynamically by Claude's ToolSearchTool only AFTER a channel notification
  // arrives. A static allowedTools list blocks that dynamic registration
  // and the reply tool never becomes callable. Mirror main.ts: minimal opts.
  const session = runner.start(prompt, {
    channelEnabled: true,
    skipPermissions: true,
    maxTurns: 10,
    timeout: 120_000,
    includePartialMessages: true,
    debugFile: values["debug-file"],
    channelLogLevel: "debug",
  });

  // Send the notification ONLY after we observe the first assistant tool_use
  // block. That guarantees Claude has dispatched a tool and is blocked on
  // its result — the only window where notifications are reliably folded
  // into the next turn.
  let sent = false;
  const sendNotification = async (trigger: string) => {
    if (sent) return;
    sent = true;
    if (sendDelayMs > 0) {
      log(`[Channel] (${trigger}) waiting ${sendDelayMs}ms before send...`);
      await new Promise((r) => setTimeout(r, sendDelayMs));
    }
    log(`[Channel] sending source_id=${NOTIFICATION.sourceId}`);
    try {
      await session.send(NOTIFICATION);
      log("[Channel] send succeeded");
    } catch (err) {
      logErr("[Channel] send error:", err);
    }
  };

  for await (const msg of session.messages) {
    switch (msg.type) {
      case "system": {
        log("[system]");
        break;
      }
      case "assistant": {
        const text = messageText(msg);
        const tool = messageToolName(msg);
        log(
          `[assistant] tool=${tool ?? "-"} text=${text ? JSON.stringify(text.slice(0, 80)) : "-"}`,
        );
        if (tool) void sendNotification(`assistant-tool=${tool}`);
        break;
      }
      case "stream_event": {
        // With includePartialMessages, the consolidated "assistant" message
        // arrives only at end of turn. The earliest signal that Claude has
        // started a tool call is content_block_start with type=tool_use.
        const sev = msg.data as StreamEventStreamMessage;
        const ev = sev.event;
        if (
          ev?.type === "content_block_start" &&
          ev.content_block?.type === "tool_use"
        ) {
          const name = ev.content_block.name ?? "?";
          log(`[stream_event] tool_use start: ${name}`);
          void sendNotification(`stream-tool=${name}`);
        }
        break;
      }
      case "user": {
        log(`[user/tool-result]`);
        break;
      }
      case "channel_reply": {
        const content = messageChannelReplyContent(msg);
        const dest = messageChannelReplyDestination(msg);
        log(`[channel_reply] -> ${dest}: ${content}`);
        break;
      }
      case "result": {
        log("[result]");
        break;
      }
      default: {
        log(`[${msg.type}]`);
      }
    }
  }

  const result = await session.result;
  log("---");
  log(`final text: ${result.text}`);
  log(`is_error: ${result.isError}`);
  log(`session: ${result.sessionId}`);
}

main().catch((err) => {
  logErr("error:", err);
  process.exit(1);
});
