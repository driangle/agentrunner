import { describe, it, expect } from "vitest";
import { PassThrough } from "node:stream";
import { ChildProcess } from "node:child_process";
import { EventEmitter } from "node:events";
import { createClaudeRunner } from "../../src/claudecode/runner.js";
import type { SpawnFn } from "../../src/claudecode/options.js";
import type { Message } from "../../src/types.js";
import {
  NoResultError,
  NonZeroExitError,
  TimeoutError,
  CancelledError,
  NotSupportedError,
} from "../../src/errors.js";

/**
 * Create a mock spawn function that writes canned lines to stdout
 * and exits with the given code.
 */
function mockSpawn(
  lines: string[],
  exitCode = 0,
  stderrText = "",
  delay = 0,
): SpawnFn {
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

    const emit = () => {
      if (stderrText) {
        stderr.write(stderrText);
      }
      stderr.end();

      for (const line of lines) {
        stdout.write(line + "\n");
      }
      stdout.end();

      proc.exitCode = exitCode;
      proc.emit("close", exitCode, null);
    };

    if (delay > 0) {
      setTimeout(emit, delay);
    } else {
      // Use setImmediate to allow the caller to set up listeners.
      setImmediate(emit);
    }

    return proc;
  };
}

/** A mock spawn that never completes (for timeout/cancel tests). */
function slowSpawn(signal?: AbortSignal): SpawnFn {
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
      proc.killed = true;
      proc.exitCode = null;
      stdout.end();
      stderr.end();
      // Emit close on next tick so readline can finish consuming first.
      setImmediate(() => proc.emit("close", null, "SIGTERM"));
      return true;
    };

    // Listen for abort to simulate process termination.
    const abortSignal = _options.signal ?? signal;
    if (abortSignal) {
      const onAbort = () => {
        proc.kill();
      };
      abortSignal.addEventListener("abort", onAbort, { once: true });
    }

    return proc;
  };
}

const happyLines = [
  `{"type":"system","subtype":"init","session_id":"sess-1","model":"claude-sonnet-4-6"}`,
  `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","content":[{"type":"text","text":"Hello world"}]}}`,
  `{"type":"result","subtype":"success","result":"Hello world","is_error":false,"total_cost_usd":0.05,"duration_ms":1234,"duration_api_ms":1100,"num_turns":2,"session_id":"sess-1","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":20}}`,
];

const streamMultiLines = [
  `{"type":"system","subtype":"init","session_id":"sess-s1","model":"claude-sonnet-4-6"}`,
  `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}},"session_id":"sess-s1"}`,
  `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}},"session_id":"sess-s1"}`,
  `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","content":[{"type":"text","text":"Hello world"}]}}`,
  `{"type":"result","subtype":"success","result":"Hello world","is_error":false,"total_cost_usd":0.05,"duration_ms":500,"session_id":"sess-s1","usage":{"input_tokens":100,"output_tokens":50}}`,
];

