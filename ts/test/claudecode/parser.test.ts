import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { parse } from "../../src/claudecode/parser.js";

describe("parse", () => {
  it("parses system/init", () => {
    const msg = parse(
      `{"type":"system","subtype":"init","session_id":"abc-123","model":"claude-sonnet-4-6","tools":["Bash","Read"]}`,
    );
    expect(msg.type).toBe("system");
    expect(msg.subtype).toBe("init");
    expect(msg.session_id).toBe("abc-123");
    expect(msg.model).toBe("claude-sonnet-4-6");
    expect(msg.tools).toHaveLength(2);
  });

  it("parses assistant/text with content lifting", () => {
    const msg = parse(
      `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","content":[{"type":"text","text":"Hello world"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5}}}`,
    );
    expect(msg.type).toBe("assistant");
    expect(msg.content).toHaveLength(1);
    expect(msg.content[0].type).toBe("text");
    expect(msg.content[0].text).toBe("Hello world");
    expect(msg.message?.stop_reason).toBe("end_turn");
    expect(msg.message?.usage?.input_tokens).toBe(10);
  });

  it("parses assistant/thinking", () => {
    const msg = parse(
      `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_02","content":[{"type":"thinking","thinking":"Let me think about this..."}]}}`,
    );
    expect(msg.content).toHaveLength(1);
    expect(msg.content[0].type).toBe("thinking");
    expect(msg.content[0].thinking).toBe("Let me think about this...");
  });

  it("parses assistant/tool_use", () => {
    const msg = parse(
      `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_03","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/test.go"}}]}}`,
    );
    expect(msg.content).toHaveLength(1);
    expect(msg.content[0].type).toBe("tool_use");
    expect(msg.content[0].name).toBe("Read");
    expect(msg.content[0].input).toEqual({ file_path: "/tmp/test.go" });
  });

  it("parses result/success", () => {
    const msg = parse(
      `{"type":"result","subtype":"success","is_error":false,"result":"Task completed","total_cost_usd":0.05,"duration_ms":1234,"duration_api_ms":1100,"num_turns":3,"session_id":"sess-1"}`,
    );
    expect(msg.type).toBe("result");
    expect(msg.subtype).toBe("success");
    expect(msg.is_error).toBe(false);
    expect(msg.result).toBe("Task completed");
    expect(msg.total_cost_usd).toBe(0.05);
    expect(msg.duration_ms).toBe(1234);
    expect(msg.num_turns).toBe(3);
    expect(msg.session_id).toBe("sess-1");
  });

  it("parses result/error", () => {
    const msg = parse(
      `{"type":"result","subtype":"error","is_error":true,"result":"Something failed","session_id":"sess-2"}`,
    );
    expect(msg.subtype).toBe("error");
    expect(msg.is_error).toBe(true);
    expect(msg.result).toBe("Something failed");
  });

  it("parses stream_event/message_start", () => {
    const msg = parse(
      `{"type":"stream_event","event":{"type":"message_start","message":{"model":"claude-sonnet-4-6","id":"msg_04","usage":{"input_tokens":100,"output_tokens":0}}},"session_id":"sess-3"}`,
    );
    expect(msg.type).toBe("stream_event");
    expect(msg.event).toBeDefined();
    expect(msg.event?.type).toBe("message_start");
    expect(msg.event?.message?.model).toBe("claude-sonnet-4-6");
  });

  it("parses stream_event/content_block_start", () => {
    const msg = parse(
      `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_01","name":"Bash"}},"session_id":"sess-3"}`,
    );
    expect(msg.event?.type).toBe("content_block_start");
    expect(msg.event?.content_block?.name).toBe("Bash");
  });

  it("parses stream_event/content_block_delta/text", () => {
    const msg = parse(
      `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}},"session_id":"sess-3"}`,
    );
    expect(msg.event?.delta?.type).toBe("text_delta");
    expect(msg.event?.delta?.text).toBe("Hello");
  });

  it("parses stream_event/content_block_delta/thinking", () => {
    const msg = parse(
      `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"hmm"}},"session_id":"sess-3"}`,
    );
    expect(msg.event?.delta?.thinking).toBe("hmm");
  });

  it("parses stream_event/content_block_delta/input_json", () => {
    const msg = parse(
      `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\\"key\\":"}},"session_id":"sess-3"}`,
    );
    expect(msg.event?.delta?.partial_json).toBe(`{"key":`);
  });

  it("parses stream_event/message_delta", () => {
    const msg = parse(
      `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":42}},"session_id":"sess-3"}`,
    );
    expect(msg.event?.delta?.stop_reason).toBe("end_turn");
    expect(msg.event?.usage?.output_tokens).toBe(42);
  });

  it("parses rate_limit_event", () => {
    const msg = parse(
      `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed_warning","rateLimitType":"seven_day","utilization":0.56,"resetsAt":1772805600,"isUsingOverage":false}}`,
    );
    expect(msg.type).toBe("rate_limit_event");
    expect(msg.rate_limit_info).toBeDefined();
    expect(msg.rate_limit_info?.status).toBe("allowed_warning");
    expect(msg.rate_limit_info?.utilization).toBe(0.56);
    expect(msg.rate_limit_info?.isUsingOverage).toBe(false);
  });

  it("handles unknown type (forward compatible)", () => {
    const msg = parse(`{"type":"new_future_type","some_field":"value"}`);
    expect(msg.type).toBe("new_future_type");
  });

  it("throws on invalid JSON", () => {
    expect(() => parse("{not valid json}")).toThrow();
  });

  it("throws on empty string", () => {
    expect(() => parse("")).toThrow();
  });

  it("throws on missing type field", () => {
    expect(() => parse(`{"no_type":"here"}`)).toThrow(
      "not a valid stream message",
    );
  });
});

describe("parse session fixture", () => {
  it("parses all lines from a real session", () => {
    const fixturePath = join(
      __dirname,
      "testdata",
      "claude-code-session.jsonl",
    );
    const content = readFileSync(fixturePath, "utf-8");
    const lines = content.split("\n").filter((l) => l.trim());

    const typeCounts: Record<string, number> = {};

    for (const line of lines) {
      const msg = parse(line);
      typeCounts[msg.type] = (typeCounts[msg.type] ?? 0) + 1;

      // Verify assistant messages have content lifted.
      if (
        msg.type === "assistant" &&
        msg.message?.content &&
        msg.message.content.length > 0
      ) {
        expect(msg.content.length).toBeGreaterThan(0);
      }

      // Verify stream events have inner event parsed.
      if (msg.type === "stream_event") {
        expect(msg.event).toBeDefined();
      }
    }

    // Verify we saw the expected message types.
    const expectedTypes = [
      "system",
      "assistant",
      "result",
      "stream_event",
      "rate_limit_event",
    ];
    for (const typ of expectedTypes) {
      expect(typeCounts[typ] ?? 0).toBeGreaterThan(0);
    }
  });
});
