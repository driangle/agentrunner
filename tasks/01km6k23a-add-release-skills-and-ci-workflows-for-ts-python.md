---
title: "Add release skills and CI workflows for TS, Python, and Java"
id: "01km6k23a"
status: pending
priority: low
type: chore
tags: ["ci", "release"]
created: "2026-03-20"
---

# Add release skills and CI workflows for TS, Python, and Java

## Objective

Create `/release-ts`, `/release-py`, and `/release-java` skills and matching `publish-*.yml` GitHub Actions workflows, following the pattern established by `/release-go` and `publish-go.yml`. Each skill uses the shared `scripts/release.sh` script.

## Tasks

- [ ] Create `.claude/skills/release-ts/SKILL.md` — invokes `./scripts/release.sh ts check-ts ts "$ARGUMENTS"`
- [ ] Create `.claude/skills/release-py/SKILL.md` — invokes `./scripts/release.sh python check-python python "$ARGUMENTS"`
- [ ] Create `.claude/skills/release-java/SKILL.md` — invokes `./scripts/release.sh java check-java java "$ARGUMENTS"`
- [ ] Update `scripts/release.sh` to support a version-bump step per language (update `version` in `package.json` for TS, `pyproject.toml` for Python, `pom.xml` for Java; Go needs no version in code). The script should update the file, commit, then tag.
- [ ] Create `.github/workflows/publish-ts.yml` — triggers on `ts/v*` tags, runs `make check-ts`
- [ ] Create `.github/workflows/publish-python.yml` — triggers on `python/v*` tags, runs `make check-python`
- [ ] Create `.github/workflows/publish-java.yml` — triggers on `java/v*` tags, runs `make check-java`

## Acceptance Criteria

- Each skill is user-invocable as `/release-ts`, `/release-py`, `/release-java`
- Each workflow triggers only on its own tag prefix
- All skills use the shared `scripts/release.sh` script
