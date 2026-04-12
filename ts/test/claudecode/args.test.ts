import { describe, it, expect } from "vitest";
import { buildArgs } from "../../src/claudecode/args.js";

describe("buildArgs", () => {
  it("builds minimal args", () => {
    const args = buildArgs("hello world");
    expect(args).toEqual([
      "--print",
      "--output-format",
      "stream-json",
      "--verbose",
      "--",
      "hello world",
    ]);
  });

  it("includes all common options", () => {
    const args = buildArgs("test prompt", {
      model: "claude-sonnet-4-6",
      systemPrompt: "You are helpful",
      appendSystemPrompt: "Be concise",
      maxTurns: 5,
      skipPermissions: true,
    });

    const joined = args.join(" ");
    expect(joined).toContain("--model claude-sonnet-4-6");
    expect(joined).toContain("--system-prompt You are helpful");
    expect(joined).toContain("--append-system-prompt Be concise");
    expect(joined).toContain("--max-turns 5");
    expect(joined).toContain("--dangerously-skip-permissions");

    // Prompt must be last, after "--".
    expect(args.at(-1)).toBe("test prompt");
    expect(args.at(-2)).toBe("--");
  });

  it("includes Claude-specific options", () => {
    const args = buildArgs("test", {
      allowedTools: ["Read", "Write"],
      disallowedTools: ["Bash"],
      mcpConfig: "/tmp/mcp.json",
      jsonSchema: `{"type":"object"}`,
      maxBudgetUSD: 1.5,
      resume: "sess-123",
    });

    const joined = args.join(" ");
    expect(joined).toContain("--allowedTools Read");
    expect(joined).toContain("--allowedTools Write");
    expect(joined).toContain("--disallowedTools Bash");
    expect(joined).toContain("--mcp-config /tmp/mcp.json");
    expect(joined).toContain(`--json-schema {"type":"object"}`);
    expect(joined).toContain("--max-budget-usd 1.5");
    expect(joined).toContain("--resume sess-123");
  });

  it("includes session-id", () => {
    const args = buildArgs("test", { sessionId: "my-session-42" });
    const joined = args.join(" ");
    expect(joined).toContain("--session-id my-session-42");
  });

  it("includes --continue flag", () => {
    const args = buildArgs("test", { continueSession: true });
    expect(args).toContain("--continue");
  });

  it("includes --include-partial-messages flag", () => {
    const args = buildArgs("test", { includePartialMessages: true });
    expect(args).toContain("--include-partial-messages");
  });

  it("includes channel flags when channelEnabled", () => {
    const args = buildArgs("test", { channelEnabled: true });
    const joined = args.join(" ");
    expect(joined).toContain("--channels server:agentrunner-channel");
    expect(joined).toContain(
      "--dangerously-load-development-channels server:agentrunner-channel",
    );
    expect(joined).toContain("--strict-mcp-config");
  });

  it("includes --debug-file when debugFile is set", () => {
    const args = buildArgs("test", { debugFile: "/tmp/claude-debug.log" });
    const joined = args.join(" ");
    expect(joined).toContain("--debug-file /tmp/claude-debug.log");
  });

  it("omits channel flags when channelEnabled is false", () => {
    const args = buildArgs("test", {});
    const joined = args.join(" ");
    expect(joined).not.toContain("--channels");
    expect(joined).not.toContain("--dangerously-load-development-channels");
  });

  it("omits zero/undefined optional values", () => {
    const args = buildArgs("test", {
      maxTurns: 0,
      maxBudgetUSD: 0,
    });
    const joined = args.join(" ");
    expect(joined).not.toContain("--max-turns");
    expect(joined).not.toContain("--max-budget-usd");
  });
});
