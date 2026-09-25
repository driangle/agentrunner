---
title: "CI workflow to publish TypeScript to npm on release"
id: "01kky7rj5"
status: in-progress
priority: low
type: chore
tags: ["ci", "release"]
created: "2026-03-17"
---

# CI workflow to publish TypeScript to npm on release

## Objective

Create a GitHub Actions workflow that automatically publishes the TypeScript library to npm when a TypeScript-specific release is created. Each library should be independently releasable using a tag convention like `ts/v*` (e.g., `ts/v0.1.0`).

## Tasks

- [x] Create `.github/workflows/publish-ts.yml` GitHub Actions workflow
- [x] Trigger on push of tags matching `ts/v*` pattern
- [x] Add job steps: checkout, setup Node.js, install dependencies, run `make check-ts`, build, publish to npm
- [ ] Configure `NPM_TOKEN` as a repository secret for authentication
- [x] Use `npm publish --access public` with the token
- [x] Extract version from the git tag and verify it matches `package.json` version
- [ ] Test the workflow with a dry run (e.g., `npm publish --dry-run`)
- [x] Document the release process in the TypeScript library README

## Acceptance Criteria

- Pushing a tag like `ts/v0.1.0` triggers the workflow and publishes to npm
- The workflow runs `make check-ts` before publishing (fails fast on lint/test errors)
- Publishing one library does not trigger publishing of other libraries
- The workflow fails clearly if the npm token is missing or invalid

## Notes

Workflow, tag/version check and docs (`RELEASING.md`, `ts/README.md`) are done. Remaining:
- `NPM_TOKEN` is not set as a repository secret (`gh secret list` shows only `PYPI_TOKEN`), so the publish step will fail until it is added. The legacy v0.0.2 run failed on the same step.
- Not yet exercised end to end; the first `ts/v*` release is the real test.
