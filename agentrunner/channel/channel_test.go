package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// tempSockPath returns a short Unix socket path under /tmp to avoid macOS's
// 104-character limit on socket paths. The file is cleaned up by t.Cleanup.
func tempSockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "ar-test-")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "s.sock")
}

func waitForSocket(t *testing.T, sockPath string) net.Conn {
	t.Helper()
	for i := 0; i < 50; i++ {
		conn, err := net.Dial("unix", sockPath)
		if err == nil {
			return conn
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("could not connect to socket")
	return nil
}

func sendMessage(t *testing.T, conn net.Conn, msg ChannelMessage) {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestListenSocketReceivesMessages(t *testing.T) {
	sockPath := tempSockPath(t)
	msgCh := make(chan ChannelMessage, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ListenSocket(ctx, sockPath, msgCh)
	}()

	conn := waitForSocket(t, sockPath)
	defer conn.Close()

	msg := ChannelMessage{
		Content:    "build failed",
		SourceID:   "ci-123",
		SourceName: "github-actions",
	}
	sendMessage(t, conn, msg)

	select {
	case got := <-msgCh:
		if got.Content != msg.Content {
			t.Errorf("content = %q, want %q", got.Content, msg.Content)
		}
		if got.SourceID != msg.SourceID {
			t.Errorf("source_id = %q, want %q", got.SourceID, msg.SourceID)
		}
		if got.SourceName != msg.SourceName {
			t.Errorf("source_name = %q, want %q", got.SourceName, msg.SourceName)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestListenSocketMultipleConnections(t *testing.T) {
	sockPath := tempSockPath(t)
	msgCh := make(chan ChannelMessage, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ListenSocket(ctx, sockPath, msgCh)
	}()

	for i := 0; i < 3; i++ {
		conn := waitForSocket(t, sockPath)
		msg := ChannelMessage{Content: fmt.Sprintf("msg-%d", i), SourceID: fmt.Sprintf("src-%d", i)}
		sendMessage(t, conn, msg)
		conn.Close()
	}

	received := 0
	timeout := time.After(2 * time.Second)
	for received < 3 {
		select {
		case <-msgCh:
			received++
		case <-timeout:
			t.Fatalf("received only %d/3 messages", received)
		}
	}
}

func TestListenSocketContextCancellation(t *testing.T) {
	sockPath := tempSockPath(t)
	msgCh := make(chan ChannelMessage, 10)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenSocket(ctx, sockPath, msgCh)
	}()

	// Wait for socket to be ready.
	waitForSocket(t, sockPath).Close()

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ListenSocket did not return after context cancellation")
	}
}

func TestListenSocketMalformedJSON(t *testing.T) {
	sockPath := tempSockPath(t)
	msgCh := make(chan ChannelMessage, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ListenSocket(ctx, sockPath, msgCh)
	}()

	conn := waitForSocket(t, sockPath)
	defer conn.Close()

	// Send malformed JSON followed by valid JSON.
	conn.Write([]byte("not json\n"))
	valid := ChannelMessage{Content: "valid", SourceID: "ok"}
	sendMessage(t, conn, valid)

	select {
	case got := <-msgCh:
		if got.Content != "valid" {
			t.Errorf("content = %q, want %q", got.Content, "valid")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for valid message after malformed input")
	}
}
