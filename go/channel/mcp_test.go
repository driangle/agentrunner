package channel

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"
)

// helper to write a JSON-RPC line and read the response.
func roundTrip(t *testing.T, s *Server, req any) json.RawMessage {
	t.Helper()
	// We use dispatch directly for synchronous tests.
	var r Request
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("unmarshal into Request: %v", err)
	}
	if err := s.dispatch(&r); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	return nil // response is in the output buffer
}

func newTestServer(t *testing.T, out *bytes.Buffer) *Server {
	t.Helper()
	return &Server{
		writer:   bufio.NewWriter(out),
		sockPath: tempSockPath(t),
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func readResponse(t *testing.T, data string) Response {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) == 0 {
		t.Fatal("no output lines")
	}
	var resp Response
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, lines[len(lines)-1])
	}
	return resp
}

func TestInitializeHandshake(t *testing.T) {
	var out bytes.Buffer
	srv := newTestServer(t, &out)

	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  methodInitialize,
		Params:  json.RawMessage(`{}`),
	}
	if err := srv.dispatch(&req); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	resp := readResponse(t, out.String())
	if string(resp.ID) != "1" {
		t.Errorf("response id = %s, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var parsed map[string]any
	json.Unmarshal(result, &parsed)

	if parsed["protocolVersion"] != protocolVersion {
		t.Errorf("protocolVersion = %v, want %s", parsed["protocolVersion"], protocolVersion)
	}

	caps, ok := parsed["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("capabilities not found")
	}
	exp, ok := caps["experimental"].(map[string]any)
	if !ok {
		t.Fatal("experimental capability not found")
	}
	if _, ok := exp["claude/channel"]; !ok {
		t.Error("claude/channel capability not advertised")
	}
	if _, ok := caps["tools"]; !ok {
		t.Error("tools capability not advertised")
	}

	info, ok := parsed["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("serverInfo not found")
	}
	if info["name"] != serverName {
		t.Errorf("serverInfo.name = %v, want %s", info["name"], serverName)
	}

	if parsed["instructions"] == nil || parsed["instructions"] == "" {
		t.Error("instructions should be present")
	}
}

func TestInitializedNotification(t *testing.T) {
	var out bytes.Buffer
	srv := newTestServer(t, &out)

	req := Request{
		JSONRPC: "2.0",
		Method:  methodInitialized,
	}
	if err := srv.dispatch(&req); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if out.Len() != 0 {
		t.Errorf("notifications/initialized should produce no output, got: %s", out.String())
	}
}

func TestToolsList(t *testing.T) {
	var out bytes.Buffer
	srv := newTestServer(t, &out)

	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  methodToolsList,
	}
	if err := srv.dispatch(&req); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	resp := readResponse(t, out.String())
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, _ := json.Marshal(resp.Result)
	var parsed map[string]any
	json.Unmarshal(result, &parsed)

	tools, ok := parsed["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}

	tool := tools[0].(map[string]any)
	if tool["name"] != "reply" {
		t.Errorf("tool name = %v, want reply", tool["name"])
	}
	if tool["inputSchema"] == nil {
		t.Error("tool should have inputSchema")
	}
}

func TestToolsCallReply(t *testing.T) {
	var out bytes.Buffer
	srv := newTestServer(t, &out)

	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  methodToolsCall,
		Params:  json.RawMessage(`{"name":"reply","arguments":{"destination_id":"ci-123","content":"fixed"}}`),
	}
	if err := srv.dispatch(&req); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	resp := readResponse(t, out.String())
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, _ := json.Marshal(resp.Result)
	var parsed map[string]any
	json.Unmarshal(result, &parsed)

	if parsed["isError"] != false {
		t.Errorf("isError = %v, want false", parsed["isError"])
	}
}

func TestToolsCallUnknownTool(t *testing.T) {
	var out bytes.Buffer
	srv := newTestServer(t, &out)

	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  methodToolsCall,
		Params:  json.RawMessage(`{"name":"unknown","arguments":{}}`),
	}
	if err := srv.dispatch(&req); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	resp := readResponse(t, out.String())
	result, _ := json.Marshal(resp.Result)
	var parsed map[string]any
	json.Unmarshal(result, &parsed)

	if parsed["isError"] != true {
		t.Errorf("isError = %v, want true for unknown tool", parsed["isError"])
	}
}

