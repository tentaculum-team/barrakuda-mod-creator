package execute_init

import (
	"fmt"
	"os"

	"barrakudaModKit/internal/manifest"
)

// CreateGoMcp scaffolds a Go MCP server: a stdio JSON-RPC MCP implementation
// hand-rolled with no SDK (matching the TS/Python scaffolds), wired through
// the same domain/repository/service layering used by auth/payments,
// exposing two example tools — greet and reset_greeting_count — plus a
// comment block showing how to register a new one.
//
// This is the entry point the TUI (src/internal/tui/init) calls through the
// shared `Create func(name string) (string, error)` interface used by all 6
// mod screens, so its signature must not change. It's a thin wrapper over
// CreateGoMcpAt, which does the real work and additionally supports writing
// to a directory whose name differs from the Go module name (used by
// `barrakuda-mod-creator init --out`).
func CreateGoMcp(name string) (string, error) {
	return CreateGoMcpAt(name, name, false)
}

// CreateGoMcpAt scaffolds the Go MCP project at dir with Go module name
// moduleName. If dir already exists and is non-empty, it refuses to write
// unless force is true, returning a clear error instead of silently
// overwriting whatever's there. Exported so the non-interactive `init`
// flags (src/cmd/init.go) can call it directly, bypassing the TUI.
func CreateGoMcpAt(dir, moduleName string, force bool) (string, error) {
	if err := ensureEmptyOrForced(dir, force); err != nil {
		return "", err
	}

	manifestJSON, err := mcpManifestJSON(moduleName, []manifest.Tool{
		{
			Name: "greet", Description: "Greets someone by name and remembers how many times they've been greeted.",
			InputSchema:  greetInputSchema,
			Capabilities: []string{},
		},
		{
			Name: "reset_greeting_count", Description: "Resets the greet counter for a name back to zero.",
			InputSchema:  greetInputSchema,
			Capabilities: []string{},
		},
	})
	if err != nil {
		return "", err
	}

	return writeFiles(dir, map[string]string{
		"specs.json":   manifestJSON,
		"go.mod":          fmt.Sprintf(goMcpGoMod, moduleName),
		"README.md":       fmt.Sprintf(goMcpReadme, moduleName),
		"cmd/api/main.go": fmt.Sprintf(goMcpMain, moduleName, moduleName, moduleName),

		"internal/domain/greeting.go": goMcpDomain,

		"internal/repository/greeting_repository.go": goMcpRepository,

		"internal/service/greeting_service.go": fmt.Sprintf(goMcpService, moduleName, moduleName),

		"internal/mcp/server.go": fmt.Sprintf(goMcpServer, moduleName, moduleName),
	})
}

// greetInputSchema is the input_schema shared by the example tools (both
// take a single required "name" string) across all 3 MCP scaffolds.
var greetInputSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"name": map[string]any{"type": "string", "description": "Who to greet"},
	},
	"required": []string{"name"},
}

// mcpManifestJSON builds the shared specs.json content for all 3 MCP
// scaffolds (Go/Typescript/Python) — same shape regardless of language,
// since `entry` just names the OS-resolved executable/script the app spawns.
// tools must match exactly what the generated server's own tools/list
// handler reports — this is the reference example for manifest_version 2,
// so it declares real per-tool capabilities instead of leaving them out.
func mcpManifestJSON(name string, tools []manifest.Tool) (string, error) {
	m := manifest.Manifest{
		Name:            name,
		Version:         "0.0.1",
		Description:     name + " MCP server",
		License:         "MIT",
		Type:            manifest.TypeMCP,
		Tags:            []string{"barrakuda", "mcp"},
		ManifestVersion: 2,
		Server:          name,
		Entry:           name,
		Platforms:       []string{"windows", "linux", "macos"},
		Tools:           tools,
		Skill:           "./README.md",
	}
	m.RequiresPermission = m.DeriveLegacyPermission()
	return m.JSON()
}

