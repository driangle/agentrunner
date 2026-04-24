import type { CodexRunOptions } from "./options.js";

/** Build CLI arguments from prompt and options. */
export function buildArgs(
  prompt: string,
  options: CodexRunOptions = {},
): string[] {
  const args: string[] = ["exec", "--json"];

  // Common options.
  if (options.model) {
    args.push("--model", options.model);
  }
  if (options.workingDir) {
    args.push("--cd", options.workingDir);
  }
  if (options.dangerouslySkipPermissions) {
    args.push("--dangerously-bypass-approvals-and-sandbox");
  }

  // Codex-specific options.
  if (options.sandbox) {
    args.push("--sandbox", options.sandbox);
  }
  if (options.approval) {
    args.push("--ask-for-approval", options.approval);
  }
  if (options.outputSchema) {
    args.push("--output-schema", options.outputSchema);
  }
  if (options.images) {
    for (const img of options.images) {
      args.push("--image", img);
    }
  }
  if (options.profile) {
    args.push("--profile", options.profile);
  }
  if (options.fullAuto) {
    args.push("--full-auto");
  }
  if (options.ephemeral) {
    args.push("--ephemeral");
  }
  if (options.search) {
    args.push("--search");
  }
  if (options.addDirs) {
    for (const dir of options.addDirs) {
      args.push("--add-dir", dir);
    }
  }
  if (options.resume) {
    args.push("resume", options.resume);
  }

  args.push("--", prompt);
  return args;
}
