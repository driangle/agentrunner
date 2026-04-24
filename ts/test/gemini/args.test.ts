import { describe, it, expect } from "vitest";
import { buildArgs } from "../../src/gemini/args.js";

describe("buildArgs", () => {
  it("minimal args", () => {
    const args = buildArgs("hello world");
    expect(args).toEqual([
      "--output-format",
      "stream-json",
      "--prompt",
      "hello world",
    ]);
  });

  it("common options", () => {
    const args = buildArgs("hello", { model: "gemini-2.0-flash" });
    expect(args).toContain("--model");
    expect(args).toContain("gemini-2.0-flash");
  });

  it("dangerously skip permissions maps to --yolo", () => {
    const args = buildArgs("hello", { dangerouslySkipPermissions: true });
    expect(args).toContain("--yolo");
  });

  it("gemini-specific options", () => {
    const args = buildArgs("hello", {
      approvalMode: "auto_edit",
      sandbox: true,
      extensions: ["ext1", "ext2"],
      allowedTools: ["Bash", "ReadFile"],
      resume: "sess-123",
      includeDirs: ["/dir1", "/dir2"],
      rawOutput: true,
    });

    expect(args).toContain("--approval-mode");
    expect(args).toContain("auto_edit");
    expect(args).toContain("--sandbox");
    expect(args).toContain("--raw-output");
    expect(args).toContain("--resume");
    expect(args).toContain("sess-123");

    // Extensions: --extensions ext1 --extensions ext2
    const extIndices = args.reduce<number[]>((acc, a, i) => {
      if (a === "--extensions") acc.push(i);
      return acc;
    }, []);
    expect(extIndices).toHaveLength(2);
    expect(args[extIndices[0] + 1]).toBe("ext1");
    expect(args[extIndices[1] + 1]).toBe("ext2");

    // AllowedTools: --allowed-tools Bash --allowed-tools ReadFile
    const toolIndices = args.reduce<number[]>((acc, a, i) => {
      if (a === "--allowed-tools") acc.push(i);
      return acc;
    }, []);
    expect(toolIndices).toHaveLength(2);
    expect(args[toolIndices[0] + 1]).toBe("Bash");
    expect(args[toolIndices[1] + 1]).toBe("ReadFile");

    // IncludeDirs: --include-directories /dir1 --include-directories /dir2
    const dirIndices = args.reduce<number[]>((acc, a, i) => {
      if (a === "--include-directories") acc.push(i);
      return acc;
    }, []);
    expect(dirIndices).toHaveLength(2);
    expect(args[dirIndices[0] + 1]).toBe("/dir1");
    expect(args[dirIndices[1] + 1]).toBe("/dir2");
  });

  it("prompt comes last after --prompt", () => {
    const args = buildArgs("my prompt", { model: "gemini-2.0-flash" });
    const promptFlagIdx = args.indexOf("--prompt");
    expect(promptFlagIdx).toBeGreaterThan(0);
    expect(args[promptFlagIdx + 1]).toBe("my prompt");
    expect(args.length).toBe(promptFlagIdx + 2);
  });
});
