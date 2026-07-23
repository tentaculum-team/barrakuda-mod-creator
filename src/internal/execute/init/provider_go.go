package execute_init

import (
	"fmt"

	"barrakudaModKit/internal/manifest"
)

// CreateGoProvider scaffolds a Go provider mod: a stdin/stdout NDJSON loop
// speaking the protocol documented in docs/manifest-schema.md#type-provider —
// one request line in, a stream of delta/done (or error) lines out, invoked
// once per chat turn (not long-lived), mirroring how `claude -p` itself is a
// one-shot-per-prompt CLI.
//
// This is the entry point the TUI (src/internal/tui/init) calls through the
// shared `Create func(name string) (string, error)` interface, so its
// signature must not change. Thin wrapper over CreateGoProviderAt, which does
// the real work and additionally supports writing to a directory whose name
// differs from the Go module name (used by `barrakuda-mod-creator init --out`).
func CreateGoProvider(name string) (string, error) {
	return CreateGoProviderAt(name, name, false)
}

// CreateGoProviderAt scaffolds the Go provider project at dir with Go module
// name moduleName. If dir already exists and is non-empty, it refuses to
// write unless force is true. Exported so the non-interactive `init` flags
// (src/cmd/init.go) can call it directly, bypassing the TUI.
func CreateGoProviderAt(dir, moduleName string, force bool) (string, error) {
	if err := ensureEmptyOrForced(dir, force); err != nil {
		return "", err
	}

	manifestJSON, err := providerManifestJSON(moduleName)
	if err != nil {
		return "", err
	}

	return writeFiles(dir, map[string]string{
		"manifest.json":               manifestJSON,
		"go.mod":                      fmt.Sprintf(goProviderGoMod, moduleName),
		"README.md":                   fmt.Sprintf(goProviderReadme, moduleName),
		"cmd/api/main.go":             fmt.Sprintf(goProviderMain, moduleName),
		"internal/provider/server.go": goProviderServer,
	})
}

// providerManifestJSON builds the manifest.json content for the Go provider
// scaffold. `entry` matches the module name — the server resolves the OS
// suffix (`.exe` on Windows) itself, same convention as `mcp`'s `entry`.
func providerManifestJSON(name string) (string, error) {
	m := manifest.Manifest{
		Name:            name,
		Version:         "0.0.1",
		Description:     name + " provider mod",
		License:         "MIT",
		Type:            manifest.TypeProvider,
		Tags:            []string{"barrakuda", "provider"},
		ProviderID:      name,
		Entry:           name,
		Platforms:       []string{"windows", "linux", "macos"},
		ProtocolVersion: 1,
		Skill:           "./README.md",
	}
	return m.JSON()
}

const goProviderGoMod = `module %s

go 1.26.4
`

const goProviderReadme = `# %s

Minimal model-generation provider over stdio. No dependencies. Speaks the
NDJSON protocol documented in barrakuda-mod-creator's
docs/manifest-schema.md#type-provider — hand-rolled in
internal/provider/server.go, no SDK.

This mod type is sideload-only: barrakuda-server spawns this binary on its
own host once per chat turn, on the same host running the server (not the
client machine), and an admin registers its resolved path directly in the
server's provider_mods table — there is no store/install flow for it.

## Run

    go run ./cmd/api

## Try it

Paste one JSON line into the running process's stdin, then press enter —
the process reads exactly one request, streams its response, and exits:

    {"messages":[{"role":"user","content":"hi"}],"tools":[],"params":{"temperature":0.7,"seed":""}}

It streams back:

    {"delta":"..."}
    {"delta":"..."}
    {"done":true,"input_tokens":1,"output_tokens":2,"tool_calls":[]}

## Adapting to a real backend

See the comment block at the top of internal/provider/server.go's respond
function — replace the demo echo logic with a real call to whatever local
model backend this mod wraps (e.g. shelling out to a CLI and translating its
own streaming output into delta/done lines).
`

const goProviderMain = `package main

import "%s/internal/provider"

func main() {
	provider.Serve()
}
`

const goProviderServer = `package provider

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Message/Tool/Params/Request mirror the neutral shapes barrakuda-server
// uses internally (internal/providers/types.go) — this mod is a standalone
// binary, so it can't import that package directly, but the JSON shape on
// the wire must match exactly.

type Message struct {
	Role    string ` + "`json:\"role\"`" + `
	Content string ` + "`json:\"content\"`" + `
}

type Tool struct {
	Name        string          ` + "`json:\"name\"`" + `
	Description string          ` + "`json:\"description\"`" + `
	InputSchema json.RawMessage ` + "`json:\"input_schema\"`" + `
}

type Params struct {
	Temperature float64 ` + "`json:\"temperature\"`" + `
	Seed        string  ` + "`json:\"seed\"`" + `
}

type Request struct {
	Messages []Message ` + "`json:\"messages\"`" + `
	Tools    []Tool    ` + "`json:\"tools\"`" + `
	Params   Params    ` + "`json:\"params\"`" + `
}

// Serve reads exactly one NDJSON request line from stdin, streams the
// response as delta/done (or error) lines to stdout, then returns — the
// protocol is invocation-per-turn, not a long-lived loop.
func Serve() {
	reader := bufio.NewReaderSize(os.Stdin, 1<<20)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		writeError(fmt.Sprintf("failed to read request: %s", err))
		return
	}

	var req Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		writeError(fmt.Sprintf("invalid request JSON: %s", err))
		return
	}

	inputTokens, outputTokens, err := respond(req, emitDelta)
	if err != nil {
		writeError(err.Error())
		return
	}
	writeDone(inputTokens, outputTokens)
}

// ---------------------------------------------------------------------------
// Adapting to a real backend
// ---------------------------------------------------------------------------
//
// Replace this demo body with a real call to whatever local model backend
// this mod wraps. For example, a CLI-backed provider would: flatten
// req.Messages into that CLI's own prompt format, shell out to it, and
// translate its native streaming output into emit(delta) calls as it
// arrives, returning real input/output token counts at the end instead of
// the word-count approximation below.
// ---------------------------------------------------------------------------

// respond is the demo backend: echoes the last user message back one word
// at a time, proving the delta/done wiring end to end. inputTokens/
// outputTokens here are a word-count stand-in — a real backend reports
// whatever usage numbers its own API/CLI gives it.
func respond(req Request, emit func(string)) (inputTokens, outputTokens int, err error) {
	lastUser := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUser = req.Messages[i].Content
			break
		}
	}
	if lastUser == "" {
		return 0, 0, fmt.Errorf("no user message in request")
	}

	words := strings.Fields(lastUser)
	inputTokens = len(words)

	reply := append([]string{"echo:"}, words...)
	for i, w := range reply {
		if i > 0 {
			emit(" ")
		}
		emit(w)
	}
	outputTokens = len(reply)

	return inputTokens, outputTokens, nil
}

func emitDelta(delta string) {
	writeLine(map[string]any{"delta": delta})
}

func writeDone(inputTokens, outputTokens int) {
	writeLine(map[string]any{
		"done":          true,
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"tool_calls":    []any{},
	})
}

func writeError(message string) {
	writeLine(map[string]any{"error": message})
}

func writeLine(v map[string]any) {
	out, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Println(string(out))
}
`
