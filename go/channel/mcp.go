package channel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

const (
	protocolVersion = "2025-03-26"
	serverName      = "agentrunner-channel"
	serverVersion   = "0.1.0"

	instructions = "Messages from external sources arrive via this channel as notifications. " +
		"Each message has content and metadata (source_id, source_name, reply_to). " +
		"To respond to a channel message, use the 'reply' tool with the destination_id " +
		"set to the source_id of the message you want to reply to."

	methodInitialize  = "initialize"
	methodInitialized = "notifications/initialized"
	methodToolsList   = "tools/list"
	methodToolsCall   = "tools/call"
	methodPing        = "ping"

	notifyChannel = "notifications/claude/channel"

	errCodeMethodNotFound = -32601
	errCodeInvalidParams  = -32602
)

// Server is a lightweight MCP server that bridges Unix socket messages
// to Claude Code's channel system via stdio JSON-RPC.
type Server struct {
	scanner  *bufio.Scanner
	writer   *bufio.Writer
	outMu    sync.Mutex
	sockPath string
	log      *slog.Logger
}

// NewServer creates a new MCP channel server. If logger is nil, logging is disabled.
func NewServer(sockPath string, in io.Reader, out io.Writer, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &Server{
		scanner:  scanner,
		writer:   bufio.NewWriter(out),
		sockPath: sockPath,
		log:      logger,
	}
}

// Run starts the MCP server. It blocks until stdin is closed, the context
// is cancelled, or an unrecoverable error occurs.
func (s *Server) Run(ctx context.Context) error {
	s.log.Info("server starting", "sock_path", s.sockPath)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	msgCh := make(chan ChannelMessage, 64)

	var wg sync.WaitGroup

	// Start socket listener.
	wg.Add(1)
	var sockErr error
	go func() {
		defer wg.Done()
		sockErr = ListenSocket(ctx, s.sockPath, msgCh)
	}()

	// Start message forwarder.
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.forwardMessages(ctx, msgCh)
	}()

	// Read stdin on the current goroutine.
	err := s.readStdin(ctx)

	// Stdin closed or error — shut everything down.
	s.log.Info("server shutting down", "stdin_err", err, "sock_err", sockErr)
	cancel()
	wg.Wait()

	if err != nil {
		return err
	}
	return sockErr
}

// readStdin reads JSON-RPC requests from stdin and dispatches them.
func (s *Server) readStdin(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		req, err := ReadRequest(s.scanner)
		if err != nil {
			if err == io.EOF {
				s.log.Debug("stdin EOF")
				return nil
			}
			s.log.Warn("read error, skipping", "error", err)
			continue
		}

		s.log.Debug("received request", "method", req.Method, "id", string(req.ID))
		if err := s.dispatch(req); err != nil {
			return fmt.Errorf("dispatch %s: %w", req.Method, err)
		}
	}
}

// dispatch routes a JSON-RPC request to the appropriate handler.
func (s *Server) dispatch(req *Request) error {
	switch req.Method {
	case methodInitialize:
		s.log.Info("MCP initialize handshake")
		return s.handleInitialize(req)
	case methodInitialized:
		s.log.Info("MCP client initialized")
		return nil // notification, no response
	case methodToolsList:
		return s.handleToolsList(req)
	case methodToolsCall:
		return s.handleToolsCall(req)
	case methodPing:
		return s.writeResult(req.ID, map[string]any{})
	default:
		if req.IsNotification() {
			s.log.Debug("ignoring unknown notification", "method", req.Method)
			return nil
		}
		s.log.Warn("unknown method", "method", req.Method)
		return s.writeError(req.ID, errCodeMethodNotFound, "method not found: "+req.Method)
	}
}

// handleInitialize responds to the MCP initialize handshake.
func (s *Server) handleInitialize(req *Request) error {
	result := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"experimental": map[string]any{
				"claude/channel": map[string]any{},
			},
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    serverName,
			"version": serverVersion,
		},
		"instructions": instructions,
	}
	return s.writeResult(req.ID, result)
}

// replyTool is the tool definition for the reply tool.
var replyTool = map[string]any{
	"name":        "reply",
	"description": "Send a reply back through the channel to the message source",
	"inputSchema": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"destination_id": map[string]any{
				"type":        "string",
				"description": "The source_id of the message to reply to",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The reply message content",
			},
			"reply_to": map[string]any{
				"type":        "string",
				"description": "Optional reference ID for threading",
			},
		},
		"required": []string{"destination_id", "content"},
	},
}

// handleToolsList responds with the list of available tools.
func (s *Server) handleToolsList(req *Request) error {
	result := map[string]any{
		"tools": []any{replyTool},
	}
	return s.writeResult(req.ID, result)
}

// toolCallParams is the params structure for tools/call.
type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// handleToolsCall dispatches a tool call by name.
func (s *Server) handleToolsCall(req *Request) error {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.writeError(req.ID, errCodeInvalidParams, "invalid tool call params")
	}

	s.log.Info("tool call", "tool", params.Name, "args", string(params.Arguments))
	if params.Name != "reply" {
		s.log.Warn("unknown tool", "tool", params.Name)
		return s.writeResult(req.ID, map[string]any{
			"content": []map[string]any{{
				"type": "text",
				"text": fmt.Sprintf("unknown tool: %s", params.Name),
			}},
			"isError": true,
		})
	}

	// Acknowledge the reply. Actual routing is handled by library integration.
	return s.writeResult(req.ID, map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": "sent",
		}},
		"isError": false,
	})
}

// forwardMessages reads ChannelMessages from msgCh and sends them as
// MCP channel notifications to stdout.
func (s *Server) forwardMessages(ctx context.Context, msgCh <-chan ChannelMessage) {
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			s.log.Info("forwarding channel message",
				"source_id", msg.SourceID,
				"source_name", msg.SourceName,
				"content_len", len(msg.Content),
			)
			meta := map[string]any{
				"source_id":   msg.SourceID,
				"source_name": msg.SourceName,
			}
			if msg.ReplyTo != "" {
				meta["reply_to"] = msg.ReplyTo
			}
			notif := Notification{
				JSONRPC: "2.0",
				Method:  notifyChannel,
				Params: map[string]any{
					"content": msg.Content,
					"meta":    meta,
				},
			}
			// Best-effort write; if stdout is broken, readStdin will detect it.
			if err := WriteMessage(s.writer, &s.outMu, notif); err != nil {
				s.log.Error("failed to forward channel message", "error", err)
			} else {
				s.log.Debug("channel notification sent")
			}
		case <-ctx.Done():
			return
		}
	}
}

// writeResult sends a successful JSON-RPC response.
func (s *Server) writeResult(id json.RawMessage, result any) error {
	return WriteMessage(s.writer, &s.outMu, Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

// writeError sends a JSON-RPC error response.
func (s *Server) writeError(id json.RawMessage, code int, message string) error {
	return WriteMessage(s.writer, &s.outMu, Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	})
}
