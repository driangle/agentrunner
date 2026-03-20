import { describe, it, expect } from "vitest";
import { createOllamaRunner } from "../../src/ollama/runner.js";
import type { FetchFn } from "../../src/ollama/options.js";
import type { ChatResponse } from "../../src/ollama/types.js";
import type { Message } from "../../src/types.js";
import {
  NotFoundError,
  HttpError,
  TimeoutError,
  CancelledError,
  ParseError,
  NoResultError,
  NotSupportedError,
} from "../../src/errors.js";

/** Build a JSON line for a non-final chat response chunk. */
function streamLine(content: string, thinking = ""): string {
  const resp: ChatResponse = {
    model: "llama3",
    created_at: "2026-01-01T00:00:00Z",
    message: { role: "assistant", content, ...(thinking ? { thinking } : {}) },
    done: false,
  };
  return JSON.stringify(resp);
}

/** Build a JSON line for the final (done=true) chat response. */
function doneLine(
  content: string,
  totalDuration: number,
  promptEvalCount: number,
  evalCount: number,
): string {
  const resp: ChatResponse = {
    model: "llama3",
    created_at: "2026-01-01T00:00:00Z",
    message: { role: "assistant", content },
    done: true,
    done_reason: "stop",
    total_duration: totalDuration,
    prompt_eval_count: promptEvalCount,
    eval_count: evalCount,
  };
  return JSON.stringify(resp);
}

/** Build a streaming ndjson response body from lines. */
function ndjsonBody(lines: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  const text = lines.map((l) => l + "\n").join("");
  return new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode(text));
      controller.close();
    },
  });
}

const happyLines = [
  streamLine("Hello"),
  streamLine(" world"),
  doneLine("", 2_000_000_000, 100, 50), // 2s = 2000ms
];

/** Create a mock fetch that returns the given lines as ndjson. */
function mockFetch(lines: string[]): FetchFn {
  return async (_url, _init) => {
    return new Response(ndjsonBody(lines), {
      status: 200,
      headers: { "Content-Type": "application/x-ndjson" },
    });
  };
}

/** Create a mock fetch that captures the request body. */
function capturingFetch(
  lines: string[],
  captured: { body?: unknown },
): FetchFn {
  return async (_url, init) => {
    if (init?.body) {
      captured.body = JSON.parse(init.body as string);
    }
    return new Response(ndjsonBody(lines), {
      status: 200,
      headers: { "Content-Type": "application/x-ndjson" },
    });
  };
}

/** Create a mock fetch that returns a specific HTTP status. */
function statusFetch(status: number): FetchFn {
  return async () => new Response(null, { status });
}

/** Create a mock fetch that throws a network error. */
function errorFetch(err: Error): FetchFn {
  return async () => {
    throw err;
  };
}

/** Create a mock fetch that never resolves (for timeout tests). */
function slowFetch(): FetchFn {
  return async (_url, init) => {
    return new Promise<Response>((_resolve, reject) => {
      const signal = init?.signal;
      if (signal) {
        signal.addEventListener("abort", () => {
          reject(new DOMException("The operation was aborted.", "AbortError"));
        });
      }
    });
  };
}

// --- Run tests ---

