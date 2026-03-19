# agentrunner

Language-native libraries for programmatically invoking AI coding agent CLIs.

## Supported Runners

| Runner | Go | TypeScript | Python | Java |
|--------|----|------------|--------|------|
| Claude Code (`claude`) | :white_check_mark: | :white_check_mark: | :white_check_mark: | Planned |
| Gemini CLI (`gemini`) | Planned | Planned | Planned | Planned |
| Codex CLI (`codex`) | Planned | Planned | Planned | Planned |
| Ollama (`ollama`) | :white_check_mark: | :white_check_mark: | Planned | Planned |

## Libraries

| Language   | Path        | Package |
|------------|-------------|---------|
| Go         | [`go/`](go/)       | `agentrunner` |
| TypeScript | [`ts/`](ts/)       | `agentrunner` |
| Python     | [`python/`](python/) | `agentrunner` |
| Java       | [`java/`](java/)     | `agentrunner` |

## Overview

Each library implements a common `Runner` interface across all supported AI agent CLIs. This lets you swap between Claude Code, Gemini, and Codex with a consistent API for process management, streaming output, and result parsing.
