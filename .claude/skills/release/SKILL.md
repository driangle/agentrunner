---
name: release
description: "Release one package of this monorepo (go, ts, python, or channel) independently: validate, bump its version, tag, and push. Use when the user says /release <package> <version>, or asks to release, publish, or cut a version of the Go, TypeScript, Python, or channel package."
user_invocable: true
argument: "<go|ts|python|channel> [version]"
---

# Release a package

Each package is released on its own `<package>/vX.Y.Z` tag. `RELEASING.md` is the source of truth; read it if anything below is unclear.

## Steps

1. **Parse `$ARGUMENTS`** as `<package> [version]`. The package must be one of `go`, `ts`, `python`, `channel`; ask if it's missing or ambiguous.

2. **Pick the version** if none was given. Find the last release with `git describe --tags --abbrev=0 --match "<package>/v*"` (first release: `--match "v[0-9]*"`), then list what changed with `git log --oneline <last-tag>..HEAD -- <dirs>`, where dirs are `go/`, `ts/`, `python/ ':(exclude)python/channel/'`, or for channel `go/ npm/ python/channel/`. Propose a semver bump (pre-1.0: `feat` or breaking → minor, otherwise patch) and confirm with the user before continuing. If nothing changed, say so and stop.

3. **Preflight.** Be on `main`, up to date with `origin/main` (`git fetch && git status -sb`), with a clean tree. For `ts`, if `npm/channel-*/package.json` has a version newer than the pins in `ts/package.json`, check that `publish-channel` finished for it (`gh run list --workflow publish-channel.yml -L 1`); the ts release pins that version.

4. **Release:**
   ```bash
   ./scripts/release.sh <package> <version>
   ```

5. **Report** the tag and the workflow it triggered (`publish-<package>.yml`), with the run link from `gh run list --workflow publish-<package>.yml -L 1`. After a `channel` release, remind the user that npm users only get the new binary after the next `ts` release.

If the script or workflow fails, report the error and follow the "When a release fails" section of `RELEASING.md`. Never delete a pushed tag or re-tag without the user's go-ahead.
