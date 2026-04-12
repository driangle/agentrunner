// Binary agentrunner-channel is a lightweight MCP server that bridges Unix socket
// IPC from agentrunner libraries to Claude Code's channel system via stdio JSON-RPC.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/driangle/agentrunner/go/channel"
)

// printVersion prints the server version and, when available, VCS build
// info (git commit, build time). Works without AGENTRUNNER_CHANNEL_SOCK.
func printVersion() {
	fmt.Printf("%s %s\n", channel.ServerName, channel.ServerVersion)
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	var rev, tm, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			tm = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if rev != "" {
		short := rev
		if len(short) > 12 {
			short = short[:12]
		}
		dirty := ""
		if modified == "true" {
			dirty = "-dirty"
		}
		fmt.Printf("commit: %s%s\n", short, dirty)
	}
	if tm != "" {
		fmt.Printf("built:  %s\n", tm)
	}
	if info.GoVersion != "" {
		fmt.Printf("go:     %s\n", info.GoVersion)
	}
}

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
	// Version flag must be handled before the env-var check so users can
	// verify the installed binary without any environment setup.
	for _, a := range os.Args[1:] {
		if a == "--version" || a == "-v" || a == "-version" {
			printVersion()
			return
		}
	}

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
