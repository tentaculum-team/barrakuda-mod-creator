package mcpclient

import (
	"bufio"
	"encoding/json"
	"io"
	"testing"
)

// fakeServer speaks just enough newline-delimited JSON-RPC to answer
// initialize/notifications-initialized/tools-list, mirroring the protocol
// the hand-rolled scaffolds (mcp_typescript.go's tsMcpServer) implement.
// Runs on its own goroutine, so it can't use *testing.T's Fatal/FailNow
// (only safe on the test's own goroutine) — a malformed request or write
// failure just ends the fake server early, which fails the real assertions
// in the calling test instead.
func fakeServer(r io.Reader, w io.Writer, tools []map[string]any) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(line, &req) != nil {
			return
		}

		switch req.Method {
		case "notifications/initialized":
			continue
		case "initialize":
			if !writeLine(w, map[string]any{
				"jsonrpc": "2.0", "id": json.RawMessage(req.ID),
				"result": map[string]any{"protocolVersion": "2025-06-18"},
			}) {
				return
			}
		case "tools/list":
			writeLine(w, map[string]any{
				"jsonrpc": "2.0", "id": json.RawMessage(req.ID),
				"result": map[string]any{"tools": tools},
			})
			return
		}
	}
}

func writeLine(w io.Writer, v any) bool {
	b, err := json.Marshal(v)
	if err != nil {
		return false
	}
	_, err = w.Write(append(b, '\n'))
	return err == nil
}

func TestListToolsOverIO(t *testing.T) {
	reqR, reqW := io.Pipe()
	respR, respW := io.Pipe()

	go fakeServer(reqR, respW, []map[string]any{
		{"name": "greet", "inputSchema": map[string]any{"type": "object"}},
	})

	tools, err := listToolsOverIO(reqW, bufio.NewScanner(respR))
	if err != nil {
		t.Fatalf("listToolsOverIO: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "greet" {
		t.Fatalf("expected one tool named greet, got %+v", tools)
	}
}

func TestListToolsOverIO_EmptyToolList(t *testing.T) {
	reqR, reqW := io.Pipe()
	respR, respW := io.Pipe()

	go fakeServer(reqR, respW, []map[string]any{})

	tools, err := listToolsOverIO(reqW, bufio.NewScanner(respR))
	if err != nil {
		t.Fatalf("listToolsOverIO: %v", err)
	}
	if len(tools) != 0 {
		t.Fatalf("expected zero tools, got %+v", tools)
	}
}
