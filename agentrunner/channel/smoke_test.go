package channel_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/driangle/agent-runner/agentrunner/channel"
)

// TestSmoke exercises the full channel server binary end-to-end:
// 1. Builds the binary
// 2. Starts it with a Unix socket
// 3. Performs an MCP initialize handshake over stdio
// 4. Sends a ChannelMessage via Unix socket
// 5. Verifies the notification arrives on stdout
// 6. Sends tools/list and tools/call requests
//
// Run with: go test ./channel/ -run TestSmoke -v
func TestSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test in short mode")
	}

	// Build the binary.
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "agentrunner-channel")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/agentrunner-channel")
	build.Dir = filepath.Join(findModuleRoot(t), "agentrunner")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// Create a short socket path (macOS 104-char limit).
	sockDir, err := os.MkdirTemp("/tmp", "ar-smoke-")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(sockDir) })
	sockPath := filepath.Join(sockDir, "s.sock")

	// Start the binary.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath)
	cmd.Env = append(os.Environ(), "AGENTRUNNER_CHANNEL_SOCK="+sockPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	readLine := func(label string) string {
		t.Helper()
		done := make(chan string, 1)
		go func() {
			if scanner.Scan() {
				done <- scanner.Text()
			} else {
				done <- ""
			}
		}()
		select {
		case line := <-done:
			if line == "" {
				t.Fatalf("%s: empty or no output", label)
			}
			return line
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: timed out waiting for output", label)
			return ""
		}
	}

	// --- Step 1: Initialize handshake ---
	writeJSON(t, stdin, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "smoke-test", "version": "1.0.0"},
		},
	})

	initResp := readLine("initialize")
	assertJSONField(t, initResp, "result.protocolVersion", "2025-03-26")
	t.Log("initialize: OK")

	writeJSON(t, stdin, map[string]any{
		"jsonrpc": "2.0", "method": "notifications/initialized",
	})

	// --- Step 2: Send a ChannelMessage via Unix socket ---
	var conn net.Conn
	for i := 0; i < 50; i++ {
		conn, err = net.Dial("unix", sockPath)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if conn == nil {
		t.Fatal("could not connect to socket")
	}

	msg := channel.ChannelMessage{
		Content:    "smoke test message",
		SourceID:   "smoke-1",
		SourceName: "smoke-test",
	}
	data, _ := json.Marshal(msg)
	data = append(data, '\n')
	conn.Write(data)
	conn.Close()

	notifLine := readLine("channel notification")
	if !strings.Contains(notifLine, "notifications/claude/channel") {
		t.Errorf("expected channel notification, got: %s", notifLine)
	}
	if !strings.Contains(notifLine, "smoke test message") {
		t.Errorf("expected message content, got: %s", notifLine)
	}
	t.Log("channel notification: OK")

	// --- Step 3: tools/list ---
	writeJSON(t, stdin, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/list",
	})

	toolsLine := readLine("tools/list")
	if !strings.Contains(toolsLine, "reply") {
		t.Errorf("expected reply tool, got: %s", toolsLine)
	}
	t.Log("tools/list: OK")

	// --- Step 4: tools/call (reply) ---
	writeJSON(t, stdin, map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]any{
			"name":      "reply",
			"arguments": map[string]any{"destination_id": "smoke-1", "content": "ack"},
		},
	})

	replyLine := readLine("tools/call")
	if !strings.Contains(replyLine, "sent") {
		t.Errorf("expected 'sent' in reply, got: %s", replyLine)
	}
	t.Log("tools/call reply: OK")

	// Close stdin to trigger clean shutdown.
	stdin.Close()
	if err := cmd.Wait(); err != nil {
		// Context cancellation is expected.
		if ctx.Err() == nil {
			t.Logf("process exit: %v (expected)", err)
		}
	}
	t.Log("clean shutdown: OK")
}

func writeJSON(t *testing.T, w interface{ Write([]byte) (int, error) }, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func assertJSONField(t *testing.T, line, path, expected string) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("unmarshal %q: %v", line, err)
	}
	parts := strings.Split(path, ".")
	var current any = m
	for _, p := range parts {
		obj, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("path %q: expected object at %q, got %T", path, p, current)
		}
		current = obj[p]
	}
	if current != expected {
		t.Errorf("path %q = %v, want %q", path, current, expected)
	}
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root")
		}
		dir = parent
	}
}