describe("run", () => {
  it("happy path", async () => {
    const runner = createClaudeRunner({ spawn: mockSpawn(happyLines) });
    const result = await runner.run("say hello");

    expect(result.text).toBe("Hello world");
    expect(result.sessionId).toBe("sess-1");
    expect(result.costUSD).toBe(0.05);
    expect(result.durationMs).toBe(1234);
    expect(result.isError).toBe(false);
    expect(result.usage.inputTokens).toBe(100);
    expect(result.usage.outputTokens).toBe(50);
    expect(result.usage.cacheCreationInputTokens).toBe(10);
    expect(result.usage.cacheReadInputTokens).toBe(20);
  });

  it("error result", async () => {
    const lines = [
      `{"type":"result","subtype":"error","result":"Something failed","is_error":true,"session_id":"sess-err","usage":{"input_tokens":10,"output_tokens":5}}`,
    ];
    const runner = createClaudeRunner({ spawn: mockSpawn(lines) });
    const result = await runner.run("fail please");

    expect(result.isError).toBe(true);
    expect(result.text).toBe("Something failed");
    expect(result.sessionId).toBe("sess-err");
  });

  it("no result throws NoResultError", async () => {
    const lines = [`{"type":"system","subtype":"init","session_id":"sess-x"}`];
    const runner = createClaudeRunner({ spawn: mockSpawn(lines) });
    await expect(runner.run("hello")).rejects.toThrow(NoResultError);
  });

  it("non-zero exit throws NonZeroExitError", async () => {
    const runner = createClaudeRunner({
      spawn: mockSpawn([], 1, "fatal error from claude"),
    });
    await expect(runner.run("hello")).rejects.toThrow(NonZeroExitError);
    try {
      await runner.run("hello");
    } catch (err) {
      expect(err).toBeInstanceOf(NonZeroExitError);
      expect((err as NonZeroExitError).message).toContain(
        "fatal error from claude",
      );
    }
  });

  it("timeout throws TimeoutError", async () => {
    const runner = createClaudeRunner({ spawn: slowSpawn() });
    await expect(runner.run("hello", { timeout: 50 })).rejects.toThrow(
      TimeoutError,
    );
  });

  it("cancellation throws CancelledError", async () => {
    const controller = new AbortController();
    const runner = createClaudeRunner({ spawn: slowSpawn() });

    setTimeout(() => controller.abort(), 50);

    await expect(
      runner.run("hello", { signal: controller.signal }),
    ).rejects.toThrow(CancelledError);
  });

  it("session ID falls back to init message", async () => {
    const lines = [
      `{"type":"system","subtype":"init","session_id":"sess-from-init","model":"claude-sonnet-4-6"}`,
      `{"type":"result","subtype":"success","result":"done","is_error":false,"total_cost_usd":0.01,"duration_ms":100,"usage":{"input_tokens":10,"output_tokens":5}}`,
    ];
    const runner = createClaudeRunner({ spawn: mockSpawn(lines) });
    const result = await runner.run("hello");

    expect(result.sessionId).toBe("sess-from-init");
  });
});

describe("runStream", () => {
  it("happy path yields correct messages", async () => {
    const runner = createClaudeRunner({ spawn: mockSpawn(streamMultiLines) });
    const messages: Message[] = [];

    for await (const msg of runner.runStream("say hello")) {
      messages.push(msg);
    }

    expect(messages).toHaveLength(5);
    expect(messages[0].type).toBe("system");
    expect(messages[1].type).toBe("stream_event");
    expect(messages[2].type).toBe("stream_event");
    expect(messages[3].type).toBe("assistant");
    expect(messages[4].type).toBe("result");
  });

  it("message ordering: system first, result last", async () => {
    const runner = createClaudeRunner({ spawn: mockSpawn(streamMultiLines) });
    const types: string[] = [];

    for await (const msg of runner.runStream("test ordering")) {
      types.push(msg.type);
    }

    expect(types[0]).toBe("system");
    expect(types[types.length - 1]).toBe("result");
  });

  it("channels close even without result message", async () => {
    const lines = [
      `{"type":"system","subtype":"init","session_id":"sess-nr"}`,
      `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","content":[{"type":"text","text":"partial"}]}}`,
    ];
    const runner = createClaudeRunner({ spawn: mockSpawn(lines) });
    const messages: Message[] = [];

    // runStream should throw NoResultError after draining
    try {
      for await (const msg of runner.runStream("partial")) {
        messages.push(msg);
      }
    } catch {
      // Expected — NoResultError after stream ends
    }

    expect(messages).toHaveLength(2);
  });

  it("timeout throws TimeoutError", async () => {
    const runner = createClaudeRunner({ spawn: slowSpawn() });

    const consume = async () => {
      for await (const _msg of runner.runStream("hello", { timeout: 50 })) {
        // drain
      }
    };

    await expect(consume()).rejects.toThrow(TimeoutError);
  });

  it("cancellation throws CancelledError", async () => {
    const controller = new AbortController();
    const runner = createClaudeRunner({ spawn: slowSpawn() });

    setTimeout(() => controller.abort(), 50);

    const consume = async () => {
      for await (const _msg of runner.runStream("hello", {
        signal: controller.signal,
      })) {
        // drain
      }
    };

    await expect(consume()).rejects.toThrow(CancelledError);
  });

  it("non-zero exit throws NonZeroExitError", async () => {
    const runner = createClaudeRunner({
      spawn: mockSpawn([], 1, "fatal error"),
    });

    const consume = async () => {
      for await (const _msg of runner.runStream("hello")) {
        // drain
      }
    };

    await expect(consume()).rejects.toThrow(NonZeroExitError);
  });

  it("onMessage callback receives all messages", async () => {
    const runner = createClaudeRunner({ spawn: mockSpawn(streamMultiLines) });
    const callbackMessages: Message[] = [];
    const channelMessages: Message[] = [];

    for await (const msg of runner.runStream("test callback", {
      onMessage: (m) => callbackMessages.push(m),
    })) {
      channelMessages.push(msg);
    }

    expect(callbackMessages).toHaveLength(channelMessages.length);
    for (let i = 0; i < callbackMessages.length; i++) {
      expect(callbackMessages[i].type).toBe(channelMessages[i].type);
    }
  });

  it("raw JSON is populated on all messages", async () => {
    const runner = createClaudeRunner({ spawn: mockSpawn(streamMultiLines) });

    for await (const msg of runner.runStream("test raw")) {
      expect(msg.raw.length).toBeGreaterThan(0);
    }
  });
});

