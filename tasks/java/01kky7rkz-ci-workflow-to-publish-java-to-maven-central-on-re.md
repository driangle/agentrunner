---
title: "CI workflow to publish Java to Maven Central on release"
id: "01kky7rkz"
status: pending
priority: low
type: chore
tags: ["ci", "release"]
created: "2026-03-17"
---

# CI workflow to publish Java to Maven Central on release

## Objective

Create a GitHub Actions workflow that automatically builds, signs, and publishes the Java library to Maven Central when a Java-specific release tag is pushed. Use a tag convention like `java/v*` (e.g., `java/v0.1.0`) to enable independent releases.

## Tasks

- [ ] Create `.github/workflows/publish-java.yml` GitHub Actions workflow
- [ ] Trigger on push of tags matching `java/v*` pattern
- [ ] Add job steps: checkout, setup JDK, run `make check-java`, build, sign artifacts with GPG, publish to Maven Central
- [ ] Configure repository secrets: `MAVEN_USERNAME`, `MAVEN_PASSWORD`, `GPG_PRIVATE_KEY`, `GPG_PASSPHRASE`
- [ ] Import GPG key in the workflow for artifact signing
- [ ] Extract version from the git tag and verify it matches the build file version
- [ ] Publish using the Maven Central publishing plugin (Sonatype OSSRH or Central Portal)
- [ ] Document the release process in the Java library README

## Acceptance Criteria

- Pushing a tag like `java/v0.1.0` triggers the workflow and publishes to Maven Central
- The workflow runs `make check-java` before publishing (fails fast on lint/test errors)
- Publishing one library does not trigger publishing of other libraries
- All artifacts (JAR, sources, Javadoc) are signed and uploaded
