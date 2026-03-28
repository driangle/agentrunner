#!/usr/bin/env bash
# Synchronize the channel binary version across all npm platform packages
# and the PyPI channel package.
#
# Usage: ./scripts/sync-channel-version.sh <version>
#   version - semver version without 'v' prefix (e.g., 0.2.0)

set -euo pipefail

version="$1"

echo "==> Syncing channel version to $version..."

# Update npm platform packages
for pkg_dir in npm/channel-*/; do
  if [ -f "$pkg_dir/package.json" ]; then
    # Use a temp file for portability (macOS sed -i requires backup extension)
    tmp=$(mktemp)
    jq --arg v "$version" '.version = $v' "$pkg_dir/package.json" > "$tmp"
    mv "$tmp" "$pkg_dir/package.json"
    echo "  Updated $pkg_dir/package.json"
  fi
done

# Update optionalDependencies in ts/package.json
if [ -f "ts/package.json" ]; then
  tmp=$(mktemp)
  jq --arg v "$version" '
    .optionalDependencies |= with_entries(
      if .key | startswith("@agentrunner/channel-") then .value = $v else . end
    )
  ' ts/package.json > "$tmp"
  mv "$tmp" ts/package.json
  echo "  Updated ts/package.json optionalDependencies"
fi

# Update PyPI channel package version
if [ -f "python/channel/pyproject.toml" ]; then
  sed -i.bak "s/^version = \".*\"/version = \"$version\"/" python/channel/pyproject.toml
  rm -f python/channel/pyproject.toml.bak
  echo "  Updated python/channel/pyproject.toml"
fi

echo "==> Done. All channel packages set to $version"
