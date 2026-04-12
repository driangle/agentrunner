import { createConnection } from "node:net";
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { resolveChannelBinary } from "../channel/binary.js";
import type { AssistantStreamMessage, ContentBlock } from "./types.js";

/** [Experimental] Message sent to or received from the channel. */
export interface ChannelMessage {
  /** Message body that Claude reads. */
  content: string;
  /** Caller-defined correlation ID. */
  sourceId: string;
  /** Human-readable origin (e.g. "github-webhook"). */
  sourceName: string;
  /** Optional reference to a prior message's sourceId. */
  replyTo?: string;
}

/** MCP tool name for channel replies (prefixed by Claude CLI). */
export const CHANNEL_REPLY_TOOL_NAME = "mcp__agentrunner-channel__reply";

interface MCPServerConfig {
  command: string;
  args?: string[];
  env?: Record<string, string>;
}

interface MCPConfig {
  mcpServers: Record<string, MCPServerConfig>;
}

/** Artifacts created by setupChannel. */
export interface ChannelSetup {
  sockPath: string;
  mcpConfigPath: string;
  cleanup: () => void;
}

/** Options for configuring the channel MCP server. */
export interface ChannelOptions {
  /** Existing MCP config file path to merge with. */
  mcpConfig?: string;
  /** File path for channel server logs. */
  logFile?: string;
  /** Log level for the channel server. */
  logLevel?: "debug" | "info" | "warn" | "error";
}

/**
 * [Experimental] Prepare the channel infrastructure for a Claude CLI invocation.
 * Resolves the binary, creates a temp directory with Unix socket path
 * and MCP config, and optionally merges with the user's existing config.
 */
export function setupChannel(options: ChannelOptions = {}): ChannelSetup {
  const binPath = resolveChannelBinary();

  // Use /tmp for short socket paths (macOS has a 104-char limit).
  const tmpDir = mkdtempSync("/tmp/ar-ch-");
  const sockPath = join(tmpDir, "ch.sock");

  const env: Record<string, string> = {
    AGENTRUNNER_CHANNEL_SOCK: sockPath,
  };
  if (options.logFile) {
    env.AGENTRUNNER_CHANNEL_LOG = options.logFile;
  }
  if (options.logLevel) {
    env.AGENTRUNNER_CHANNEL_LOG_LEVEL = options.logLevel;
  }

  const cfg: MCPConfig = {
    mcpServers: {
      "agentrunner-channel": {
        command: binPath,
        env,
      },
    },
  };

  // Merge with user's MCP config if provided.
  if (options.mcpConfig) {
    const userData = readFileSync(options.mcpConfig, "utf-8");
    const userCfg = JSON.parse(userData) as MCPConfig;
    for (const [k, v] of Object.entries(userCfg.mcpServers ?? {})) {
      if (k !== "agentrunner-channel") {
        cfg.mcpServers[k] = v;
      }
    }
  }

  const cfgPath = join(tmpDir, "mcp.json");
  writeFileSync(cfgPath, JSON.stringify(cfg), { mode: 0o600 });

  return {
    sockPath,
    mcpConfigPath: cfgPath,
    cleanup: () => rmSync(tmpDir, { recursive: true, force: true }),
  };
}

/** [Experimental] Send a ChannelMessage to the channel server via Unix socket. */
export async function sendMessage(
  sockPath: string,
  msg: ChannelMessage,
): Promise<void> {
  const data = JSON.stringify({
    content: msg.content,
    source_id: msg.sourceId,
    source_name: msg.sourceName,
    ...(msg.replyTo != null ? { reply_to: msg.replyTo } : {}),
  });

  // Retry on ENOENT — the MCP server may not have created the socket yet.
  const maxRetries = 10;
  const retryDelayMs = 500;
  for (let attempt = 0; ; attempt++) {
    try {
      await writeToSocket(sockPath, data + "\n");
      return;
    } catch (err: unknown) {
      const code =
        err instanceof Error ? (err as NodeJS.ErrnoException).code : undefined;
      const isENOENT = code === "ENOENT";
      if (isENOENT && attempt < maxRetries - 1) {
        await new Promise((r) => setTimeout(r, retryDelayMs));
        continue;
      }
      throw err;
    }
  }
}

function writeToSocket(sockPath: string, data: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const socket = createConnection(sockPath, () => {
      socket.end(data, () => resolve());
    });
    socket.on("error", reject);
  });
}

/** Check whether an assistant message contains a channel reply tool call. */
export function isChannelReply(data: AssistantStreamMessage): boolean {
  return data.content.some(
    (b) => b.type === "tool_use" && b.name === CHANNEL_REPLY_TOOL_NAME,
  );
}

/** Find the reply tool_use content block, if present. */
function findReplyBlock(
  data: AssistantStreamMessage,
): ContentBlock | undefined {
  return data.content.find(
    (b) => b.type === "tool_use" && b.name === CHANNEL_REPLY_TOOL_NAME,
  );
}

/** Extract the reply content from a channel reply tool call. */
export function channelReplyContent(
  data: AssistantStreamMessage,
): string | undefined {
  const block = findReplyBlock(data);
  if (!block?.input) return undefined;
  const args = block.input as { content?: string };
  return args.content;
}

/** Extract the destination ID from a channel reply tool call. */
export function channelReplyDestination(
  data: AssistantStreamMessage,
): string | undefined {
  const block = findReplyBlock(data);
  if (!block?.input) return undefined;
  const args = block.input as { destination_id?: string };
  return args.destination_id;
}
