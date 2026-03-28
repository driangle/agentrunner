package channel

import (
	"context"
	"os"
	"testing"
)

func TestBinaryPath_EnvOverride(t *testing.T) {
	t.Setenv("AGENTRUNNER_CHANNEL_BIN", "/custom/agentrunner-channel")
	path, err := BinaryPath(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/custom/agentrunner-channel" {
		t.Errorf("got %q, want /custom/agentrunner-channel", path)
	}
}

func TestBinaryPath_EnvTakesPrecedence(t *testing.T) {
	// Even if the binary is on PATH, the env var should win.
	t.Setenv("AGENTRUNNER_CHANNEL_BIN", "/override/path")
	t.Setenv("PATH", os.Getenv("PATH"))
	path, err := BinaryPath(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/override/path" {
		t.Errorf("got %q, want /override/path", path)
	}
}
