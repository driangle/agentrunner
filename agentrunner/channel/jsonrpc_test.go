package channel

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestReadRequest(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		method string
		hasID  bool
	}{
		{
			name:   "request with numeric id",
			input:  `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n",
			method: "initialize",
			hasID:  true,
		},
		{
			name:   "request with string id",
			input:  `{"jsonrpc":"2.0","id":"abc","method":"tools/list"}` + "\n",
			method: "tools/list",
			hasID:  true,
		},
		{
			name:   "notification without id",
			input:  `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n",
			method: "notifications/initialized",
			hasID:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(tt.input))
			req, err := ReadRequest(scanner)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.Method != tt.method {
				t.Errorf("method = %q, want %q", req.Method, tt.method)
			}
			if tt.hasID && req.IsNotification() {
				t.Error("expected request with ID, got notification")
			}
			if !tt.hasID && !req.IsNotification() {
				t.Error("expected notification, got request with ID")
			}
		})
	}
}

func TestReadRequestEOF(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(""))
	_, err := ReadRequest(scanner)
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestReadRequestMalformed(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("not json\n"))
	_, err := ReadRequest(scanner)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestWriteMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  any
	}{
		{
			name: "response",
			msg: Response{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Result:  map[string]string{"key": "value"},
			},
		},
		{
			name: "notification",
			msg: Notification{
				JSONRPC: "2.0",
				Method:  "notifications/claude/channel",
				Params:  map[string]string{"content": "hello"},
			},
		},
		{
			name: "error response",
			msg: Response{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`2`),
				Error:   &RPCError{Code: -32601, Message: "method not found"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			var mu sync.Mutex

			if err := WriteMessage(w, &mu, tt.msg); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			if !strings.HasSuffix(output, "\n") {
				t.Error("output should end with newline")
			}

			// Verify it's valid JSON.
			trimmed := strings.TrimSpace(output)
			if !json.Valid([]byte(trimmed)) {
				t.Errorf("output is not valid JSON: %s", trimmed)
			}
		})
	}
}

func TestWriteMessageConcurrent(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := Notification{
				JSONRPC: "2.0",
				Method:  "test",
				Params:  map[string]int{"id": id},
			}
			if err := WriteMessage(w, &mu, msg); err != nil {
				t.Errorf("write error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Verify each line is individually parseable JSON.
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 20 {
		t.Fatalf("expected 20 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d is not valid JSON: %s", i, line)
		}
	}
}
