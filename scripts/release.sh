#!/usr/bin/env bash
# Release one package independently: validate, bump its version, tag, and push.
#
# Usage: ./scripts/release.sh <lang> <version>
#   lang    - go | ts | python | channel
#   version - semver version without 'v' prefix (e.g., 0.1.0)
#
# Pushes the tag <lang>/v<version>, which triggers .github/workflows/publish-<lang>.yml.

set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <go|ts|python|channel> <version>"
  exit 1
fi

lang="$1"
version="$2"

case "$lang" in
  go | channel) check_target="check-go" ;;
  ts)           check_target="check-ts" ;;
  python)       check_target="check-python" ;;
  *)
    echo "Error: unknown lang '$lang' (expected go, ts, python, or channel)."
    exit 1
    ;;
esac

tag="${lang}/v${version}"

if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "Error: '$version' is not a valid semver version (e.g., 0.1.0)."
  exit 1
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "Error: working tree is not clean. Commit or stash changes first."
  exit 1
fi

if git rev-parse "$tag" >/dev/null 2>&1; then
  echo "Error: tag $tag already exists."
  exit 1
fi

bump_version() {
  case "$lang" in
    go) ;; # Go versions live only in the tag.
    ts)
      # Pin the channel binary to the latest channel release, then resync the lockfile.
      channel_version=$(jq -r .version npm/channel-darwin-arm64/package.json)
      tmp=$(mktemp)
      jq --arg v "$version" --arg cv "$channel_version" '
        .version = $v
        | .optionalDependencies |= with_entries(
            if .key | startswith("@driangle/agentrunner-channel-") then .value = $cv else . end
          )
      ' ts/package.json > "$tmp"
      mv "$tmp" ts/package.json
      (cd ts && npm install --package-lock-only --silent)
      ;;
    python)
      sed -i.bak "s/^version = \".*\"/version = \"$version\"/" python/pyproject.toml
      rm -f python/pyproject.toml.bak
      ;;
    channel)
      ./scripts/sync-channel-version.sh "$version"
      ;;
  esac
}

echo "==> Validating $lang (make $check_target)..."
make "$check_target"

echo "==> Bumping $lang version to $version..."
bump_version

if [ -n "$(git status --porcelain)" ]; then
  git add -A
  git commit -m "chore(release): $lang v$version"
fi

echo "==> Creating tag $tag..."
git tag "$tag"

echo "==> Pushing $tag to origin..."
git push --atomic origin HEAD "$tag"

echo "==> Released $tag"
