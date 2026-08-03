---
title: "Reconcile CLAUDE.md project structure with reality (Java)"
id: "01kz3bppw"
status: pending
priority: medium
type: chore
tags: ["docs", "foundation"]
created: "2026-08-03"
---

# Reconcile CLAUDE.md project structure with reality (Java)

## Description

The `CLAUDE.md` project-structure block lists `java/  # Java library`, and there are 8 task files under `tasks/java/`, but there is no `java/` directory, no `.java` source, and no `check-java` target in the `Makefile` (which `CLAUDE.md`'s own onboarding steps require for every language). The `README.md` is accurate — it lists only Go/TS/Python — so the canonical project doc is the one that's misleading. A newcomer reading `CLAUDE.md` will look for code that doesn't exist.

Make the canonical docs match the actual state, without deleting the planned Java work.

## Tasks

- [ ] In `CLAUDE.md`, mark Java as planned/not-yet-implemented in the project-structure block (or move it to a clearly-labeled "planned" section) so it isn't presented as existing code.
- [ ] Confirm the `README.md` support matrix stays accurate (Java absent or explicitly "Planned").
- [ ] When Java work begins, ensure `check-java`/`build-java`/`lint-java`/`test-java` targets are added to the `Makefile` and wired into `check`, per `CLAUDE.md`'s "adding a new language library" checklist.

## Tasks (checklist)

- [ ] `CLAUDE.md` structure reflects that `java/` is not yet implemented
- [ ] `README.md` matrix verified consistent
- [ ] Planned Java tasks remain intact under `tasks/java/`
