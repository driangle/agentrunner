import type { ClaudeRunOptions } from "./options.js";

/** Build CLI arguments from prompt and options. */
export function buildArgs(
  prompt: string,
  options: ClaudeRunOptions = {},
): string[] {
  const args: string[] = [
    "--print",
    "--output-format",
    "stream-json",
    "--verbose", // Required with stream-json to get full message output.
  ];

  // Common options.
  if (options.model) {
    args.push("--model", options.model);
  }
  if (options.systemPrompt) {
    args.push("--system-prompt", options.systemPrompt);
  }
  if (options.appendSystemPrompt) {
    args.push("--append-system-prompt", options.appendSystemPrompt);
  }
  if (options.maxTurns != null && options.maxTurns > 0) {
    args.push("--max-turns", String(options.maxTurns));
  }
  if (options.skipPermissions) {
    args.push("--dangerously-skip-permissions");
  }

  // Claude-specific options.
  if (options.allowedTools) {
    for (const tool of options.allowedTools) {
      args.push("--allowedTools", tool);
    }
  }
  if (options.disallowedTools) {
    for (const tool of options.disallowedTools) {
      args.push("--disallowedTools", tool);
    }
  }
  if (options.mcpConfig) {
    args.push("--mcp-config", options.mcpConfig);
  }
  if (options.jsonSchema) {
    args.push("--json-schema", options.jsonSchema);
  }
  if (options.maxBudgetUSD != null && options.maxBudgetUSD > 0) {
    args.push("--max-budget-usd", String(options.maxBudgetUSD));
  }
  if (options.resume) {
    args.push("--resume", options.resume);
  }
  if (options.continueSession) {
    args.push("--continue");
  }
  if (options.sessionId) {
    args.push("--session-id", options.sessionId);
  }
  if (options.includePartialMessages) {
    args.push("--include-partial-messages");
  }
  if (options.channelEnabled) {
    args.push(
      "--dangerously-load-development-channels",
      "server:agentrunner-channel",
    );
  }

  args.push("--", prompt);
  return args;
}