func TestUnknownMethodWithID(t *testing.T) {
	var out bytes.Buffer
	srv := newTestServer(t, &out)

	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`5`),
		Method:  "unknown/method",
	}
	if err := srv.dispatch(&req); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	resp := readResponse(t, out.String())
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != errCodeMethodNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, errCodeMethodNotFound)
	}
}

func TestUnknownNotification(t *testing.T) {
	var out bytes.Buffer
	srv := newTestServer(t, &out)

	req := Request{
		JSONRPC: "2.0",
		Method:  "unknown/notification",
	}
	if err := srv.dispatch(&req); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if out.Len() != 0 {
		t.Errorf("unknown notification should produce no output, got: %s", out.String())
	}
}

func TestPing(t *testing.T) {
	var out bytes.Buffer
	srv := newTestServer(t, &out)

	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`6`),
		Method:  methodPing,
	}
	if err := srv.dispatch(&req); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	resp := readResponse(t, out.String())
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestChannelNotificationViaSocket(t *testing.T) {
	sockPath := tempSockPath(t)

	var out bytes.Buffer
	srv := &Server{
		writer:   bufio.NewWriter(&out),
		sockPath: sockPath,
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgCh := make(chan ChannelMessage, 10)

	// Start socket listener.
	go func() {
		ListenSocket(ctx, sockPath, msgCh)
	}()

	// Start message forwarder.
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.forwardMessages(ctx, msgCh)
	}()

	// Wait for socket.
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	msg := ChannelMessage{
		Content:    "deploy complete",
		SourceID:   "deploy-456",
		SourceName: "ci-bot",
		ReplyTo:    "req-789",
	}
	data, _ := json.Marshal(msg)
	data = append(data, '\n')
	conn.Write(data)
	conn.Close()

	// Give the forwarder time to process.
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	output := out.String()
	if output == "" {
		t.Fatal("expected notification output")
	}

	var notif Notification
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &notif); err != nil {
		t.Fatalf("unmarshal notification: %v\nraw: %s", err, output)
	}

	if notif.Method != notifyChannel {
		t.Errorf("method = %q, want %q", notif.Method, notifyChannel)
	}

	params, _ := json.Marshal(notif.Params)
	var parsed map[string]any
	json.Unmarshal(params, &parsed)

	if parsed["content"] != "deploy complete" {
		t.Errorf("content = %v, want %q", parsed["content"], "deploy complete")
	}

	meta, ok := parsed["meta"].(map[string]any)
	if !ok {
		t.Fatal("meta not found")
	}
	if meta["source_id"] != "deploy-456" {
		t.Errorf("source_id = %v, want %q", meta["source_id"], "deploy-456")
	}
	if meta["source_name"] != "ci-bot" {
		t.Errorf("source_name = %v, want %q", meta["source_name"], "ci-bot")
	}
	if meta["reply_to"] != "req-789" {
		t.Errorf("reply_to = %v, want %q", meta["reply_to"], "req-789")
	}
}

func TestFullRunInitializeAndToolsList(t *testing.T) {
	sockPath := tempSockPath(t)

	// Build stdin with initialize request + tools/list + close.
	var stdin bytes.Buffer
	initReq, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})
	stdin.Write(initReq)
	stdin.WriteByte('\n')

	initializedNotif, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	stdin.Write(initializedNotif)
	stdin.WriteByte('\n')

	toolsReq, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	stdin.Write(toolsReq)
	stdin.WriteByte('\n')

	var stdout bytes.Buffer
	srv := NewServer(sockPath, &stdin, &stdout, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 response lines, got %d: %s", len(lines), stdout.String())
	}

	// First response: initialize.
	var initResp Response
	json.Unmarshal([]byte(lines[0]), &initResp)
	if string(initResp.ID) != "1" {
		t.Errorf("first response id = %s, want 1", initResp.ID)
	}

	// Second response: tools/list.
	var toolsResp Response
	json.Unmarshal([]byte(lines[1]), &toolsResp)
	if string(toolsResp.ID) != "2" {
		t.Errorf("second response id = %s, want 2", toolsResp.ID)
	}
}
