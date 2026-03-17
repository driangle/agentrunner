---
title: "CI workflow to publish Python to PyPI on release"
id: "01kky7rkf"
status: pending
priority: low
type: chore
tags: ["ci", "release"]
created: "2026-03-17"
---

# CI workflow to publish Python to PyPI on release

## Objective

Create a GitHub Actions workflow that automatically builds and publishes the Python library to PyPI when a Python-specific release tag is pushed. Use a tag convention like `py/v*` (e.g., `py/v0.1.0`) to enable independent releases.

## Tasks

- [ ] Create `.github/workflows/publish-py.yml` GitHub Actions workflow
- [ ] Trigger on push of tags matching `py/v*` pattern
- [ ] Add job steps: checkout, setup Python, install build tools, run `make check-python`, build sdist and wheel, publish to PyPI
- [ ] Use PyPI trusted publishing (OIDC) or configure `PYPI_TOKEN` as a repository secret
- [ ] Extract version from the git tag and verify it matches the version in `pyproject.toml`
- [ ] Use `twine check` to validate the distribution before uploading
- [ ] Test the workflow against TestPyPI first
- [ ] Document the release process in the Python library README

## Acceptance Criteria

- Pushing a tag like `py/v0.1.0` triggers the workflow and publishes to PyPI
- The workflow runs `make check-python` before publishing (fails fast on lint/test errors)
- Publishing one library does not trigger publishing of other libraries
- Both sdist and wheel are uploaded to PyPI
