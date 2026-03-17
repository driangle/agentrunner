---
title: "Publish Python library to PyPI"
id: "01kky7q37"
status: pending
priority: low
type: chore
tags: ["release", "pypi"]
created: "2026-03-17"
---

# Publish Python library to PyPI

## Objective

Publish the Python agentrunner library to PyPI so users can install it via `pip install agentrunner` (or the chosen package name). This includes configuring the package metadata, build system, and ensuring the distribution is correct.

## Tasks

- [ ] Choose and verify package name availability on PyPI
- [ ] Configure `pyproject.toml` with proper metadata (name, version, description, license, authors, repository URL, classifiers, keywords)
- [ ] Ensure the build system is configured (e.g., hatchling, setuptools, or poetry)
- [ ] Add a `py.typed` marker file for PEP 561 type stub support
- [ ] Build the distribution with `python -m build` (produces sdist and wheel)
- [ ] Verify package contents with `twine check dist/*`
- [ ] Publish to TestPyPI first to verify (`twine upload --repository testpypi dist/*`)
- [ ] Publish initial version (0.1.0) to PyPI with `twine upload dist/*`
- [ ] Verify installation works in a fresh virtualenv

## Acceptance Criteria

- Package is publicly available on pypi.org
- `pip install <package-name>` works and provides correct type hints
- Package includes `py.typed` marker for type checker support
- Only library code is included in the distribution (no tests or dev files)