// ensureEmptyOrForced returns an error if dir already exists and is not
// empty, unless force is true. It never creates or removes dir itself —
// writeFiles (via os.MkdirAll) creates it on the happy path.
func ensureEmptyOrForced(dir string, force bool) error {
	if force {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(entries) > 0 {
		return fmt.Errorf("directory %s already exists and is not empty — use --force", dir)
	}

	return nil
}

const goMcpGoMod = `module %s

go 1.26.4
`

const goMcpReadme = `# %s

Minimal MCP server over stdio. No dependencies. The MCP protocol itself
(JSON-RPC, newline-delimited) is hand-rolled in internal/mcp/server.go — no
SDK — so it's readable top to bottom. See
https://modelcontextprotocol.io/specification for the protocol itself.

## Layout

- internal/domain     — the Greeting type
- internal/repository — in-memory greet counter (swap for a real DB when needed)
- internal/service    — business logic (Greet, ResetGreetingCount)
- internal/mcp        — the MCP protocol: the stdio loop
- cmd/api             — wiring + entrypoint

## Run

    go run ./cmd/api

## Try it

Paste these lines (one JSON object per line) into the running process's stdin:

    {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"manual","version":"0"}}}
    {"jsonrpc":"2.0","method":"notifications/initialized"}
    {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"greet","arguments":{"name":"Ada"}}}

It exposes two example tools:

- greet(name) — greets someone and remembers how many times they've been greeted.
- reset_greeting_count(name) — resets that counter; demonstrates returning a
  client-visible error (isError: true) instead of a transport failure when
  name is empty.

## Adding a new tool

See the comment block at the top of internal/mcp/server.go — it walks through
adding a tool to the static table, writing its handler, and registering it.

## Optional: user-editable config

Drop a config.yaml next to specs.json to let users of this mod edit
settings from Barrakuda's mod library. Flat key/value pairs only — no
labels, descriptions, enums, or secret-masking, the widget is inferred
from each value's own YAML type (string/number/boolean):

    api_key: ""
    max_tokens: 4000
    modo: rapido

No config.yaml means no config icon shows up for this mod — it's entirely
optional.
`

const goMcpDomain = `package domain

// Greeting is the result of greeting someone by name.
type Greeting struct {
	Name    string
	Count   int
	Message string
}
`

const goMcpRepository = `package repository

import "sync"

// GreetingRepository counts how many times each name has been greeted.
// In-memory only — restarts reset the count. Swap for a real database when needed.
type GreetingRepository struct {
	mu     sync.Mutex
	counts map[string]int
}

func NewGreetingRepository() *GreetingRepository {
	return &GreetingRepository{counts: map[string]int{}}
}

func (r *GreetingRepository) Increment(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counts[name]++
	return r.counts[name]
}

// Reset clears the greet counter for name back to zero.
func (r *GreetingRepository) Reset(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.counts, name)
}
`

const goMcpService = `package service

import (
	"errors"
	"fmt"

	"%s/internal/domain"
	"%s/internal/repository"
)

type GreetingService struct {
	repo *repository.GreetingRepository
}

func NewGreetingService(repo *repository.GreetingRepository) *GreetingService {
	return &GreetingService{repo: repo}
}

func (s *GreetingService) Greet(name string) (domain.Greeting, error) {
	if name == "" {
		return domain.Greeting{}, errors.New("name is required")
	}

	count := s.repo.Increment(name)

	return domain.Greeting{
		Name:    name,
		Count:   count,
		Message: fmt.Sprintf("Hello, %%s! I've greeted you %%d time(s).", name, count),
	}, nil
}

// ResetGreetingCount resets the greet counter for name back to zero.
// Returns a validation error if name is empty — the caller (internal/mcp)
// turns that into a structured tool-result error, not a transport failure.
func (s *GreetingService) ResetGreetingCount(name string) error {
	if name == "" {
		return errors.New("name is required")
	}

	s.repo.Reset(name)
	return nil
}
`

const goMcpServer = `package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"%s/internal/service"
)

// Just enough of MCP's JSON-RPC shapes to serve two tools over stdio. No
// SDK — plain maps in, plain maps out. See
// https://modelcontextprotocol.io/specification for the full spec.

const protocolVersion = "2025-06-18"

var greetTool = map[string]any{
	"name":        "greet",
	"description": "Greets someone by name and remembers how many times they've been greeted.",
	"inputSchema": map[string]any{
		"type":       "object",
		"properties": map[string]any{"name": map[string]any{"type": "string", "description": "Who to greet"}},
		"required":   []string{"name"},
	},
}

var resetGreetingCountTool = map[string]any{
	"name":        "reset_greeting_count",
	"description": "Resets the greet counter for a name back to zero.",
	"inputSchema": map[string]any{
		"type":       "object",
		"properties": map[string]any{"name": map[string]any{"type": "string", "description": "Whose greet counter to reset"}},
		"required":   []string{"name"},
	},
}

// Serve reads newline-delimited JSON-RPC requests from stdin and writes
// responses to stdout: initialize, tools/list and tools/call, which is all a
// client needs to discover and call greet/reset_greeting_count. Blocks
// until stdin is closed.
func Serve(greetingService *service.GreetingService) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req map[string]any
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		method, _ := req["method"].(string)
		if method == "notifications/initialized" {
			continue
		}

		res := map[string]any{"jsonrpc": "2.0", "id": req["id"]}

		switch method {
		case "initialize":
			res["result"] = map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "%s", "version": "0.1.0"},
			}
		case "tools/list":
			res["result"] = map[string]any{"tools": []map[string]any{greetTool, resetGreetingCountTool}}
		case "tools/call":
			res["result"] = callTool(greetingService, req)
		default:
			res["error"] = map[string]any{"code": -32601, "message": fmt.Sprintf("method not found: %%s", method)}
		}

		out, err := json.Marshal(res)
		if err != nil {
			continue
		}
		fmt.Println(string(out))
	}
}

// ---------------------------------------------------------------------------
// Adding a new tool
// ---------------------------------------------------------------------------
//
// 1. Add its {name, description, inputSchema} to a var like greetTool above,
//    and to the "tools/list" case's slice in Serve.
// 2. Add a case for its name in callTool below, pulling arguments out of
//    args (the map under params["arguments"]).
// 3. Put any real logic behind internal/service, keeping this file limited
//    to request/response translation — the same domain/repository/service
//    layering the rest of the ecosystem (auth/payments) uses.
// ---------------------------------------------------------------------------

// callTool dispatches a tools/call request to the right handler and returns
// the MCP "content" result shape (or an isError:true result on validation
// failure — that's a client-visible tool error, not a transport failure).
func callTool(greetingService *service.GreetingService, req map[string]any) map[string]any {
	params, _ := req["params"].(map[string]any)
	toolName, _ := params["name"].(string)
	args, _ := params["arguments"].(map[string]any)
	argName, _ := args["name"].(string)

	switch toolName {
	case "greet":
		greeting, err := greetingService.Greet(argName)
		if err != nil {
			return errorResult(err.Error())
		}
		return textResult(greeting.Message)
	case "reset_greeting_count":
		if err := greetingService.ResetGreetingCount(argName); err != nil {
			return errorResult(fmt.Sprintf("validation failed: %%s", err.Error()))
		}
		return textResult(fmt.Sprintf("Greet counter for %%s reset to 0.", argName))
	default:
		return errorResult(fmt.Sprintf("unknown tool: %%s", toolName))
	}
}

func textResult(text string) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
}

func errorResult(text string) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": text}}, "isError": true}
}
`

const goMcpMain = `package main

import (
	"%s/internal/mcp"
	"%s/internal/repository"
	"%s/internal/service"
)

func main() {
	repo := repository.NewGreetingRepository()
	greetingService := service.NewGreetingService(repo)

	mcp.Serve(greetingService)
}
`
