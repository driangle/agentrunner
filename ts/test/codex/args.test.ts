import { describe, it, expect } from "vitest";
import { buildArgs } from "../../src/codex/args.js";

describe("buildArgs", () => {
  it("minimal args", () => {
    const args = buildArgs("hello world");
    expect(args).toEqual(["exec", "--json", "--", "hello world"]);
  });

  it("common options", () => {
    const args = buildArgs("hello", {
      model: "gpt-4o",
      workingDir: "/tmp/test",
    });
    expect(args).toContain("--model");
    expect(args).toContain("gpt-4o");
    expect(args).toContain("--cd");
    expect(args).toContain("/tmp/test");
  });

  it("dangerously skip permissions", () => {
    const args = buildArgs("hello", { dangerouslySkipPermissions: true });
    expect(args).toContain("--dangerously-bypass-approvals-and-sandbox");
  });

  it("codex-specific options", () => {
    const args = buildArgs("hello", {
      sandbox: "workspace-write",
      approval: "never",
      outputSchema: "/path/to/schema.json",
      images: ["/img/a.png", "/img/b.png"],
      profile: "my-profile",
      fullAuto: true,
      ephemeral: true,
      search: true,
      addDirs: ["/extra/dir1", "/extra/dir2"],
    });

    expect(args).toContain("--sandbox");
    expect(args).toContain("workspace-write");
    expect(args).toContain("--ask-for-approval");
    expect(args).toContain("never");
    expect(args).toContain("--output-schema");
    expect(args).toContain("/path/to/schema.json");
    expect(args).toContain("--profile");
    expect(args).toContain("my-profile");
    expect(args).toContain("--full-auto");
    expect(args).toContain("--ephemeral");
    expect(args).toContain("--search");

    // Images: --image /img/a.png --image /img/b.png
    const imageIndices = args.reduce<number[]>((acc, a, i) => {
      if (a === "--image") acc.push(i);
      return acc;
    }, []);
    expect(imageIndices).toHaveLength(2);
    expect(args[imageIndices[0] + 1]).toBe("/img/a.png");
    expect(args[imageIndices[1] + 1]).toBe("/img/b.png");

    // AddDirs: --add-dir /extra/dir1 --add-dir /extra/dir2
    const addDirIndices = args.reduce<number[]>((acc, a, i) => {
      if (a === "--add-dir") acc.push(i);
      return acc;
    }, []);
    expect(addDirIndices).toHaveLength(2);
    expect(args[addDirIndices[0] + 1]).toBe("/extra/dir1");
    expect(args[addDirIndices[1] + 1]).toBe("/extra/dir2");
  });

  it("resume session", () => {
    const args = buildArgs("hello", { resume: "sess-123" });
    expect(args).toContain("resume");
    expect(args).toContain("sess-123");
    // resume should come before -- prompt
    const resumeIdx = args.indexOf("resume");
    const dashIdx = args.indexOf("--");
    expect(resumeIdx).toBeLessThan(dashIdx);
  });

  it("prompt comes last after --", () => {
    const args = buildArgs("my prompt", { model: "gpt-4o" });
    const dashIdx = args.indexOf("--");
    expect(dashIdx).toBeGreaterThan(0);
    expect(args[dashIdx + 1]).toBe("my prompt");
    expect(args.length).toBe(dashIdx + 2);
  });
});