describe("start (Session)", () => {
  it("happy path: messages and result", async () => {
    const runner = createClaudeRunner({ spawn: mockSpawn(happyLines) });
    const session = runner.start("say hello");

    const messages: Message[] = [];
    for await (const msg of session.messages) {
      messages.push(msg);
    }

    const result = await session.result;

    expect(messages).toHaveLength(3);
    expect(messages[0].type).toBe("system");
    expect(messages[messages.length - 1].type).toBe("result");

    expect(result.text).toBe("Hello world");
    expect(result.sessionId).toBe("sess-1");
    expect(result.costUSD).toBe(0.05);
  });

  it("abort terminates the process", async () => {
    const runner = createClaudeRunner({ spawn: slowSpawn() });
    const session = runner.start("long task", { timeout: 5000 });

    // Abort after a brief delay.
    setTimeout(() => session.abort(), 50);

    const messages: Message[] = [];
    for await (const msg of session.messages) {
      messages.push(msg);
    }

    // Result should reject with CancelledError or TimeoutError.
    await expect(session.result).rejects.toThrow();
  });

  it("send throws NotSupportedError", () => {
    const runner = createClaudeRunner({ spawn: mockSpawn(happyLines) });
    const session = runner.start("hello");

    expect(() => session.send("test")).toThrow(NotSupportedError);

    // Drain to avoid resource leak.
    (async () => {
      for await (const _msg of session.messages) {
        // drain
      }
    })();
  });

  it("session ID falls back to init message", async () => {
    const lines = [
      `{"type":"system","subtype":"init","session_id":"sess-from-init","model":"claude-sonnet-4-6"}`,
      `{"type":"result","subtype":"success","result":"done","is_error":false,"total_cost_usd":0.01,"duration_ms":100,"usage":{"input_tokens":10,"output_tokens":5}}`,
    ];
    const runner = createClaudeRunner({ spawn: mockSpawn(lines) });
    const session = runner.start("hello");

    for await (const _msg of session.messages) {
      // drain
    }

    const result = await session.result;
    expect(result.sessionId).toBe("sess-from-init");
  });

  it("streaming messages with onMessage callback", async () => {
    const runner = createClaudeRunner({ spawn: mockSpawn(streamMultiLines) });
    const callbackMessages: Message[] = [];

    const session = runner.start("test callback", {
      onMessage: (m) => callbackMessages.push(m),
    });

    const messages: Message[] = [];
    for await (const msg of session.messages) {
      messages.push(msg);
    }

    expect(callbackMessages).toHaveLength(messages.length);
    for (let i = 0; i < callbackMessages.length; i++) {
      expect(callbackMessages[i].type).toBe(messages[i].type);
    }
  });
});
