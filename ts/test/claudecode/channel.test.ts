import { describe, it, expect, afterEach } from "vitest";
import {
  existsSync,
  readFileSync,
  writeFileSync,
  mkdtempSync,
  rmSync,
} from "node:fs";
import { createServer } from "node:net";
import { PassThrough } from "node:stream";
import { ChildProcess } from "node:child_process";
import { EventEmitter } from "node:events";
import {
  isChannelReply,
  channelReplyContent,
  channelReplyDestination,
  CHANNEL_REPLY_TOOL_NAME,
} from "../../src/claudecode/channel.js";
import { setupChannel, sendMessage } from "../../src/claudecode/channel.js";
import { createClaudeRunner } from "../../src/claudecode/runner.js";
import {
  messageChannelReplyContent,
  messageChannelReplyDestination,
} from "../../src/claudecode/accessors.js";
import type { SpawnFn } from "../../src/claudecode/options.js";
import type { AssistantStreamMessage } from "../../src/claudecode/types.js";
import type { Message } from "../../src/types.js";

// --- Channel reply detection tests ---

function makeAssistantMessage(
  blocks: Array<{ type: string; name?: string; input?: unknown }>,
): AssistantStreamMessage {
  return {
    type: "assistant",
    content: blocks.map((b) => ({
      type: b.type,
      name: b.name,
      input: b.input,
    })),
  };
}

describe("isChannelReply", () => {
  it("returns true for channel reply tool call", () => {
    const msg = makeAssistantMessage([
      {
        type: "tool_use",
        name: CHANNEL_REPLY_TOOL_NAME,
        input: { destination_id: "ci-123", content: "ok" },
      },
    ]);
    expect(isChannelReply(msg)).toBe(true);
  });

  it("returns false for other tool calls", () => {
    const msg = makeAssistantMessage([
      { type: "tool_use", name: "Read", input: { path: "/tmp" } },
    ]);
    expect(isChannelReply(msg)).toBe(false);
  });

  it("returns false for text-only messages", () => {
    const msg = makeAssistantMessage([{ type: "text" }]);
    expect(isChannelReply(msg)).toBe(false);
  });

  it("returns false for empty content", () => {
    const msg: AssistantStreamMessage = { type: "assistant", content: [] };
    expect(isChannelReply(msg)).toBe(false);
  });
});

describe("channelReplyContent", () => {
  it("extracts content from reply tool call", () => {
    const msg = makeAssistantMessage([
      {
        type: "tool_use",
        name: CHANNEL_REPLY_TOOL_NAME,
        input: { destination_id: "ci-123", content: "Build looks fine" },
      },
    ]);
    expect(channelReplyContent(msg)).toBe("Build looks fine");
  });

  it("returns undefined for non-reply messages", () => {
    const msg = makeAssistantMessage([{ type: "text" }]);
    expect(channelReplyContent(msg)).toBeUndefined();
  });
});

describe("channelReplyDestination", () => {
  it("extracts destination_id from reply tool call", () => {
    const msg = makeAssistantMessage([
      {
        type: "tool_use",
        name: CHANNEL_REPLY_TOOL_NAME,
        input: { destination_id: "ci-123", content: "ok" },
      },
    ]);
    expect(channelReplyDestination(msg)).toBe("ci-123");
  });

  it("returns undefined for non-reply messages", () => {
    const msg = makeAssistantMessage([{ type: "text" }]);
    expect(channelReplyDestination(msg)).toBeUndefined();
  });
});

// --- Message accessor tests ---

describe("messageChannelReplyContent", () => {
  it("extracts content from a ClaudeMessage", () => {
    const data = makeAssistantMessage([
      {
        type: "tool_use",
        name: CHANNEL_REPLY_TOOL_NAME,
        input: { destination_id: "ci-1", content: "Reply text" },
      },
    ]);
    const msg = { type: "channel_reply", raw: "{}", data } as Message;
    expect(messageChannelReplyContent(msg)).toBe("Reply text");
  });
});

describe("messageChannelReplyDestination", () => {
  it("extracts destination from a ClaudeMessage", () => {
    const data = makeAssistantMessage([
      {
        type: "tool_use",
        name: CHANNEL_REPLY_TOOL_NAME,
        input: { destination_id: "ci-1", content: "ok" },
      },
    ]);
    const msg = { type: "channel_reply", raw: "{}", data } as Message;
    expect(messageChannelReplyDestination(msg)).toBe("ci-1");
  });
});

