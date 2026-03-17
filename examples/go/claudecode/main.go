// This example demonstrates how to use the agentrunner Go library to invoke
// Claude Code CLI programmatically, covering basic usage, streaming, and
// session management.
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
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	agentrunner "github.com/driangle/agentrunner-go"
	"github.com/driangle/agentrunner-go/claudecode"
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

	if err := run(runnerOpts, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(runnerOpts []claudecode.RunnerOption, verbose bool) error {
	ctx := context.Background()

	runner := claudecode.NewRunner(runnerOpts...)

	// --- Example 1: Simple Run ---
	fmt.Println("=== Example 1: Simple Run ===")
	if err := exampleSimpleRun(ctx, runner); err != nil {
		return fmt.Errorf("simple run: %w", err)
	}

	// --- Example 2: Streaming ---
	fmt.Println("\n=== Example 2: Streaming ===")
	if err := exampleStreaming(ctx, runner, verbose); err != nil {
		return fmt.Errorf("streaming: %w", err)
	}

	// --- Example 3: Session Resume ---
	fmt.Println("\n=== Example 3: Session Resume ===")
	if err := exampleSessionResume(ctx, runner); err != nil {
		return fmt.Errorf("session resume: %w", err)
	}

	return nil
}

// exampleSimpleRun sends a single prompt and prints the result.
func exampleSimpleRun(ctx context.Context, runner *claudecode.Runner) error {
	prompt := "What is 2+2? Reply with just the number."
	fmt.Printf("Prompt:   %s\n", prompt)

	result, err := runner.Run(ctx, prompt,
		agentrunner.WithMaxTurns(1),
		agentrunner.WithTimeout(30*time.Second),
	)
	if err != nil {
		return err
	}

	fmt.Printf("Response: %s\n", result.Text)
	fmt.Printf("Cost:     $%.4f\n", result.CostUSD)
	fmt.Printf("Tokens:   %d in / %d out\n", result.Usage.InputTokens, result.Usage.OutputTokens)
	fmt.Printf("Duration: %dms\n", result.DurationMs)
	fmt.Printf("Session:  %s\n", result.SessionID)
	fmt.Printf("Error:    %v\n", result.IsError)
	fmt.Printf("Exit:     %d\n", result.ExitCode)
	return nil
}

// exampleStreaming uses RunStream to print messages as they arrive.
func exampleStreaming(ctx context.Context, runner *claudecode.Runner, verbose bool) error {
	prompt := "List 3 fun facts about Go (the programming language). Be brief."
	fmt.Printf("Prompt: %s\n", prompt)
	fmt.Println("---")

	msgCh, errCh := runner.RunStream(ctx, prompt,
		agentrunner.WithMaxTurns(1),
		agentrunner.WithTimeout(30*time.Second),
		claudecode.WithIncludePartialMessages(true),
	)

	var model string

	for msg := range msgCh {
		switch msg.Type {
		case agentrunner.MessageTypeSystem:
			if verbose {
				fmt.Printf("[system] %s\n", msg.Raw)
			}
			parsed, parseErr := claudecode.Parse(string(msg.Raw))
			if parseErr == nil && parsed.Model != "" {
				model = parsed.Model
			}
		case agentrunner.MessageTypeAssistant:
			// With --include-partial-messages, the CLI emits two kinds of
			// messages mapped to "assistant":
			//   1. stream_event with content_block_delta — real-time text deltas
			//   2. assistant — full accumulated message (arrives at the end)
			// Print deltas for real-time streaming; skip the final assistant
			// message to avoid duplicating the output.
			parsed, parseErr := claudecode.Parse(string(msg.Raw))
			if parseErr != nil {
				continue
			}
			if parsed.Type == "stream_event" && parsed.Event != nil &&
				parsed.Event.Delta != nil && parsed.Event.Delta.Type == "text_delta" {
				fmt.Print(parsed.Event.Delta.Text)
			}
		case agentrunner.MessageTypeResult:
			parsed, parseErr := claudecode.Parse(string(msg.Raw))
			if parseErr != nil {
				continue
			}
			if parsed.Model != "" {
				model = parsed.Model
			}
			fmt.Println("\n---")
			fmt.Printf("Cost:     $%.4f\n", parsed.TotalCostUSD)
			fmt.Printf("Duration: %.0fms\n", parsed.DurationMs)
			fmt.Printf("Turns:    %d\n", parsed.NumTurns)
			fmt.Printf("Model:    %s\n", model)
			fmt.Printf("Session:  %s\n", parsed.SessionID)
			fmt.Printf("Error:    %v\n", parsed.IsError)
		}
	}

	if err := <-errCh; err != nil {
		return err
	}
	return nil
}

// exampleSessionResume demonstrates multi-turn conversations using session IDs.
func exampleSessionResume(ctx context.Context, runner *claudecode.Runner) error {
	// First turn: ask Claude to remember something.
	prompt1 := "Remember this number: 42. Just confirm you've noted it."
	fmt.Printf("Prompt 1: %s\n", prompt1)

	result, err := runner.Run(ctx, prompt1,
		agentrunner.WithMaxTurns(1),
		agentrunner.WithTimeout(30*time.Second),
	)
	if err != nil {
		return err
	}
	fmt.Printf("Response: %s\n", result.Text)
	fmt.Printf("Session:  %s\n", result.SessionID)

	if result.SessionID == "" {
		return errors.New("no session ID returned — cannot demonstrate resume")
	}

	// Second turn: resume the session and reference the earlier context.
	prompt2 := "What number did I ask you to remember?"
	fmt.Printf("\nPrompt 2: %s (resume: %s)\n", prompt2, result.SessionID)

	result, err = runner.Run(ctx, prompt2,
		agentrunner.WithMaxTurns(1),
		agentrunner.WithTimeout(30*time.Second),
		claudecode.WithResume(result.SessionID),
	)
	if err != nil {
		return err
	}
	fmt.Printf("Response: %s\n", result.Text)
	return nil
}
