---
title: "CI workflow to publish Python to PyPI on release"
id: "01kky7rkf"
status: completed
priority: low
type: chore
tags: ["ci", "release"]
created: "2026-03-17"
---

# CI workflow to publish Python to PyPI on release

## Objective

Create a GitHub Actions workflow that automatically builds and publishes the Python library to PyPI when a Python-specific release tag is pushed. Use a tag convention like `python/v*` (e.g., `python/v0.1.0`) to enable independent releases.

## Tasks

- [x] Create `.github/workflows/publish-python.yml` GitHub Actions workflow
- [x] Trigger on push of tags matching `python/v*` pattern
- [x] Add job steps: checkout, setup Python, install build tools, run `make check-python`, build sdist and wheel, publish to PyPI
- [x] Use PyPI trusted publishing (OIDC) or configure `PYPI_TOKEN` as a repository secret
- [x] Extract version from the git tag and verify it matches the version in `pyproject.toml`
- [x] Use `twine check` to validate the distribution before uploading
- [ ] Test the workflow against TestPyPI first
- [x] Document the release process in the Python library README

## Acceptance Criteria

- Pushing a tag like `python/v0.1.0` triggers the workflow and publishes to PyPI
- The workflow runs `make check-python` before publishing (fails fast on lint/test errors)
- Publishing one library does not trigger publishing of other libraries
- Both sdist and wheel are uploaded to PyPI

## Notes

Uses `python/v*` tags (matching the `python/` directory) instead of `py/v*`, and the existing `PYPI_TOKEN` secret rather than trusted publishing. TestPyPI was skipped; the same token and upload path already published 0.0.2 via the legacy workflow. PyPI already has 0.1.0 (published before the repo's 0.0.x versions), so the next Python release should be >= 0.1.1.