describe("run", () => {
  it("happy path", async () => {
    const runner = createOllamaRunner({ fetch: mockFetch(happyLines) });
    const result = await runner.run("say hello", { model: "llama3" });

    expect(result.text).toBe("Hello world");
    expect(result.durationMs).toBe(2000);
    expect(result.usage.inputTokens).toBe(100);
    expect(result.usage.outputTokens).toBe(50);
    expect(result.costUSD).toBe(0);
    expect(result.isError).toBe(false);
    expect(result.sessionId).toBe("");
  });

  it("sends system prompt", async () => {
    const captured: { body?: unknown } = {};
    const runner = createOllamaRunner({
      fetch: capturingFetch(happyLines, captured),
    });
    await runner.run("hello", {
      model: "llama3",
      systemPrompt: "You are helpful",
    });

    const body = captured.body as {
      messages: Array<{ role: string; content: string }>;
    };
    expect(body.messages).toHaveLength(2);
    expect(body.messages[0].role).toBe("system");
    expect(body.messages[0].content).toBe("You are helpful");
    expect(body.messages[1].role).toBe("user");
    expect(body.messages[1].content).toBe("hello");
  });

  it("sends append system prompt", async () => {
    const captured: { body?: unknown } = {};
    const runner = createOllamaRunner({
      fetch: capturingFetch(happyLines, captured),
    });
    await runner.run("hello", {
      model: "llama3",
      systemPrompt: "Be helpful",
      appendSystemPrompt: "Be concise",
    });

    const body = captured.body as {
      messages: Array<{ role: string; content: string }>;
    };
    expect(body.messages[0].content).toBe("Be helpful\nBe concise");
  });

  it("sends model name", async () => {
    const captured: { body?: unknown } = {};
    const runner = createOllamaRunner({
      fetch: capturingFetch(happyLines, captured),
    });
    await runner.run("hello", { model: "codellama" });

    const body = captured.body as { model: string };
    expect(body.model).toBe("codellama");
  });

  it("sends ollama options", async () => {
    const captured: { body?: unknown } = {};
    const runner = createOllamaRunner({
      fetch: capturingFetch(happyLines, captured),
    });
    await runner.run("hello", {
      model: "llama3",
      temperature: 0.7,
      numCtx: 4096,
      numPredict: 256,
      seed: 42,
      stop: ["END", "STOP"],
      topK: 40,
      topP: 0.9,
      minP: 0.05,
      format: "json",
      keepAlive: "5m",
    });

    const body = captured.body as {
      format: string;
      keep_alive: string;
      options: {
        temperature: number;
        num_ctx: number;
        num_predict: number;
        seed: number;
        stop: string[];
        top_k: number;
        top_p: number;
        min_p: number;
      };
    };
    expect(body.format).toBe("json");
    expect(body.keep_alive).toBe("5m");
    expect(body.options.temperature).toBe(0.7);
    expect(body.options.num_ctx).toBe(4096);
    expect(body.options.num_predict).toBe(256);
    expect(body.options.seed).toBe(42);
    expect(body.options.stop).toEqual(["END", "STOP"]);
    expect(body.options.top_k).toBe(40);
    expect(body.options.top_p).toBe(0.9);
    expect(body.options.min_p).toBe(0.05);
  });

  it("omits model options when none set", async () => {
    const captured: { body?: unknown } = {};
    const runner = createOllamaRunner({
      fetch: capturingFetch(happyLines, captured),
    });
    await runner.run("hello", { model: "llama3" });

    const body = captured.body as { options?: unknown };
    expect(body.options).toBeUndefined();
  });

  it("sends think field", async () => {
    const captured: { body?: unknown } = {};
    const runner = createOllamaRunner({
      fetch: capturingFetch(happyLines, captured),
    });
    await runner.run("hello", { model: "qwen3", think: true });

    const body = captured.body as { think: boolean };
    expect(body.think).toBe(true);
  });

  it("HTTP 500 throws HttpError", async () => {
    const runner = createOllamaRunner({ fetch: statusFetch(500) });
    await expect(runner.run("hello", { model: "llama3" })).rejects.toThrow(
      HttpError,
    );
  });

  it("HTTP 404 throws NotFoundError", async () => {
    const runner = createOllamaRunner({ fetch: statusFetch(404) });
    await expect(runner.run("hello", { model: "nonexistent" })).rejects.toThrow(
      NotFoundError,
    );
  });

  it("connection refused throws NotFoundError", async () => {
    const runner = createOllamaRunner({
      fetch: errorFetch(new TypeError("fetch failed")),
    });
    await expect(runner.run("hello", { model: "llama3" })).rejects.toThrow(
      NotFoundError,
    );
  });

  it("timeout throws TimeoutError", async () => {
    const runner = createOllamaRunner({ fetch: slowFetch() });
    await expect(
      runner.run("hello", { model: "llama3", timeout: 50 }),
    ).rejects.toThrow(TimeoutError);
  });

  it("cancellation throws CancelledError", async () => {
    const controller = new AbortController();
    const runner = createOllamaRunner({ fetch: slowFetch() });

    setTimeout(() => controller.abort(), 50);

    await expect(
      runner.run("hello", { model: "llama3", signal: controller.signal }),
    ).rejects.toThrow(CancelledError);
  });

  it("invalid JSON throws ParseError", async () => {
    const badFetch: FetchFn = async () =>
      new Response(
        new ReadableStream({
          start(controller) {
            controller.enqueue(new TextEncoder().encode("not valid json\n"));
            controller.close();
          },
        }),
        { status: 200 },
      );
    const runner = createOllamaRunner({ fetch: badFetch });
    await expect(runner.run("hello", { model: "llama3" })).rejects.toThrow(
      ParseError,
    );
  });

  it("no done message throws NoResultError", async () => {
    const lines = [streamLine("partial")];
    const runner = createOllamaRunner({ fetch: mockFetch(lines) });
    await expect(runner.run("hello", { model: "llama3" })).rejects.toThrow(
      NoResultError,
    );
  });

  it("empty response throws NoResultError", async () => {
    const runner = createOllamaRunner({ fetch: mockFetch([]) });
    await expect(runner.run("hello", { model: "llama3" })).rejects.toThrow(
      NoResultError,
    );
  });
});

