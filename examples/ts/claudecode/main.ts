// This example demonstrates how to use the agentrunner TypeScript library to
// invoke Claude Code CLI programmatically, covering basic usage, streaming,
// session management, and the Session object pattern.
//
// Prerequisites:
//   - Claude Code CLI installed (>= 1.0.12): https://docs.anthropic.com/en/docs/claude-code
//   - Authenticated with `claude login`
//
// Run:
//   npx tsx main.ts
//   npx tsx main.ts --binary /path/to/claude

import { parseArgs } from "node:util";
import { createClaudeRunner, parse } from "agentrunner/claudecode";
import type { ClaudeRunOptions } from "agentrunner/claudecode";
import type { Runner } from "agentrunner";

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
  // --- Example 1: Simple Run ---
  console.log("=== Example 1: Simple Run ===");
  await exampleSimpleRun(runner);

  // --- Example 2: Streaming ---
  console.log("\n=== Example 2: Streaming ===");
  await exampleStreaming(runner, values.verbose ?? false);

  // --- Example 3: Session Resume ---
  console.log("\n=== Example 3: Session Resume ===");
  await exampleSessionResume(runner);

  // --- Example 4: Session Object ---
  console.log("\n=== Example 4: Session Object ===");
  await exampleSession(runner);
}

/** Send a single prompt and print the result. */
async function exampleSimpleRun(runner: Runner) {
  const prompt = "What is 2+2? Reply with just the number.";
  console.log(`Prompt:   ${prompt}`);

  const result = await runner.run(prompt, {
    maxTurns: 1,
    timeout: 30_000,
  });

  console.log(`Response: ${result.text}`);
  console.log(`Cost:     $${result.costUSD.toFixed(4)}`);
  console.log(
    `Tokens:   ${result.usage.inputTokens} in / ${result.usage.outputTokens} out`,
  );
  console.log(`Duration: ${result.durationMs}ms`);
  console.log(`Session:  ${result.sessionId}`);
  console.log(`Error:    ${result.isError}`);
  console.log(`Exit:     ${result.exitCode}`);
}

/** Use runStream to print messages as they arrive. */
async function exampleStreaming(runner: Runner, verbose: boolean) {
  const prompt = "List 3 fun facts about TypeScript. Be brief.";
  console.log(`Prompt: ${prompt}`);
  console.log("---");

  const stream = runner.runStream(prompt, {
    maxTurns: 1,
    timeout: 30_000,
    includePartialMessages: true,
  });

  let model = "unknown";

  for await (const msg of stream) {
    switch (msg.type) {
      case "system": {
        if (verbose) console.log(`[system] ${msg.raw}`);
        const parsed = parse(msg.raw);
        if (parsed.model) model = parsed.model;
        break;
      }
      case "assistant": {
        // With --include-partial-messages, the CLI emits two kinds of
        // messages mapped to "assistant":
        //   1. stream_event with content_block_delta — real-time text deltas
        //   2. assistant — full accumulated message (arrives at the end)
        // Print deltas for real-time streaming; skip the final assistant
        // message to avoid duplicating the output.
        const parsed = parse(msg.raw);
        if (parsed.type === "stream_event") {
          const delta = parsed.event?.delta;
          if (delta?.type === "text_delta" && delta.text) {
            process.stdout.write(delta.text);
          }
        }
        break;
      }
      case "result": {
        const parsed = parse(msg.raw);
        console.log("\n---");
        console.log(`Cost:     $${(parsed.total_cost_usd ?? 0).toFixed(4)}`);
        console.log(`Duration: ${parsed.duration_ms ?? 0}ms`);
        console.log(`Turns:    ${parsed.num_turns ?? 0}`);
        console.log(`Model:    ${parsed.model ?? model}`);
        console.log(`Session:  ${parsed.session_id ?? ""}`);
        console.log(`Error:    ${parsed.is_error ?? false}`);
        break;
      }
    }
  }
}

/** Demonstrate multi-turn conversations using session IDs. */
async function exampleSessionResume(runner: Runner) {
  // First turn: ask Claude to remember something.
  const prompt1 = "Remember this number: 42. Just confirm you've noted it.";
  console.log(`Prompt 1: ${prompt1}`);

  const first = await runner.run(prompt1, {
    maxTurns: 1,
    timeout: 30_000,
  });

  console.log(`Response: ${first.text}`);
  console.log(`Session:  ${first.sessionId}`);

  if (!first.sessionId) {
    throw new Error("No session ID returned — cannot demonstrate resume");
  }

  // Second turn: resume the session and reference the earlier context.
  const prompt2 = "What number did I ask you to remember?";
  console.log(`\nPrompt 2: ${prompt2} (resume: ${first.sessionId})`);

  const resumeOptions: ClaudeRunOptions = {
    maxTurns: 1,
    timeout: 30_000,
    resume: first.sessionId,
  };
  const second = await runner.run(prompt2, resumeOptions);

  console.log(`Response: ${second.text}`);
}

/** Demonstrate the Session object pattern with full lifecycle control. */
async function exampleSession(runner: Runner) {
  const prompt = "What is the capital of France? Reply with just the city name.";
  console.log(`Prompt: ${prompt}`);

  const session = runner.start(prompt, {
    maxTurns: 1,
    timeout: 30_000,
  });

  // Iterate messages as they arrive.
  for await (const msg of session.messages) {
    const preview = msg.raw.length > 80 ? msg.raw.slice(0, 80) + "..." : msg.raw;
    console.log(`[${msg.type}] ${preview}`);
  }

  // Get the final result.
  const result = await session.result;

  console.log(`Response: ${result.text}`);
  console.log(`Cost:     $${result.costUSD.toFixed(4)}`);
  console.log(`Session:  ${result.sessionId}`);
}

main().catch((err) => {
  console.error("error:", err);
  process.exit(1);
});
