---
title: "CI workflow to publish TypeScript to npm on release"
id: "01kky7rj5"
status: pending
priority: low
type: chore
tags: ["ci", "release"]
created: "2026-03-17"
---

# CI workflow to publish TypeScript to npm on release

## Objective

Create a GitHub Actions workflow that automatically publishes the TypeScript library to npm when a TypeScript-specific release is created. Each library should be independently releasable using a tag convention like `ts/v*` (e.g., `ts/v0.1.0`).

## Tasks

- [ ] Create `.github/workflows/publish-ts.yml` GitHub Actions workflow
- [ ] Trigger on push of tags matching `ts/v*` pattern
- [ ] Add job steps: checkout, setup Node.js, install dependencies, run `make check-ts`, build, publish to npm
- [ ] Configure `NPM_TOKEN` as a repository secret for authentication
- [ ] Use `npm publish --access public` with the token
- [ ] Extract version from the git tag and verify it matches `package.json` version
- [ ] Test the workflow with a dry run (e.g., `npm publish --dry-run`)
- [ ] Document the release process in the TypeScript library README

## Acceptance Criteria

- Pushing a tag like `ts/v0.1.0` triggers the workflow and publishes to npm
- The workflow runs `make check-ts` before publishing (fails fast on lint/test errors)
- Publishing one library does not trigger publishing of other libraries
- The workflow fails clearly if the npm token is missing or invalid
