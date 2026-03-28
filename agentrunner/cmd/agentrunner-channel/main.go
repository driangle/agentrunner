// Binary agentrunner-channel is a lightweight MCP server that bridges Unix socket
// IPC from agentrunner libraries to Claude Code's channel system via stdio JSON-RPC.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/driangle/agent-runner/agentrunner/channel"
)

func main() {
	sockPath := os.Getenv("AGENTRUNNER_CHANNEL_SOCK")
	if sockPath == "" {
		fmt.Fprintln(os.Stderr, "AGENTRUNNER_CHANNEL_SOCK environment variable is required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := channel.NewServer(sockPath, os.Stdin, os.Stdout)
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "agentrunner-channel: %v\n", err)
		os.Exit(1)
	}
}
