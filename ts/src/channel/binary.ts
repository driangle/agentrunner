import { createRequire } from "node:module";
import { join, dirname } from "node:path";
import { accessSync, constants } from "node:fs";
import { execFileSync } from "node:child_process";

const PLATFORM_PACKAGES: Record<string, string> = {
  "darwin-arm64": "@agentrunner/channel-darwin-arm64",
  "darwin-x64": "@agentrunner/channel-darwin-x64",
  "linux-arm64": "@agentrunner/channel-linux-arm64",
  "linux-x64": "@agentrunner/channel-linux-x64",
  "win32-x64": "@agentrunner/channel-win32-x64",
};

/**
 * Resolve the path to the `agentrunner-channel` binary.
 *
 * Resolution order:
 * 1. `AGENTRUNNER_CHANNEL_BIN` environment variable
 * 2. Platform-specific npm package (installed via optionalDependencies)
 * 3. `agentrunner-channel` on $PATH
 *
 * @throws if the binary cannot be found.
 */
export function resolveChannelBinary(): string {
  const envPath = process.env.AGENTRUNNER_CHANNEL_BIN;
  if (envPath) {
    return envPath;
  }

  const npmPath = resolveFromNpmPackage();
  if (npmPath) {
    return npmPath;
  }

  const pathBinary = resolveFromPath();
  if (pathBinary) {
    return pathBinary;
  }

  const key = `${process.platform}-${process.arch}`;
  throw new Error(
    `agentrunner-channel binary not found. Install the platform package ` +
      `(${PLATFORM_PACKAGES[key] ?? "unsupported platform: " + key}), ` +
      `add it to $PATH, or set AGENTRUNNER_CHANNEL_BIN.`,
  );
}

function resolveFromNpmPackage(): string | undefined {
  const key = `${process.platform}-${process.arch}`;
  const pkg = PLATFORM_PACKAGES[key];
  if (!pkg) return undefined;

  const ext = process.platform === "win32" ? ".exe" : "";
  const binName = `agentrunner-channel${ext}`;

  try {
    const require = createRequire(import.meta.url);
    const pkgJson = require.resolve(`${pkg}/package.json`);
    const binPath = join(dirname(pkgJson), "bin", binName);
    accessSync(binPath, constants.X_OK);
    return binPath;
  } catch {
    return undefined;
  }
}

function resolveFromPath(): string | undefined {
  const name =
    process.platform === "win32"
      ? "agentrunner-channel.exe"
      : "agentrunner-channel";
  try {
    const cmd = process.platform === "win32" ? "where" : "which";
    const result = execFileSync(cmd, [name], { encoding: "utf8" });
    const resolved = result.trim().split("\n")[0];
    if (resolved) return resolved;
  } catch {
    // not found on PATH
  }
  return undefined;
}
