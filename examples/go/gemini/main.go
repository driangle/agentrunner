// This example demonstrates how to use the agentrunner Go library to invoke
// the Gemini CLI programmatically, covering basic usage, streaming, and session
// resume.
//
// Prerequisites:
//   - Gemini CLI installed (>= 0.1.0): https://github.com/google-gemini/gemini-cli
//   - Authenticated with Gemini CLI
//
// Run:
//
//	go run .
//	go run . --gemini /path/to/gemini
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/driangle/agentrunner/go"
	"github.com/driangle/agentrunner/go/gemini"
)

func main() {
	geminiBinary := flag.String("gemini", "gemini", "path to the Gemini CLI binary")
	verbose := flag.Bool("verbose", false, "enable debug logging")
	flag.Parse()

	var runnerOpts []gemini.RunnerOption
	runnerOpts = append(runnerOpts, gemini.WithBinary(*geminiBinary))
	if *verbose {
		runnerOpts = append(runnerOpts, gemini.WithLogger(
			slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})),
		))
	}

	if err := run(runnerOpts, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(runnerOpts []gemini.RunnerOption, verbose bool) error {
	ctx := context.Background()

	runner := gemini.NewRunner(runnerOpts...)

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
func exampleSimpleRun(ctx context.Context, runner *gemini.Runner) error {
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
	fmt.Printf("Tokens:   %d in / %d out\n", result.Usage.InputTokens, result.Usage.OutputTokens)
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("Session:  %s\n", result.SessionID)
	fmt.Printf("Error:    %v\n", result.IsError)
	return nil
}

// exampleStreaming uses Start to stream messages as they arrive.
func exampleStreaming(ctx context.Context, runner *gemini.Runner, verbose bool) error {
	prompt := "List 3 fun facts about Go (the programming language). Be brief."
	fmt.Printf("Prompt: %s\n", prompt)
	fmt.Println("---")

	session, err := runner.Start(ctx, prompt,
		agentrunner.WithMaxTurns(1),
		agentrunner.WithTimeout(30*time.Second),
	)
	if err != nil {
		return err
	}

	for msg := range session.Messages {
		switch msg.Type {
		case agentrunner.MessageTypeSystem:
			if verbose {
				fmt.Printf("[system] %s\n", msg.Raw)
			}
		case agentrunner.MessageTypeAssistant:
			sm, ok := gemini.ParseMessage(msg)
			if !ok {
				continue
			}
			// Delta messages carry incremental text; non-delta carry the full response.
			if sm.Delta {
				fmt.Print(sm.Content)
			} else {
				fmt.Print(sm.Content)
			}
		case agentrunner.MessageTypeToolUse:
			sm, ok := gemini.ParseMessage(msg)
			if !ok {
				continue
			}
			fmt.Printf("\n[tool_use] %s (id: %s)\n", sm.ToolNameField, sm.ToolID)
		case agentrunner.MessageTypeToolResult:
			sm, ok := gemini.ParseMessage(msg)
			if !ok {
				continue
			}
			fmt.Printf("[tool_result] %s (status: %s)\n", sm.Output, sm.Status)
		case agentrunner.MessageTypeResult:
			fmt.Println("\n---")
		case agentrunner.MessageTypeError:
			sm, ok := gemini.ParseMessage(msg)
			if !ok {
				continue
			}
			fmt.Printf("[error] %s\n", sm.MessageField)
		}
	}

	result, err := session.Result()
	if err != nil {
		return err
	}

	fmt.Printf("Tokens:   %d in / %d out\n", result.Usage.InputTokens, result.Usage.OutputTokens)
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("Session:  %s\n", result.SessionID)
	return nil
}

// exampleSessionResume demonstrates multi-turn conversations using session IDs.
func exampleSessionResume(ctx context.Context, runner *gemini.Runner) error {
	// First turn: ask Gemini to remember something.
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
		gemini.WithResume(result.SessionID),
	)
	if err != nil {
		return err
	}
	fmt.Printf("Response: %s\n", result.Text)
	return nil
}
