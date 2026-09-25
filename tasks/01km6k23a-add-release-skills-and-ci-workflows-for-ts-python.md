---
title: "Add release skills and CI workflows for TS, Python, and Java"
id: "01km6k23a"
status: completed
priority: low
type: chore
tags: ["ci", "release"]
created: "2026-03-20"
---

# Add release skills and CI workflows for TS, Python, and Java

## Objective

Create `/release-ts`, `/release-py`, and `/release-java` skills and matching `publish-*.yml` GitHub Actions workflows, following the pattern established by `/release-go` and `publish-go.yml`. Each skill uses the shared `scripts/release.sh` script.

## Tasks

- [x] ~~Create `.claude/skills/release-ts/SKILL.md`~~ — replaced by a single `/release <package> [version]` skill (`.claude/skills/release/SKILL.md`)
- [x] ~~Create `.claude/skills/release-py/SKILL.md`~~ — covered by `/release python`
- [ ] ~~Create `.claude/skills/release-java/SKILL.md`~~ — moved to task 01kky7rkz (no `java/` library exists yet)
- [x] Update `scripts/release.sh` to support a version-bump step per language (update `version` in `package.json` for TS, `pyproject.toml` for Python, `pom.xml` for Java; Go needs no version in code). The script should update the file, commit, then tag.
- [x] Create `.github/workflows/publish-ts.yml` — triggers on `ts/v*` tags, runs `make check-ts`
- [x] Create `.github/workflows/publish-python.yml` — triggers on `python/v*` tags, runs `make check-python`
- [ ] ~~Create `.github/workflows/publish-java.yml`~~ — moved to task 01kky7rkz

## Acceptance Criteria

- Each skill is user-invocable as `/release-ts`, `/release-py`, `/release-java`
- Each workflow triggers only on its own tag prefix
- All skills use the shared `scripts/release.sh` script

## Notes

Implemented as a single `/release <package> [version]` skill over `scripts/release.sh <lang> <version>` rather than one skill per language; see `RELEASING.md`. The lockstep `release.yml` / `.release.conf` were removed. Java items are deferred to 01kky7rkz until a Java library exists.
