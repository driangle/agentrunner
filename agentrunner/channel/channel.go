package channel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

// ChannelMessage is the wire format for messages arriving over the Unix socket.
type ChannelMessage struct {
	Content    string `json:"content"`
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	ReplyTo    string `json:"reply_to,omitempty"`
}

// ListenSocket listens on a Unix socket at sockPath and delivers decoded
// ChannelMessage values to msgCh. It blocks until ctx is cancelled, then
// cleans up the socket file. The caller should run this in a goroutine.
func ListenSocket(ctx context.Context, sockPath string, msgCh chan<- ChannelMessage) error {
	// Remove stale socket file.
	os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", sockPath, err)
	}

	// Close the listener when context is cancelled to unblock Accept.
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	var wg sync.WaitGroup
	defer func() {
		ln.Close()
		wg.Wait()
		os.Remove(sockPath)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Expected when listener is closed during shutdown.
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("accepting connection: %w", err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer conn.Close()
			handleConn(ctx, conn, msgCh)
		}()
	}
}

// handleConn reads newline-delimited JSON ChannelMessages from a connection.
func handleConn(ctx context.Context, conn net.Conn, msgCh chan<- ChannelMessage) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		var msg ChannelMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue // skip malformed messages
		}
		select {
		case msgCh <- msg:
		case <-ctx.Done():
			return
		}
	}
}
