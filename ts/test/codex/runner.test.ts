import { describe, it, expect } from "vitest";
import { PassThrough } from "node:stream";
import { ChildProcess } from "node:child_process";
import { EventEmitter } from "node:events";
import { createCodexRunner } from "../../src/codex/runner.js";
import type { SpawnFn } from "../../src/codex/options.js";
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
      setImmediate(() => proc.emit("close", null, "SIGTERM"));
      return true;
    };

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
  `{"type":"thread.started","thread_id":"thread-1"}`,
  `{"type":"item.started","item":{"id":"item-1","type":"agent_message","text":"","status":"in_progress"}}`,
  `{"type":"item.completed","item":{"id":"item-1","type":"agent_message","text":"Hello world","status":"completed"}}`,
  `{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":50}}`,
];

const toolUseLine = `{"type":"item.started","item":{"id":"item-2","type":"command_execution","command":"ls -la","status":"in_progress"}}`;
const toolResultLine = `{"type":"item.completed","item":{"id":"item-2","type":"command_execution","command":"ls -la","aggregated_output":"file1.txt\\nfile2.txt","exit_code":0,"status":"completed"}}`;

const toolUseLines = [
  `{"type":"thread.started","thread_id":"thread-2"}`,
  toolUseLine,
  toolResultLine,
  `{"type":"item.completed","item":{"id":"item-3","type":"agent_message","text":"Listed files","status":"completed"}}`,
  `{"type":"turn.completed","usage":{"input_tokens":200,"output_tokens":80}}`,
];

describe("run", () => {
  it("happy path", async () => {
    const runner = createCodexRunner({ spawn: mockSpawn(happyLines) });
    const result = await runner.run("say hello");

    expect(result.text).toBe("Hello world");
    expect(result.sessionId).toBe("thread-1");
    expect(result.isError).toBe(false);
    expect(result.usage.inputTokens).toBe(100);
    expect(result.usage.outputTokens).toBe(50);
    expect(result.usage.cacheReadInputTokens).toBe(20);
    expect(result.costUSD).toBe(0);
  });

  it("with tool use", async () => {
    const runner = createCodexRunner({ spawn: mockSpawn(toolUseLines) });
    const result = await runner.run("list files");

    expect(result.text).toBe("Listed files");
    expect(result.sessionId).toBe("thread-2");
    expect(result.isError).toBe(false);
    expect(result.usage.inputTokens).toBe(200);
    expect(result.usage.outputTokens).toBe(80);
  });

  it("error result", async () => {
    const lines = [
      `{"type":"thread.started","thread_id":"thread-err"}`,
      `{"type":"error","message":"Something went wrong"}`,
    ];
    const runner = createCodexRunner({ spawn: mockSpawn(lines) });
    const result = await runner.run("fail please");

    expect(result.isError).toBe(true);
    expect(result.text).toBe("Something went wrong");
    expect(result.sessionId).toBe("thread-err");
  });

  it("turn.failed result", async () => {
    const lines = [
      `{"type":"thread.started","thread_id":"thread-fail"}`,
      `{"type":"turn.failed","error":{"message":"Rate limit exceeded"}}`,
    ];
    const runner = createCodexRunner({ spawn: mockSpawn(lines) });
    const result = await runner.run("hello");

    expect(result.isError).toBe(true);
    expect(result.text).toBe("Rate limit exceeded");
  });

  it("no result throws NoResultError", async () => {
    const lines = [`{"type":"thread.started","thread_id":"thread-x"}`];
    const runner = createCodexRunner({ spawn: mockSpawn(lines) });
    await expect(runner.run("hello")).rejects.toThrow(NoResultError);
  });

  it("non-zero exit throws NonZeroExitError", async () => {
    const runner = createCodexRunner({
      spawn: mockSpawn([], 1, "fatal error from codex"),
    });
    await expect(runner.run("hello")).rejects.toThrow(NonZeroExitError);
    try {
      await runner.run("hello");
    } catch (err) {
      expect(err).toBeInstanceOf(NonZeroExitError);
      expect((err as NonZeroExitError).message).toContain(
        "fatal error from codex",
      );
    }
  });

  it("timeout throws TimeoutError", async () => {
    const runner = createCodexRunner({ spawn: slowSpawn() });
    await expect(runner.run("hello", { timeout: 50 })).rejects.toThrow(
      TimeoutError,
    );
  });

  it("cancellation throws CancelledError", async () => {
    const controller = new AbortController();
    const runner = createCodexRunner({ spawn: slowSpawn() });

    setTimeout(() => controller.abort(), 50);

    await expect(
      runner.run("hello", { signal: controller.signal }),
    ).rejects.toThrow(CancelledError);
  });
});

