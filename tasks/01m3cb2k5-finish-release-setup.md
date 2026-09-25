---
title: "Finish release setup and cut first per-package releases"
id: "01m3cb2k5"
status: pending
priority: high
type: chore
tags: ["ci", "release"]
created: "2026-09-25"
---

# Finish release setup and cut first per-package releases

## Objective

Releases now use per-package tags (`go/`, `ts/`, `python/`, `channel/`; see `RELEASING.md`), but none of the new `publish-*` workflows have run yet, and a few one-time setup gaps remain. This task walks through closing those gaps and cutting the first release of each package, in dependency order, so that every workflow is proven end to end.

Current state (2026-09-25):

| Package | Repo version | Published | Notes |
|---|---|---|---|
| Go | — | `go/v0.0.2` on proxy | Up to date; no release needed unless `go/` changes |
| TypeScript | `0.0.2` | npm `0.0.1`, `0.0.2` | `NPM_TOKEN` secret missing; pins channel `0.0.1`, which isn't on npm |
| Python | `0.0.2` | PyPI `0.0.1`, `0.0.2`, **`0.1.0`** | Next version must be `>= 0.1.1` or pip keeps installing `0.1.0` |
| Channel | `0.0.1` | **Never published** (npm 404, PyPI `agentrunner-channel` 404) | TS users currently get no channel binary |

## Tasks

### 1. One-time setup

- [ ] Push the release-split commit: `git push origin main`
- [ ] Create an npm **automation** token with publish rights on the `@driangle` scope (npmjs.com → Access Tokens), then `gh secret set NPM_TOKEN`
- [ ] Confirm `PYPI_TOKEN` is an **account-scoped** token. The first upload of a new project (`agentrunner-channel`) is rejected by a project-scoped token. If it's scoped to `driangle-agentrunner` only, replace it with an account-scoped one (`gh secret set PYPI_TOKEN`), or add a second token for the channel project.
- [ ] `gh secret list` shows both `NPM_TOKEN` and `PYPI_TOKEN`

### 2. Channel binary (first ever publish)

The files already say `0.0.1`, so the script only tags; no bump commit.

- [ ] `/release channel 0.0.1` (or `./scripts/release.sh channel 0.0.1`)
- [ ] Watch `publish-channel`: `gh run watch $(gh run list --workflow publish-channel.yml -L 1 --json databaseId -q '.[0].databaseId')`
- [ ] Verify: `npm view @driangle/agentrunner-channel-darwin-arm64 version` → `0.0.1` (and the other four platforms)
- [ ] Verify: https://pypi.org/project/agentrunner-channel/ lists five platform wheels
- [ ] Verify: the `channel/v0.0.1` GitHub Release has the five binaries attached

If a platform publish fails after others succeeded, don't retry the same version (npm/PyPI versions are immutable). Fix the cause and release `channel 0.0.2`. See "When a release fails" in `RELEASING.md`.

### 3. TypeScript

Wait until step 2 is green: the ts release pins the channel version and resyncs `package-lock.json` against npm.

- [ ] `/release ts 0.0.3`
- [ ] Check the bump commit: `ts/package.json` is `0.0.3`, the `optionalDependencies` pins are `0.0.1`, and `ts/package-lock.json` now has `node_modules/@driangle/agentrunner-channel-*` entries
- [ ] Watch `publish-ts` until it's green
- [ ] Verify: `npm view @driangle/agentrunner version` → `0.0.3`
- [ ] Smoke test in a scratch dir: `npm install @driangle/agentrunner@0.0.3` installs the matching `agentrunner-channel-<platform>` package
- [ ] Verify: `cd ts && npm ci` now passes locally under npm 11 (it failed before because the channel packages didn't exist)

### 4. Python

- [ ] `/release python 0.1.1` (must be `>= 0.1.1` because of the stray `0.1.0` on PyPI)
- [ ] Watch `publish-python` until it's green
- [ ] Verify: `pip index versions driangle-agentrunner` shows `0.1.1` as the latest
- [ ] Optional: yank `0.1.0` on PyPI if it doesn't match any repo state (PyPI → project → Manage → Release 0.1.0 → Yank)

### 5. Go (optional)

- [ ] Only if `git log --oneline go/v0.0.2..HEAD -- go/` shows changes: `/release go 0.0.3` and confirm `publish-go` is green

### 6. Wrap up

- [ ] Each GitHub Release's notes list only that package's commits (spot-check `ts/v0.0.3` and `python/v0.1.1`)
- [ ] Mark task 01kky7rj5 (TS publish workflow) completed
- [ ] If any step surprised you, add it to "When a release fails" in `RELEASING.md`

## Acceptance Criteria

- `NPM_TOKEN` and `PYPI_TOKEN` are set, and `PYPI_TOKEN` can create new projects
- `publish-channel`, `publish-ts` and `publish-python` have each completed one green run
- Channel `0.0.1` is on npm (5 platform packages) and PyPI (5 wheels)
- `@driangle/agentrunner@0.0.3` is on npm and pulls in the channel binary for the installing platform
- `driangle-agentrunner` `0.1.1` is the latest version on PyPI
- `cd ts && npm ci` passes locally
