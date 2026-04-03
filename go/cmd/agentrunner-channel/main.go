// Binary agentrunner-channel is a lightweight MCP server that bridges Unix socket
// IPC from agentrunner libraries to Claude Code's channel system via stdio JSON-RPC.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/driangle/agentrunner/go/channel"
)

func parseLogLevel() slog.Level {
	switch strings.ToLower(os.Getenv("AGENTRUNNER_CHANNEL_LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	sockPath := os.Getenv("AGENTRUNNER_CHANNEL_SOCK")
	if sockPath == "" {
		fmt.Fprintln(os.Stderr, "AGENTRUNNER_CHANNEL_SOCK environment variable is required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var logger *slog.Logger
	if logPath := os.Getenv("AGENTRUNNER_CHANNEL_LOG"); logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "agentrunner-channel: open log file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: parseLogLevel()}))
	}

	srv := channel.NewServer(sockPath, os.Stdin, os.Stdout, logger)
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "agentrunner-channel: %v\n", err)
		os.Exit(1)
	}
}