describe("runStream", () => {
  it("happy path yields correct messages", async () => {
    const runner = createCodexRunner({ spawn: mockSpawn(happyLines) });
    const messages: Message[] = [];

    for await (const msg of runner.runStream("say hello")) {
      messages.push(msg);
    }

    expect(messages).toHaveLength(4);
    expect(messages[0].type).toBe("system"); // thread.started
    expect(messages[1].type).toBe("assistant"); // item.started agent_message
    expect(messages[2].type).toBe("assistant"); // item.completed agent_message
    expect(messages[3].type).toBe("result"); // turn.completed
  });

  it("tool use messages have correct types", async () => {
    const runner = createCodexRunner({ spawn: mockSpawn(toolUseLines) });
    const messages: Message[] = [];

    for await (const msg of runner.runStream("list files")) {
      messages.push(msg);
    }

    expect(messages).toHaveLength(5);
    expect(messages[0].type).toBe("system"); // thread.started
    expect(messages[1].type).toBe("tool_use"); // item.started command_execution
    expect(messages[2].type).toBe("tool_result"); // item.completed command_execution
    expect(messages[3].type).toBe("assistant"); // item.completed agent_message
    expect(messages[4].type).toBe("result"); // turn.completed
  });

  it("timeout throws TimeoutError", async () => {
    const runner = createCodexRunner({ spawn: slowSpawn() });

    const consume = async () => {
      for await (const _msg of runner.runStream("hello", { timeout: 50 })) {
        // drain
      }
    };

    await expect(consume()).rejects.toThrow(TimeoutError);
  });

  it("cancellation throws CancelledError", async () => {
    const controller = new AbortController();
    const runner = createCodexRunner({ spawn: slowSpawn() });

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
    const runner = createCodexRunner({
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
    const runner = createCodexRunner({ spawn: mockSpawn(happyLines) });
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
    const runner = createCodexRunner({ spawn: mockSpawn(happyLines) });

    for await (const msg of runner.runStream("test raw")) {
      expect(msg.raw.length).toBeGreaterThan(0);
    }
  });
});

describe("start (Session)", () => {
  it("happy path: messages and result", async () => {
    const runner = createCodexRunner({ spawn: mockSpawn(happyLines) });
    const session = runner.start("say hello");

    const messages: Message[] = [];
    for await (const msg of session.messages) {
      messages.push(msg);
    }

    const result = await session.result;

    expect(messages).toHaveLength(4);
    expect(messages[0].type).toBe("system");
    expect(messages[messages.length - 1].type).toBe("result");

    expect(result.text).toBe("Hello world");
    expect(result.sessionId).toBe("thread-1");
  });

  it("abort terminates the process", async () => {
    const runner = createCodexRunner({ spawn: slowSpawn() });
    const session = runner.start("long task", { timeout: 5000 });

    setTimeout(() => session.abort(), 50);

    const messages: Message[] = [];
    for await (const msg of session.messages) {
      messages.push(msg);
    }

    await expect(session.result).rejects.toThrow();
  });

  it("send rejects with NotSupportedError", async () => {
    const runner = createCodexRunner({ spawn: mockSpawn(happyLines) });
    const session = runner.start("hello");

    await expect(session.send("test")).rejects.toThrow(NotSupportedError);

    // Drain to avoid resource leak.
    for await (const _msg of session.messages) {
      // drain
    }
  });

  it("streaming messages with onMessage callback", async () => {
    const runner = createCodexRunner({ spawn: mockSpawn(happyLines) });
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
