import type { GeminiRunOptions } from "./options.js";

/** Build CLI arguments from prompt and options. */
export function buildArgs(
  prompt: string,
  options: GeminiRunOptions = {},
): string[] {
  const args: string[] = ["--output-format", "stream-json"];

  // Common options.
  if (options.model) {
    args.push("--model", options.model);
  }
  if (options.dangerouslySkipPermissions) {
    args.push("--yolo");
  }

  // Gemini-specific options.
  if (options.approvalMode) {
    args.push("--approval-mode", options.approvalMode);
  }
  if (options.sandbox) {
    args.push("--sandbox");
  }
  if (options.extensions) {
    for (const ext of options.extensions) {
      args.push("--extensions", ext);
    }
  }
  if (options.allowedTools) {
    for (const tool of options.allowedTools) {
      args.push("--allowed-tools", tool);
    }
  }
  if (options.resume) {
    args.push("--resume", options.resume);
  }
  if (options.includeDirs) {
    for (const dir of options.includeDirs) {
      args.push("--include-directories", dir);
    }
  }
  if (options.rawOutput) {
    args.push("--raw-output");
  }

  args.push("--prompt", prompt);
  return args;
}
