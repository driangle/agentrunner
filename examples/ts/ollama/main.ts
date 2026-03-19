// This example demonstrates how to use the agentrunner TypeScript library to
// invoke Ollama models programmatically, covering basic usage, streaming,
// thinking models, and the Session object pattern.
//
// Prerequisites:
//   - Ollama installed and running: https://ollama.com
//   - A model pulled: ollama pull llama3.2
//
// Run:
//   npx tsx main.ts --model llama3.2
//   npx tsx main.ts --model codellama --base-url http://localhost:11434

import { parseArgs } from "node:util";
import { createOllamaRunner } from "agentrunner/ollama";
import type { OllamaRunOptions } from "agentrunner/ollama";
import type { ChatResponse } from "agentrunner/ollama";
import type { Runner } from "agentrunner";

const { values } = parseArgs({
  options: {
    model: { type: "string" },
    "base-url": { type: "string", default: "http://localhost:11434" },
    verbose: { type: "boolean", default: false },
  },
});

if (!values.model) {
  console.error("error: --model is required (e.g. --model llama3.2)");
  process.exit(1);
}
const model: string = values.model;

const logger = values.verbose
  ? {
      debug: (msg: string, ...args: unknown[]) =>
        console.error("[debug]", msg, ...args),
      error: (msg: string, ...args: unknown[]) =>
        console.error("[error]", msg, ...args),
    }
  : undefined;

const runner = createOllamaRunner({
  baseURL: values["base-url"],
  logger,
});

async function main() {
  // --- Example 1: Simple Run ---
  console.log("=== Example 1: Simple Run ===");
  await exampleSimpleRun(runner, model);

  // --- Example 2: Streaming ---
  console.log("\n=== Example 2: Streaming ===");
  await exampleStreaming(runner, model);

  // --- Example 3: Thinking Model ---
  console.log("\n=== Example 3: Thinking Model ===");
  await exampleThinking(runner, model);

  // --- Example 4: Session Object ---
  console.log("\n=== Example 4: Session Object ===");
  await exampleSession(runner, model);
}

/** Send a single prompt and print the result. */
async function exampleSimpleRun(runner: Runner, model: string) {
  const prompt = "What is 2+2? Reply with just the number.";
  console.log(`Prompt:   ${prompt}`);

  const result = await runner.run(prompt, { model, timeout: 300_000 });

  console.log(`Response: ${result.text}`);
  console.log(`Cost:     $${result.costUSD.toFixed(4)} (always 0 for local models)`);
  console.log(
    `Tokens:   ${result.usage.inputTokens} in / ${result.usage.outputTokens} out`,
  );
  console.log(`Duration: ${result.durationMs}ms`);
  console.log(`Error:    ${result.isError}`);
}

/** Use runStream to print tokens as they arrive. */
async function exampleStreaming(runner: Runner, model: string) {
  const prompt = "List 3 fun facts about TypeScript. Be brief.";
  console.log(`Prompt: ${prompt}`);
  console.log("---");

  const options: OllamaRunOptions = {
    model,
    timeout: 300_000,
    systemPrompt: "You are a helpful assistant. Keep answers concise.",
    temperature: 0.7,
  };

  for await (const msg of runner.runStream(prompt, options)) {
    if (msg.type === "assistant") {
      const chunk = JSON.parse(msg.raw) as ChatResponse;
      if (chunk.message.content) {
        process.stdout.write(chunk.message.content);
      }
    } else if (msg.type === "result") {
      const final = JSON.parse(msg.raw) as ChatResponse;
      console.log("\n---");
      console.log(
        `Duration: ${final.total_duration ? Math.floor(final.total_duration / 1e6) : 0}ms`,
      );
      console.log(
        `Tokens:   ${final.prompt_eval_count ?? 0} in / ${final.eval_count ?? 0} out`,
      );
    }
  }
}

/** Demonstrate streaming with a thinking model (e.g. qwen3). */
async function exampleThinking(runner: Runner, model: string) {
  const prompt = "How many r's are in the word strawberry?";
  console.log(`Prompt: ${prompt}`);
  console.log("---");

  const options: OllamaRunOptions = {
    model,
    timeout: 300_000,
    think: true,
  };

  for await (const msg of runner.runStream(prompt, options)) {
    if (msg.type !== "assistant") continue;

    const chunk = JSON.parse(msg.raw) as ChatResponse;
    if (chunk.message.thinking) {
      // Dim text for thinking output.
      process.stdout.write(`\x1b[2m${chunk.message.thinking}\x1b[0m`);
    }
    if (chunk.message.content) {
      process.stdout.write(chunk.message.content);
    }
  }
  console.log("\n---");
}

/** Demonstrate the Session object pattern with full lifecycle control. */
async function exampleSession(runner: Runner, model: string) {
  const prompt = "What is the capital of France? Reply with just the city name.";
  console.log(`Prompt: ${prompt}`);

  const session = runner.start(prompt, { model, timeout: 300_000 });

  // Iterate messages as they arrive, printing streamed content.
  let count = 0;
  for await (const msg of session.messages) {
    count++;
    if (msg.type === "assistant") {
      const chunk = JSON.parse(msg.raw) as ChatResponse;
      if (chunk.message.content) {
        process.stdout.write(chunk.message.content);
      }
    }
  }
  console.log();
  console.log(`Received ${count} messages`);

  // Get the final result.
  const result = await session.result;

  console.log(`Response: ${result.text}`);
  console.log(`Duration: ${result.durationMs}ms`);
}

main().catch((err) => {
  console.error("error:", err);
  process.exit(1);
});
