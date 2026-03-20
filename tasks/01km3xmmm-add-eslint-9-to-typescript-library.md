---
title: "Add ESLint 9 to TypeScript library"
id: "01km3xmmm"
status: completed
priority: medium
type: chore
tags: ["typescript", "linting"]
created: "2026-03-19"
---

# Add ESLint 9 to TypeScript library

## Objective

Set up ESLint 9 for the TypeScript library (`ts/`) using the flat config format. Include a custom rule enforcing a maximum of 200 lines per file. Integrate linting into the existing build and CI pipeline.

## Tasks

- [x] Install `eslint` and `typescript-eslint` as devDependencies in `ts/`
- [x] Create `ts/eslint.config.js` using ESLint 9 flat config format
- [x] Configure TypeScript-aware linting rules
- [x] Add custom rule: `max-lines` set to 200 lines per file
- [x] Add `"lint"` script to `ts/package.json`
- [x] Ensure the Makefile `lint-ts` target runs ESLint
- [x] Fix any existing source files that violate the new rules
- [x] Verify `make check` passes with ESLint integrated

## Acceptance Criteria

- `eslint` v9+ is listed as a devDependency in `ts/package.json`
- `ts/eslint.config.js` exists and uses the flat config format
- `max-lines` rule is configured with a limit of 200 lines per file
- `npm run lint` in `ts/` runs ESLint successfully
- `make lint-ts` runs ESLint and passes
- `make check` passes with no lint errors
- All existing TypeScript source files comply with the configured rules
