# Getting Started

agentrunner provides language-native libraries for programmatically invoking AI coding agents. Each library implements a common [Runner interface](/guide/runner-interface) that lets you swap agents without changing your code.

## Installation

::: code-group

```sh [Go]
go get github.com/anthropics/agentrunner/go
```

```sh [TypeScript]
npm install agentrunner
```

```sh [Python]
pip install agentrunner
```

:::

## Prerequisites

You need the CLI for whichever runner you plan to use:

| Runner | Requirement |
|--------|-------------|
| Claude Code | `claude` CLI >= 1.0.12 |
| Ollama | Ollama server running at `localhost:11434` |

## Quick Example

::: code-group

```go [Go]
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/anthropics/agentrunner/go/claudecode"
)

func main() {
	runner := claudecode.NewRunner()
	result, err := runner.Run(context.Background(), "What is 2+2?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Text)
}
```

```ts [TypeScript]
import { createClaudeRunner } from "agentrunner/claudecode";

const runner = createClaudeRunner();
const result = await runner.run("What is 2+2?");
console.log(result.text);
```

```python [Python]
import asyncio
from agentrunner import ClaudeRunner

async def main():
    runner = ClaudeRunner()
    result = await runner.run("What is 2+2?")
    print(result.text)

asyncio.run(main())
```

:::

## Streaming

All runners support real-time message streaming:

::: code-group

```go [Go]
session, err := runner.Start(ctx, "Explain this codebase")
if err != nil {
    log.Fatal(err)
}
for msg := range session.Messages {
    if text := msg.Text(); text != "" {
        fmt.Print(text)
    }
}
result, err := session.Result()
```

```ts [TypeScript]
const session = runner.start("Explain this codebase");
for await (const msg of session.messages) {
  const text = messageText(msg);
  if (text) process.stdout.write(text);
}
const result = await session.result;
```

```python [Python]
session = runner.start("Explain this codebase")
async for msg in session:
    if msg.text:
        print(msg.text, end="")
result = await session.result
```

:::

## Next Steps

- [Runner Interface](/guide/runner-interface) — understand the common interface all runners share
- [Channels](/guide/channels) — two-way communication with running agents
- [Go Library](/libraries/go) — Go-specific API reference
- [TypeScript Library](/libraries/typescript) — TypeScript-specific API reference
- [Python Library](/libraries/python) — Python-specific API reference
