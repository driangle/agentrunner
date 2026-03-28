import { describe, it, expect, afterEach } from "vitest";
import { resolveChannelBinary } from "../../src/channel/binary.js";

describe("resolveChannelBinary", () => {
  const originalEnv = process.env.AGENTRUNNER_CHANNEL_BIN;

  afterEach(() => {
    if (originalEnv === undefined) {
      delete process.env.AGENTRUNNER_CHANNEL_BIN;
    } else {
      process.env.AGENTRUNNER_CHANNEL_BIN = originalEnv;
    }
  });

  it("returns AGENTRUNNER_CHANNEL_BIN when set", () => {
    process.env.AGENTRUNNER_CHANNEL_BIN = "/custom/path/agentrunner-channel";
    expect(resolveChannelBinary()).toBe("/custom/path/agentrunner-channel");
  });

  it("throws when binary is not found", () => {
    delete process.env.AGENTRUNNER_CHANNEL_BIN;
    // With no platform package installed and no binary on PATH,
    // this should throw. We rely on the test environment not having
    // agentrunner-channel installed.
    expect(() => resolveChannelBinary()).toThrow(
      "agentrunner-channel binary not found",
    );
  });
});