// --- setupChannel tests ---

describe("setupChannel", () => {
  const cleanups: Array<() => void> = [];

  afterEach(() => {
    for (const fn of cleanups) {
      try {
        fn();
      } catch {
        // ignore
      }
    }
    cleanups.length = 0;
  });

  it("creates temp dir, socket path, and MCP config", () => {
    // Mock AGENTRUNNER_CHANNEL_BIN to avoid real binary resolution.
    const prev = process.env.AGENTRUNNER_CHANNEL_BIN;
    process.env.AGENTRUNNER_CHANNEL_BIN = "/usr/bin/fake-channel";
    try {
      const setup = setupChannel();
      cleanups.push(setup.cleanup);

      expect(setup.sockPath).toMatch(/\/tmp\/ar-ch-.+\/ch\.sock/);
      expect(existsSync(setup.mcpConfigPath)).toBe(true);

      const cfg = JSON.parse(readFileSync(setup.mcpConfigPath, "utf-8"));
      expect(cfg.mcpServers["agentrunner-channel"]).toBeDefined();
      expect(cfg.mcpServers["agentrunner-channel"].command).toBe(
        "/usr/bin/fake-channel",
      );
      expect(
        cfg.mcpServers["agentrunner-channel"].env.AGENTRUNNER_CHANNEL_SOCK,
      ).toBe(setup.sockPath);
    } finally {
      if (prev === undefined) delete process.env.AGENTRUNNER_CHANNEL_BIN;
      else process.env.AGENTRUNNER_CHANNEL_BIN = prev;
    }
  });

  it("merges user MCP config", () => {
    const prev = process.env.AGENTRUNNER_CHANNEL_BIN;
    process.env.AGENTRUNNER_CHANNEL_BIN = "/usr/bin/fake-channel";

    // Write a user config to a temp file.
    const userDir = mkdtempSync("/tmp/test-mcp-");
    cleanups.push(() => rmSync(userDir, { recursive: true, force: true }));

    const userCfgPath = `${userDir}/user-mcp.json`;
    writeFileSync(
      userCfgPath,
      JSON.stringify({
        mcpServers: {
          "my-server": {
            command: "/usr/bin/my-server",
            args: ["--port", "8080"],
          },
          "agentrunner-channel": { command: "/should-be-replaced" },
        },
      }),
    );

    try {
      const setup = setupChannel({ mcpConfig: userCfgPath });
      cleanups.push(setup.cleanup);

      const cfg = JSON.parse(readFileSync(setup.mcpConfigPath, "utf-8"));

      // Both servers present.
      expect(cfg.mcpServers["agentrunner-channel"].command).toBe(
        "/usr/bin/fake-channel",
      );
      expect(cfg.mcpServers["my-server"].command).toBe("/usr/bin/my-server");

      // User's agentrunner-channel was replaced, not kept.
      expect(cfg.mcpServers["agentrunner-channel"].command).not.toBe(
        "/should-be-replaced",
      );
    } finally {
      if (prev === undefined) delete process.env.AGENTRUNNER_CHANNEL_BIN;
      else process.env.AGENTRUNNER_CHANNEL_BIN = prev;
    }
  });

  it("cleanup removes temp dir", () => {
    const prev = process.env.AGENTRUNNER_CHANNEL_BIN;
    process.env.AGENTRUNNER_CHANNEL_BIN = "/usr/bin/fake-channel";
    try {
      const setup = setupChannel();
      const dir = setup.mcpConfigPath.replace(/\/mcp\.json$/, "");
      expect(existsSync(dir)).toBe(true);

      setup.cleanup();
      expect(existsSync(dir)).toBe(false);
    } finally {
      if (prev === undefined) delete process.env.AGENTRUNNER_CHANNEL_BIN;
      else process.env.AGENTRUNNER_CHANNEL_BIN = prev;
    }
  });
});

// --- sendMessage tests ---

