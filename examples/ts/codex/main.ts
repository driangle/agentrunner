// This example demonstrates how to use the agentrunner TypeScript library to
// invoke the Codex CLI programmatically, covering basic usage, streaming,
// session management, and the Session object pattern.
//
// Prerequisites:
//   - Codex CLI installed (>= 0.118.0): https://github.com/openai/codex
//
// Run:
//   npx tsx main.ts
//   npx tsx main.ts --binary /path/to/codex

import { parseArgs } from "node:util";
import {
  createCodexRunner,
  messageText,
  messageToolName,
  messageToolOutput,
} from "agentrunner/codex";
import type { ItemStreamMessage } from "agentrunner/codex";

const { values } = parseArgs({
  options: {
    binary: { type: "string", default: "codex" },
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

const runner = createCodexRunner({
  binary: values.binary,
  logger,
});

async function main() {
  // --- Example 1: Simple Run ---
  console.log("=== Example 1: Simple Run ===");
  await exampleSimpleRun(runner);

  // --- Example 2: Streaming ---
  console.log("\n=== Example 2: Streaming ===");
  await exampleStreaming(runner);

  // --- Example 3: Session Resume ---
  console.log("\n=== Example 3: Session Resume ===");
  await exampleSessionResume(runner);

  // --- Example 4: Session Object ---
  console.log("\n=== Example 4: Session Object ===");
  await exampleSession(runner);
}

/** Send a single prompt and print the result. */
async function exampleSimpleRun(runner: ReturnType<typeof createCodexRunner>) {
  const prompt = "What is 2+2? Reply with just the number.";
  console.log(`Prompt:   ${prompt}`);

  const result = await runner.run(prompt, {
    sandbox: "read-only",
    timeout: 60_000,
  });

  console.log(`Response: ${result.text}`);
  console.log(
    `Tokens:   ${result.usage.inputTokens} in / ${result.usage.outputTokens} out`,
  );
  console.log(`Session:  ${result.sessionId}`);
  console.log(`Error:    ${result.isError}`);
  console.log(`Exit:     ${result.exitCode}`);
}

/** Use runStream to print messages as they arrive. */
async function exampleStreaming(runner: ReturnType<typeof createCodexRunner>) {
  const prompt = "List 3 fun facts about TypeScript. Be brief.";
  console.log(`Prompt: ${prompt}`);
  console.log("---");

  for await (const msg of runner.runStream(prompt, {
    sandbox: "read-only",
    timeout: 60_000,
  })) {
    switch (msg.type) {
      case "system":
        if (values.verbose) console.log(`[system] ${msg.raw}`);
        break;
      case "assistant": {
        const text = messageText(msg);
        if (text) console.log(`[assistant] ${text}`);
        break;
      }
      case "tool_use": {
        const tool = messageToolName(msg);
        if (tool) console.log(`[tool_use] ${tool}`);
        break;
      }
      case "tool_result": {
        const output = messageToolOutput(msg);
        if (output) console.log(`[tool_result] ${output.slice(0, 100)}...`);
        break;
      }
      case "result":
        console.log("---");
        console.log(`[result] stream complete`);
        break;
    }
  }
}

/** Demonstrate multi-turn conversations using session IDs. */
async function exampleSessionResume(
  runner: ReturnType<typeof createCodexRunner>,
) {
  // First turn: ask Codex to remember something.
  const prompt1 = "Remember this number: 42. Just confirm you've noted it.";
  console.log(`Prompt 1: ${prompt1}`);

  const first = await runner.run(prompt1, {
    sandbox: "read-only",
    timeout: 60_000,
  });

  console.log(`Response: ${first.text}`);
  console.log(`Session:  ${first.sessionId}`);

  if (!first.sessionId) {
    throw new Error("No session ID returned — cannot demonstrate resume");
  }

  // Second turn: resume the session and reference the earlier context.
  const prompt2 = "What number did I ask you to remember?";
  console.log(`\nPrompt 2: ${prompt2} (resume: ${first.sessionId})`);

  const second = await runner.run(prompt2, {
    sandbox: "read-only",
    timeout: 60_000,
    resume: first.sessionId,
  });

  console.log(`Response: ${second.text}`);
}

/** Demonstrate the Session object pattern with full lifecycle control. */
async function exampleSession(runner: ReturnType<typeof createCodexRunner>) {
  const prompt =
    "What is the capital of France? Reply with just the city name.";
  console.log(`Prompt: ${prompt}`);

  const session = runner.start(prompt, {
    sandbox: "read-only",
    timeout: 60_000,
  });

  // Iterate messages as they arrive.
  for await (const msg of session.messages) {
    const preview =
      msg.raw.length > 80 ? msg.raw.slice(0, 80) + "..." : msg.raw;
    console.log(`[${msg.type}] ${preview}`);
  }

  // Get the final result.
  const result = await session.result;

  console.log(`Response: ${result.text}`);
  console.log(`Session:  ${result.sessionId}`);
}

main().catch((err) => {
  console.error("error:", err);
  process.exit(1);
});
