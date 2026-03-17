---
title: "Publish Java library to Maven Central"
id: "01kky7q38"
status: pending
priority: low
type: chore
tags: ["release", "maven"]
created: "2026-03-17"
---

# Publish Java library to Maven Central

## Objective

Publish the Java agentrunner library to Maven Central so users can add it as a dependency via Maven or Gradle coordinates. This includes configuring the POM metadata, signing artifacts, and completing the Maven Central publishing process.

## Tasks

- [ ] Choose and verify groupId/artifactId (e.g., `io.github.<org>:agentrunner`)
- [ ] Register a namespace on Maven Central (via central.sonatype.com)
- [ ] Configure `pom.xml` or `build.gradle` with required metadata (groupId, artifactId, version, name, description, URL, license, developers, SCM)
- [ ] Set up GPG signing for artifacts
- [ ] Configure the Maven Central publishing plugin (e.g., `maven-publish` + `signing` for Gradle, or `nexus-staging-maven-plugin` for Maven)
- [ ] Ensure Javadoc and sources JARs are generated alongside the main JAR
- [ ] Publish initial version (0.1.0) to Maven Central
- [ ] Verify the artifact appears on search.maven.org
- [ ] Verify dependency resolution works in a fresh Maven/Gradle project

## Acceptance Criteria

- Artifact is publicly available on Maven Central
- Adding the dependency coordinates to a `pom.xml` or `build.gradle` resolves correctly
- Published artifacts include main JAR, sources JAR, and Javadoc JAR
- All artifacts are GPG-signed
