---
title: "Add Python Ollama example program"
id: "01kky7ard"
status: completed
priority: medium
type: feature
tags: ["python", "ollama", "example"]
dependencies: ["01kkx3f67"]
created: "2026-03-17"
---

# Add Python Ollama example program

## Objective

Add a working example program that demonstrates how to use the Python Ollama runner. The example should show both `run` and `run_stream` usage with asyncio.

## Tasks

- [ ] Create `examples/python/ollama/main.py` with a working example
- [ ] Include `run` and `run_stream` usage with asyncio
- [ ] Add a `pyproject.toml` for the example

## Acceptance Criteria

- Example runs against a local Ollama installation
- Example demonstrates both synchronous and streaming usage
- Example is self-contained with its own `pyproject.toml`
