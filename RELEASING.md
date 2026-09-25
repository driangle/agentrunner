# Releasing

Each package in this monorepo is versioned and released independently. A release is a git tag of the form `<package>/vX.Y.Z`; pushing it triggers that package's publish workflow and nothing else. Version numbers do not correspond across packages — `ts` 0.4.0 and `python` 0.2.1 can be current at the same time.

| Package | Tag | Version source | Workflow | Publishes to |
|---|---|---|---|---|
| Go library | `go/vX.Y.Z` | the tag | `publish-go.yml` | Go module proxy |
| TypeScript library | `ts/vX.Y.Z` | `ts/package.json` | `publish-ts.yml` | npm (`@driangle/agentrunner`) |
| Python library | `python/vX.Y.Z` | `python/pyproject.toml` | `publish-python.yml` | PyPI (`driangle-agentrunner`) |
| Channel binary | `channel/vX.Y.Z` | `npm/channel-*/package.json`, `python/channel/pyproject.toml` | `publish-channel.yml` | npm platform packages, PyPI platform wheels, GitHub Release assets |

## Prerequisites

- **Locally:** push access to `main`, plus `git`, `make`, `jq`, and the toolchain for the package's `make check-<lang>` (`npm` is also needed for `ts` releases).
- **Repository secrets:** `NPM_TOKEN` (publish rights on the `@driangle` scope) and `PYPI_TOKEN`. Go needs no secret; the tag is enough.

## Choosing a version

Use [semver](https://semver.org). While a package is below 1.0, bump the **minor** version for new features or breaking changes and the **patch** version for fixes. To see what changed since the last release:

```bash
git describe --tags --abbrev=0 --match "ts/v*"        # last ts release
./scripts/release-notes.sh ts/v0.0.3                   # notes for an existing tag
git log --oneline ts/v0.0.3..HEAD -- ts/               # unreleased changes
```

## Cutting a release

From a clean `main` that is up to date with `origin/main`:

```bash
./scripts/release.sh <go|ts|python|channel> <version>
```

The script:

1. Checks the working tree is clean, the version is valid semver, and the tag doesn't exist.
2. Runs that package's `make check-<lang>` (`check-go` for channel).
3. Bumps the version file(s) and commits `chore(release): <lang> v<version>` (Go has no version file, so no commit).
4. Tags `<lang>/v<version>` and pushes the commit and tag together (`git push --atomic`).

Then watch the workflow: `gh run watch $(gh run list --workflow publish-<lang>.yml -L 1 --json databaseId -q '.[0].databaseId')`.

In Claude Code, `/release <package> [version]` does the same, proposing a version from the unreleased changes if you omit it.

## What each workflow does

Each workflow runs only the checks for its own package, so a failing test in one language never blocks a release of another.

- **publish-go** runs `make check-go`, confirms the module can be fetched from `proxy.golang.org`, and creates a GitHub Release.
- **publish-ts** runs `make check-ts`, checks the tag matches `ts/package.json`, runs `npm publish`, and creates a GitHub Release.
- **publish-python** runs `make check-python`, checks the tag matches `python/pyproject.toml`, validates the build with `twine check`, uploads to PyPI, and creates a GitHub Release.
- **publish-channel** runs `make check-go`, checks the tag matches the channel package versions, cross-compiles, publishes the npm and PyPI platform packages, and creates a GitHub Release with the binaries attached.

Release notes come from `scripts/release-notes.sh`: the commits that touched the package's directories since its previous tag (for a first release, since the legacy `v*` tag), plus a compare link.

## Channel binary and the TypeScript library

`ts/package.json` pins the channel platform packages in `optionalDependencies`. A channel release does not touch those pins, because the new packages aren't on npm until its workflow finishes. Instead, every `ts` release pins them to the current channel version and resyncs `package-lock.json`. To ship a new binary to npm users:

```bash
./scripts/release.sh channel 0.0.2
# wait for publish-channel to finish
./scripts/release.sh ts 0.0.4
```

## When a release fails

**The script fails before pushing** (checks fail, push rejected because `main` moved). Nothing is published. Undo the local tag and bump commit, pull, and rerun:

```bash
git tag -d <lang>/vX.Y.Z
git reset --hard HEAD~1   # only if the script made a chore(release) commit
git pull
```

**The workflow fails before publishing** (checks or tag/version mismatch). Nothing is published. Delete the tag, fix the problem on `main`, and rerun the script with the same version (the bump is already committed, so it only re-tags):

```bash
git push origin :refs/tags/<lang>/vX.Y.Z && git tag -d <lang>/vX.Y.Z
```

**The workflow fails after publishing** (e.g. only the GitHub Release step failed). npm and PyPI versions can never be re-uploaded, so don't reuse the version. Rerun the failed jobs with `gh run rerun <run-id> --failed`. If a publish was partial (some channel platforms uploaded, others not), release the next patch version instead.

**Go tags are permanent.** Once `proxy.golang.org` has fetched a `go/v*` tag it caches that commit forever, so never move or re-push a Go tag after `publish-go` has run. Release the next patch version instead.

## Adding a new package

1. Add a `publish-<lang>.yml` workflow triggered by `<lang>/v*` that runs only `make check-<lang>`, verifies the tag against the version file, publishes, and creates the GitHub Release with `scripts/release-notes.sh`.
2. Add a case for the package to `scripts/release.sh` (check target and bump step) and `scripts/release-notes.sh` (directories).
3. Add the package to the table above and to the `/release` skill's package list (`.claude/skills/release/SKILL.md`).

## Legacy tags

`v0.0.1` and `v0.0.2` date from before the split, when a single tag released every package in lockstep. They are kept for history. Never push a bare `vX.Y.Z` tag again; no workflow listens for it.
