// Package channel implements a lightweight MCP server that bridges Unix socket
// IPC from agentrunner libraries to Claude Code's channel system via stdio JSON-RPC.
package channel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Request represents an incoming JSON-RPC 2.0 request or notification.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsNotification reports whether the request is a notification (no ID).
func (r *Request) IsNotification() bool {
	return len(r.ID) == 0 || string(r.ID) == "null"
}

// Response represents an outgoing JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Notification is an outgoing JSON-RPC 2.0 notification (no id field).
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// ReadRequest reads one newline-delimited JSON-RPC request from the scanner.
// Returns io.EOF when the scanner is exhausted.
func ReadRequest(scanner *bufio.Scanner) (*Request, error) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading request: %w", err)
		}
		return nil, io.EOF
	}
	var req Request
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		return nil, fmt.Errorf("unmarshaling request: %w", err)
	}
	return &req, nil
}

// WriteMessage marshals v as JSON, appends a newline, and writes it to w
// under the provided mutex. The writer is flushed after each message.
func WriteMessage(w *bufio.Writer, mu *sync.Mutex, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}
	data = append(data, '\n')

	mu.Lock()
	defer mu.Unlock()

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flushing message: %w", err)
	}
	return nil
}
