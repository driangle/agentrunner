#!/usr/bin/env bash
# Release a library by running checks, tagging, and pushing.
#
# Usage: ./scripts/release.sh <lib> <check-target> <tag-prefix> <version>
#   lib          - library name (for display)
#   check-target - make target to validate (e.g., check-go)
#   tag-prefix   - tag prefix matching the subdirectory (e.g., agentrunner)
#   version      - semver version without 'v' prefix (e.g., 0.1.0)

set -euo pipefail

lib="$1"
check_target="$2"
tag_prefix="$3"
version="$4"

tag="${tag_prefix}/v${version}"

# Ensure we're on a clean working tree
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: working tree is not clean. Commit or stash changes first."
  exit 1
fi

# Ensure the tag doesn't already exist
if git rev-parse "$tag" >/dev/null 2>&1; then
  echo "Error: tag $tag already exists."
  exit 1
fi

echo "==> Validating $lib (make $check_target)..."
make "$check_target"

echo "==> Creating tag $tag..."
git tag "$tag"

echo "==> Pushing tag $tag to origin..."
git push origin "$tag"

echo "==> Released $lib $tag"
