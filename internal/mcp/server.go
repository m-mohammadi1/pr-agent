// Package mcp exposes pr-agent's GitHub PR review capabilities as an
// MCP (Model Context Protocol) stdio server, so MCP-aware clients like
// Cursor and Claude Code can call them as native tools.
//
// It implements the MCP stdio transport (newline-delimited JSON-RPC 2.0)
// using only the standard library.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// Version is the reported server version (visible in Cursor MCP server info).
const Version = "0.2.0"

// defaultProtocolVersion is used if the client does not send one.
const defaultProtocolVersion = "2025-06-18"

// Serve runs the MCP server over stdio until stdin closes.
func Serve(ctx context.Context) error {
	s := &server{
		in:    bufio.NewReaderSize(os.Stdin, 1<<20),
		out:   os.Stdout,
		tools: buildTools(),
	}
	return s.run(ctx)
}

type server struct {
	in    *bufio.Reader
	out   io.Writer
	mu    sync.Mutex
	tools []toolDef
}

// --- JSON-RPC types ---

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	codeParseError     = -32700
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

func (s *server) run(ctx context.Context) error {
	for {
		line, err := s.in.ReadBytes('\n')
		if len(line) > 0 {
			s.handleLine(ctx, line)
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func (s *server) handleLine(ctx context.Context, line []byte) {
	var req rpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		s.writeError(nil, codeParseError, "parse error")
		return
	}

	// Notifications have no id; never send a response.
	isNotification := len(req.ID) == 0

	switch req.Method {
	case "initialize":
		s.reply(req.ID, s.handleInitialize(req.Params))
	case "notifications/initialized", "notifications/cancelled":
		// no-op notifications
	case "ping":
		if !isNotification {
			s.reply(req.ID, map[string]any{})
		}
	case "tools/list":
		s.reply(req.ID, s.handleToolsList())
	case "tools/call":
		s.handleToolsCall(ctx, req.ID, req.Params)
	default:
		if !isNotification {
			s.writeError(req.ID, codeMethodNotFound, "method not found: "+req.Method)
		}
	}
}

func (s *server) handleInitialize(params json.RawMessage) map[string]any {
	protocol := defaultProtocolVersion
	if len(params) > 0 {
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(params, &p); err == nil && p.ProtocolVersion != "" {
			protocol = p.ProtocolVersion
		}
	}

	return map[string]any{
		"protocolVersion": protocol,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "pr-agent",
			"title":   "GitHub PR review bridge for AI agents",
			"version": Version,
		},
	}
}

func (s *server) handleToolsList() map[string]any {
	list := make([]map[string]any, 0, len(s.tools))
	for _, t := range s.tools {
		list = append(list, map[string]any{
			"name":        t.name,
			"description": t.description,
			"inputSchema": t.inputSchema,
		})
	}
	return map[string]any{"tools": list}
}

func (s *server) handleToolsCall(ctx context.Context, id json.RawMessage, params json.RawMessage) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		s.writeError(id, codeInvalidParams, "invalid params: "+err.Error())
		return
	}

	for _, t := range s.tools {
		if t.name == call.Name {
			result := t.handler(ctx, call.Arguments)
			s.reply(id, result)
			return
		}
	}
	s.reply(id, toolError(fmt.Errorf("unknown tool: %s", call.Name)))
}

// --- output helpers ---

func (s *server) reply(id json.RawMessage, result any) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *server) writeError(id json.RawMessage, code int, msg string) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func (s *server) write(resp rpcResponse) {
	b, err := json.Marshal(resp)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b = append(b, '\n')
	_, _ = s.out.Write(b)
}