describe("sendMessage", () => {
  it("sends NDJSON to Unix socket", async () => {
    const tmpDir = mkdtempSync("/tmp/test-send-");
    const sockPath = `${tmpDir}/test.sock`;

    const received: string[] = [];
    const server = createServer((conn) => {
      let data = "";
      conn.on("data", (chunk) => (data += chunk.toString()));
      conn.on("end", () => received.push(data));
    });

    await new Promise<void>((resolve) => server.listen(sockPath, resolve));

    try {
      await sendMessage(sockPath, {
        content: "Build failed",
        sourceId: "ci-123",
        sourceName: "GitHub Actions",
      });

      // Give the server time to process.
      await new Promise((r) => setTimeout(r, 50));

      expect(received).toHaveLength(1);
      const parsed = JSON.parse(received[0].trim());
      expect(parsed.content).toBe("Build failed");
      expect(parsed.source_id).toBe("ci-123");
      expect(parsed.source_name).toBe("GitHub Actions");
      expect(parsed.reply_to).toBeUndefined();
    } finally {
      server.close();
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("includes reply_to when set", async () => {
    const tmpDir = mkdtempSync("/tmp/test-send-");
    const sockPath = `${tmpDir}/test.sock`;

    const received: string[] = [];
    const server = createServer((conn) => {
      let data = "";
      conn.on("data", (chunk) => (data += chunk.toString()));
      conn.on("end", () => received.push(data));
    });

    await new Promise<void>((resolve) => server.listen(sockPath, resolve));

    try {
      await sendMessage(sockPath, {
        content: "Reply",
        sourceId: "src-1",
        sourceName: "test",
        replyTo: "orig-1",
      });

      await new Promise((r) => setTimeout(r, 50));

      const parsed = JSON.parse(received[0].trim());
      expect(parsed.reply_to).toBe("orig-1");
    } finally {
      server.close();
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});

// --- Runner integration tests ---

function mockSpawn(lines: string[], exitCode = 0): SpawnFn {
  return (_command, _args, _options) => {
    const stdout = new PassThrough();
    const stderr = new PassThrough();
    const proc = new EventEmitter() as ChildProcess;

    proc.stdout = stdout;
    proc.stderr = stderr;
    proc.stdin = null;
    proc.stdio = [null, stdout, stderr, null, null];
    proc.pid = 12345;
    proc.killed = false;
    proc.connected = false;
    proc.exitCode = null;
    proc.signalCode = null;
    proc.kill = () => {
      proc.exitCode = exitCode;
      return true;
    };

    setImmediate(() => {
      stderr.end();
      for (const line of lines) {
        stdout.write(line + "\n");
      }
      stdout.end();
      proc.exitCode = exitCode;
      proc.emit("close", exitCode, null);
    });

    return proc;
  };
}

describe("runner channel integration", () => {
  it("remaps channel reply messages to channel_reply type", async () => {
    const replyInput = JSON.stringify({
      destination_id: "ci-123",
      content: "Build analysis complete",
    });
    const lines = [
      `{"type":"system","subtype":"init","session_id":"sess-ch"}`,
      `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"${CHANNEL_REPLY_TOOL_NAME}","input":${replyInput}}]}}`,
      `{"type":"result","subtype":"success","result":"done","is_error":false,"total_cost_usd":0.01,"duration_ms":100,"usage":{"input_tokens":10,"output_tokens":5}}`,
    ];

    const runner = createClaudeRunner({ spawn: mockSpawn(lines) });
    const messages: Message[] = [];

    for await (const msg of runner.runStream("test")) {
      messages.push(msg);
    }

    expect(messages[1].type).toBe("channel_reply");
    // The data is still the original assistant message.
    expect(messages[1].data).toHaveProperty("type", "assistant");
  });

  it("non-reply assistant messages keep original type", async () => {
    const lines = [
      `{"type":"system","subtype":"init","session_id":"sess-ch2"}`,
      `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}`,
      `{"type":"result","subtype":"success","result":"Hello","is_error":false,"total_cost_usd":0.01,"duration_ms":100,"usage":{"input_tokens":10,"output_tokens":5}}`,
    ];

    const runner = createClaudeRunner({ spawn: mockSpawn(lines) });
    const messages: Message[] = [];

    for await (const msg of runner.runStream("test")) {
      messages.push(msg);
    }

    expect(messages[1].type).toBe("assistant");
  });

  it("send rejects when channels not enabled", async () => {
    const lines = [
      `{"type":"system","subtype":"init","session_id":"sess-no-ch"}`,
      `{"type":"result","subtype":"success","result":"done","is_error":false,"total_cost_usd":0.01,"duration_ms":100,"usage":{"input_tokens":10,"output_tokens":5}}`,
    ];

    const runner = createClaudeRunner({ spawn: mockSpawn(lines) });
    const session = runner.start("test");

    await expect(session.send({ content: "hi" })).rejects.toThrow(
      /channelEnabled/,
    );

    // Drain to avoid resource leak.
    for await (const _msg of session.messages) {
      // drain
    }
  });
});
