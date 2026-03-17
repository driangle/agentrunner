---
title: "Publish TypeScript library to npm"
id: "01kky7q1f"
status: pending
priority: low
type: chore
tags: ["release", "npm"]
created: "2026-03-17"
---

# Publish TypeScript library to npm

## Objective

Publish the TypeScript agentrunner library as a public npm package so users can install it via `npm install agentrunner` (or the chosen package name). This includes configuring the package for publication, setting up versioning, and ensuring the build output is correct for consumers.

## Tasks

- [ ] Choose and verify package name availability on npm
- [ ] Configure `package.json` with proper metadata (name, version, description, license, repository, keywords, author, exports, files)
- [ ] Ensure the build produces correct ESM/CJS output with type declarations
- [ ] Add `prepublishOnly` script that runs `make check-ts` before publishing
- [ ] Create `.npmignore` or configure `files` field to exclude tests, dev configs, and source maps
- [ ] Verify the package contents with `npm pack --dry-run`
- [ ] Publish initial version (0.1.0) to npm with `npm publish --access public`
- [ ] Verify installation works in a fresh project

## Acceptance Criteria

- Package is publicly available on npmjs.com
- `npm install <package-name>` works and provides correct TypeScript types
- Package only includes built output and type declarations (no test files, no source maps)
- README is included in the published package
