// Package mcpclient speaks just enough of the newline-delimited JSON-RPC 2.0
// MCP protocol to ask a running mod for the tools it actually exposes — the
// author-side half of catching manifest/implementation drift (the host-side
// half lives in barrakuda-software, checked at spawn time).
package mcpclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// ToolSummary is the subset of a tools/list entry needed to diff against a
// manifest's declared tools[].
type ToolSummary struct {
	Name        string `json:"name"`
	InputSchema any    `json:"inputSchema"`
}

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ListTools spawns bin, performs the initialize/tools-list handshake over
// its stdio, and returns the tools it reports. The process is killed once
// the handshake completes (or fails) — this is a one-shot introspection
// call, not a long-lived connection.
func ListTools(bin string) ([]ToolSummary, error) {
	cmd := exec.Command(bin)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	type result struct {
		tools []ToolSummary
		err   error
	}
	done := make(chan result, 1)
	go func() {
		tools, err := listToolsOverIO(stdin, bufio.NewScanner(stdout))
		done <- result{tools, err}
	}()

	select {
	case r := <-done:
		return r.tools, r.err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("%s did not answer tools/list within 5s", bin)
	}
}

// listToolsOverIO performs the handshake against an already-connected
// peer — split out from ListTools so the protocol logic is unit-testable
// without spawning a real process.
func listToolsOverIO(w io.Writer, scanner *bufio.Scanner) ([]ToolSummary, error) {
	if err := writeRequest(w, jsonrpcRequest{
		JSONRPC: "2.0", ID: 1, Method: "initialize",
		Params: map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "barrakuda-mod-creator", "version": "0"},
		},
	}); err != nil {
		return nil, err
	}
	if _, err := readResponse(scanner); err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if err := writeRequest(w, jsonrpcRequest{JSONRPC: "2.0", Method: "notifications/initialized"}); err != nil {
		return nil, err
	}

	if err := writeRequest(w, jsonrpcRequest{JSONRPC: "2.0", ID: 2, Method: "tools/list"}); err != nil {
		return nil, err
	}
	res, err := readResponse(scanner)
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}

	var parsed struct {
		Tools []ToolSummary `json:"tools"`
	}
	if err := json.Unmarshal(res, &parsed); err != nil {
		return nil, fmt.Errorf("tools/list: malformed result: %w", err)
	}
	return parsed.Tools, nil
}

func writeRequest(w io.Writer, req jsonrpcRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal %s request: %w", req.Method, err)
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write %s request: %w", req.Method, err)
	}
	return nil
}

func readResponse(scanner *bufio.Scanner) (json.RawMessage, error) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("read response: peer closed the connection")
	}
	var res jsonrpcResponse
	if err := json.Unmarshal(scanner.Bytes(), &res); err != nil {
		return nil, fmt.Errorf("malformed response: %w", err)
	}
	if res.Error != nil {
		return nil, fmt.Errorf("peer returned an error: %s", res.Error.Message)
	}
	return res.Result, nil
}
