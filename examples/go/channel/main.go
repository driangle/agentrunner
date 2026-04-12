// [Experimental] This example demonstrates two-way channel communication with Claude Code.
// It starts a session with channels enabled, sends a message to Claude via
// the channel, and prints any channel replies from the stream.
//
// IMPORTANT: The channels feature in Claude Code is gated behind a server-side
// feature flag. In -p (print) mode — which agentrunner uses — this flag must be
// enabled on your account. If the MCP server logs show "forwarding channel
// message" but Claude never acts on it, the feature flag is likely not enabled.
// See docs/guide/channels.md for details.
//
// Prerequisites:
//   - Claude Code CLI installed (>= 1.0.12): https://docs.anthropic.com/en/docs/claude-code
//   - Authenticated with `claude login`
//
// Run:
//
//	go run .
//	go run . --claude /path/to/claude
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/driangle/agentrunner/go"
	"github.com/driangle/agentrunner/go/channel"
	"github.com/driangle/agentrunner/go/claudecode"
)

func main() {
	claudeBinary := flag.String("claude", "claude", "path to the Claude Code CLI binary")
	verbose := flag.Bool("verbose", false, "enable debug logging")
	flag.Parse()

	var runnerOpts []claudecode.RunnerOption
	runnerOpts = append(runnerOpts, claudecode.WithBinary(*claudeBinary))
	if *verbose {
		runnerOpts = append(runnerOpts, claudecode.WithLogger(
			slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})),
		))
	}

	if err := run(runnerOpts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(runnerOpts []claudecode.RunnerOption) error {
	ctx := context.Background()
	runner := claudecode.NewRunner(runnerOpts...)

	// Scenario: Claude reviews files while a CI build failure notification
	// arrives via the channel. The prompt gives Claude multi-turn work
	// so the channel message arrives between turns.
	prompt := "Read main.go and go.mod, then give a one-paragraph summary of each. " +
		"Also respond to any incoming channel notifications. Be brief."
	fmt.Printf("Prompt: %s\n", prompt)
	fmt.Println("---")

	session, err := runner.Start(ctx, prompt,
		claudecode.WithChannelEnabled(),
		agentrunner.WithSkipPermissions(),
		agentrunner.WithMaxTurns(10),
		agentrunner.WithTimeout(60*time.Second),
	)
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// sendCh is closed once the system init message arrives, signaling
	// that the MCP server is connected and the socket is ready.
	sendCh := make(chan struct{}, 1)

	// Simulate a CI notification arriving while Claude is working.
	// A short delay after system init gives the CLI time to complete
	// the MCP handshake before we send the message.
	go func() {
		<-sendCh
		time.Sleep(500 * time.Millisecond)
		msg := channel.ChannelMessage{
			Content: "Build #1234 failed on main.\n" +
				"Stage: test\n" +
				"Failed: test_auth_flow (expected 200, got 401)\n" +
				"Commit: abc123f by @alice — \"refactor: extract token validation\"",
			SourceID:   "ci-build-1234",
			SourceName: "GitHub Actions",
		}
		fmt.Printf("[Channel] sending CI notification: %s\n", msg.SourceID)
		if err := session.Send(msg); err != nil {
			fmt.Fprintf(os.Stderr, "[Channel] send error: %v\n", err)
		}
	}()

	sentSignal := false
	for msg := range session.Messages {
		// Trigger the send as soon as we see the system init message,
		// which confirms the MCP server is connected.
		if !sentSignal && msg.Type == agentrunner.MessageTypeSystem {
			close(sendCh)
			sentSignal = true
		}

		switch msg.Type {
		case agentrunner.MessageTypeChannelReply:
			fmt.Printf("\n[Channel Reply → %s]\n%s\n",
				msg.ChannelReplyDestination(), msg.ChannelReplyContent())
		case agentrunner.MessageTypeAssistant:
			if text := msg.Text(); text != "" {
				fmt.Printf("[Assistant] %s\n", text)
			}
			// Show tool calls
			if name := msg.ToolName(); name != "" {
				fmt.Printf("[Tool Use] %s\n", name)
			}
		case agentrunner.MessageTypeToolUse:
			fmt.Printf("[Tool Use] %s\n", msg.ToolName())
		case agentrunner.MessageTypeToolResult:
			fmt.Printf("[Tool Result] (len=%d)\n", len(msg.Raw))
		case agentrunner.MessageTypeSystem:
			sm, ok := claudecode.ParseMessage(msg)
			if ok {
				fmt.Printf("[System] type=%s subtype=%s session=%s\n", sm.Type, sm.Subtype, sm.SessionID)
			}
		case agentrunner.MessageTypeResult:
			fmt.Println("\n---")
			fmt.Printf("[Result] %s\n", msg.Text())
		default:
			fmt.Printf("[%s] %s\n", msg.Type, string(msg.Raw[:min(120, len(msg.Raw))]))
		}
	}

	result, err := session.Result()
	if err != nil {
		return err
	}
	fmt.Printf("Cost: $%.4f | Duration: %s\n", result.CostUSD, result.Duration)
	return nil
}
