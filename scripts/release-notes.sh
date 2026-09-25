#!/usr/bin/env bash
# Print release notes for a package tag, listing only commits that touched
# that package's directories since its previous release.
#
# Usage: ./scripts/release-notes.sh <tag>
#   tag - a package tag such as ts/v0.3.0, python/v0.2.1, go/v0.1.0, channel/v0.0.2

set -euo pipefail

tag="$1"
lang="${tag%%/v*}"

case "$lang" in
  go)      paths=(go/) ;;
  ts)      paths=(ts/) ;;
  python)  paths=(python/ ':(exclude)python/channel/') ;;
  channel) paths=(go/ npm/ python/channel/) ;;
  *)
    echo "Error: cannot derive package from tag '$tag'." >&2
    exit 1
    ;;
esac

# Previous release of this package; for a first release, fall back to the
# legacy lockstep tags (vX.Y.Z) that predate per-package tags.
prev=$(git describe --tags --abbrev=0 --match "${lang}/v*" "${tag}^" 2>/dev/null \
  || git describe --tags --abbrev=0 --match "v[0-9]*" "${tag}^" 2>/dev/null \
  || true)

if [ -n "$prev" ]; then
  range="${prev}..${tag}"
else
  range="$tag"
fi

echo "## Changes"
echo
changes=$(git log --no-merges --pretty='- %s (%h)' "$range" -- "${paths[@]}")
echo "${changes:-- No changes to ${paths[*]}}"

if [ -n "$prev" ] && [ -n "${GITHUB_REPOSITORY:-}" ]; then
  echo
  echo "**Full diff**: https://github.com/${GITHUB_REPOSITORY}/compare/${prev}...${tag}"
fi