// --- RunStream tests ---

describe("runStream", () => {
  it("happy path yields correct messages", async () => {
    const runner = createOllamaRunner({ fetch: mockFetch(happyLines) });
    const messages: Message[] = [];

    for await (const msg of runner.runStream("say hello", {
      model: "llama3",
    })) {
      messages.push(msg);
    }

    expect(messages).toHaveLength(3);
    expect(messages[0].type).toBe("assistant");
    expect(messages[1].type).toBe("assistant");
    expect(messages[2].type).toBe("result");
  });

  it("onMessage callback receives all messages", async () => {
    const runner = createOllamaRunner({ fetch: mockFetch(happyLines) });
    const callbackMessages: Message[] = [];
    const channelMessages: Message[] = [];

    for await (const msg of runner.runStream("test callback", {
      model: "llama3",
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
    const runner = createOllamaRunner({ fetch: mockFetch(happyLines) });

    for await (const msg of runner.runStream("test raw", { model: "llama3" })) {
      expect(msg.raw.length).toBeGreaterThan(0);
    }
  });

  it("timeout throws TimeoutError", async () => {
    const runner = createOllamaRunner({ fetch: slowFetch() });

    const consume = async () => {
      for await (const _msg of runner.runStream("hello", {
        model: "llama3",
        timeout: 50,
      })) {
        // drain
      }
    };

    await expect(consume()).rejects.toThrow(TimeoutError);
  });

  it("cancellation throws CancelledError", async () => {
    const controller = new AbortController();
    const runner = createOllamaRunner({ fetch: slowFetch() });

    setTimeout(() => controller.abort(), 50);

    const consume = async () => {
      for await (const _msg of runner.runStream("hello", {
        model: "llama3",
        signal: controller.signal,
      })) {
        // drain
      }
    };

    await expect(consume()).rejects.toThrow(CancelledError);
  });

  it("no done message throws NoResultError", async () => {
    const lines = [streamLine("partial")];
    const runner = createOllamaRunner({ fetch: mockFetch(lines) });

    const consume = async () => {
      for await (const _msg of runner.runStream("hello", { model: "llama3" })) {
        // drain
      }
    };

    await expect(consume()).rejects.toThrow(NoResultError);
  });
});

// --- Start tests ---

describe("start (Session)", () => {
  it("happy path: messages and result", async () => {
    const runner = createOllamaRunner({ fetch: mockFetch(happyLines) });
    const session = runner.start("say hello", { model: "llama3" });

    const messages: Message[] = [];
    for await (const msg of session.messages) {
      messages.push(msg);
    }

    const result = await session.result;

    expect(messages).toHaveLength(3);
    expect(result.text).toBe("Hello world");
    expect(result.durationMs).toBe(2000);
  });

  it("abort cancels the request", async () => {
    const runner = createOllamaRunner({ fetch: slowFetch() });
    const session = runner.start("long task", { model: "llama3" });

    setTimeout(() => session.abort(), 50);

    const messages: Message[] = [];
    for await (const msg of session.messages) {
      messages.push(msg);
    }

    await expect(session.result).rejects.toThrow(CancelledError);
  });

  it("send throws NotSupportedError", () => {
    const runner = createOllamaRunner({ fetch: mockFetch(happyLines) });
    const session = runner.start("hello", { model: "llama3" });

    expect(() => session.send("test")).toThrow(NotSupportedError);

    // Drain to avoid resource leak.
    (async () => {
      for await (const _msg of session.messages) {
        // drain
      }
    })();
  });
});

// --- Thinking model tests ---

describe("thinking model", () => {
  const thinkingLines = [
    streamLine("", "Let me think"),
    streamLine("", " about this"),
    streamLine("The answer"),
    streamLine(" is 42"),
    doneLine("", 3_000_000_000, 50, 100),
  ];

  it("run collects content text only", async () => {
    const runner = createOllamaRunner({ fetch: mockFetch(thinkingLines) });
    const result = await runner.run("hello", { model: "qwen3" });

    expect(result.text).toBe("The answer is 42");
  });

  it("runStream exposes thinking in raw JSON", async () => {
    const runner = createOllamaRunner({ fetch: mockFetch(thinkingLines) });
    const thinkingParts: string[] = [];
    const contentParts: string[] = [];

    for await (const msg of runner.runStream("hello", { model: "qwen3" })) {
      const chunk = JSON.parse(msg.raw) as ChatResponse;
      if (chunk.message.thinking) {
        thinkingParts.push(chunk.message.thinking);
      }
      if (chunk.message.content) {
        contentParts.push(chunk.message.content);
      }
    }

    expect(thinkingParts.join("")).toBe("Let me think about this");
    expect(contentParts.join("")).toBe("The answer is 42");
  });
});
