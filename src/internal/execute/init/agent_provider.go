package execute_init

import (
	"fmt"

	"barrakudaModKit/internal/manifest"
)

// CreateAgentProvider scaffolds a chat-selectable agent mod: a Docker
// container barrakuda-software builds/runs client-side (same pipeline as a
// plain `agent` mod), exposing the inbound chat contract documented in
// docs/manifest-schema.md#agent_provider — GET /barrakuda/v1/health and
// POST /barrakuda/v1/chat (SSE delta/done/error lines). Distinct from
// CreateAgent: this variant declares the agent_provider manifest block that
// makes barrakuda-software's chat AI picker treat the mod as a selectable
// agent instead of a plain generic sidecar.
//
// This is the entry point the TUI (src/internal/tui/init) calls through the
// shared `Create func(name string) (string, error)` interface, so its
// signature must not change. Thin wrapper over CreateAgentProviderAt.
func CreateAgentProvider(name string) (string, error) {
	return CreateAgentProviderAt(name, name, false)
}

// CreateAgentProviderAt scaffolds the project at dir with Go module name
// moduleName (the stub HTTP server is a real Go binary, built inside the
// Dockerfile). If dir already exists and is non-empty, it refuses to write
// unless force is true.
func CreateAgentProviderAt(dir, moduleName string, force bool) (string, error) {
	if err := ensureEmptyOrForced(dir, force); err != nil {
		return "", err
	}

	m := manifest.Manifest{
		Name:        moduleName,
		Version:     "0.0.1",
		Description: moduleName + " agent mod",
		License:     "MIT",
		Type:        manifest.TypeAgent,
		Tags:        []string{"barrakuda", "agent"},
		Skill:       "./README.md",
		Docker: &manifest.DockerConfig{
			Dockerfile: "./docker/Dockerfile",
			Ports:      []manifest.DockerPort{{Host: 0, Container: 8080}},
			Resources:  manifest.DockerResources{CPU: 1, Memory: "512m"},
			Env: map[string]string{
				"RELAY_BASE_URL": "$relay-base-url",
				"RELAY_TOKEN":    "$relay-token",
			},
			Volumes: []manifest.DockerVolume{{Name: "memory", Container: "/data"}},
		},
		AgentProvider: &manifest.AgentProviderConfig{ChatPort: 8080},
	}

	manifestJSON, err := m.JSON()
	if err != nil {
		return "", err
	}

	return writeFiles(dir, map[string]string{
		"specs.json":               manifestJSON,
		"go.mod":                   fmt.Sprintf(agentProviderGoMod, moduleName),
		"README.md":                fmt.Sprintf(agentProviderReadme, moduleName),
		"cmd/api/main.go":          agentProviderMain,
		"internal/agent/server.go": agentProviderServer,
		"docker/Dockerfile":        agentProviderDockerfile,
	})
}

const agentProviderGoMod = `module %s

go 1.26.4
`

const agentProviderReadme = `# %s

Chat-selectable agent mod. Runs client-side as a Docker container
(barrakuda-software builds/runs it, same pipeline as a plain ` + "`agent`" + `
mod) and appears in the chat AI picker because specs.json declares
` + "`agent_provider`" + `.

## Inbound contract (barrakuda-software calls this container)

- ` + "`GET /barrakuda/v1/health`" + ` — 200 once ready to serve chat.
- ` + "`POST /barrakuda/v1/chat`" + ` — ` + "`{conversation_id, message, model, attachments[]}`" + `,
  responds ` + "`text/event-stream`" + ` with ` + "`{\"delta\":...}`" + ` /
  ` + "`{\"done\":true}`" + ` / ` + "`{\"error\":...}`" + ` lines.

See internal/agent/server.go — replace the demo echo logic with whatever
upstream agent framework this mod wraps.

## Outbound (this agent's own LLM calls)

RELAY_BASE_URL/RELAY_TOKEN (env, resolved from the ` + "`$relay-base-url`" + `/
` + "`$relay-token`" + ` placeholders in specs.json) point at
barrakuda-software's local OpenAI-compatible relay, which forwards to the
real barrakuda-server using the model the user picked for this agent. Never
call a model provider directly — point whatever upstream framework this mod
wraps at RELAY_BASE_URL as its "custom OpenAI-compatible endpoint".

## Optional: real tool access (shell/fs/docker on the user's machine)

By default this agent's LLM calls (via RELAY_BASE_URL) can only produce
text — it can't run any tool on the user's machine. To let the user opt
in, declare the reserved key in config.yaml (off unless the user
explicitly turns it on in Barrakuda's mod config dialog — the gear icon in
the library, no code on your side needed to expose that toggle):

    allow_machine_access: false

See docs/manifest-schema.md#allow_machine_access.

## Persistent memory

The ` + "`memory`" + ` volume (specs.json ` + "`docker.volumes`" + `) mounts to
` + "`/data`" + ` in the container and survives restarts — put any
upstream-framework state (SQLite, skill files, ...) there.

## Run locally

    go run ./cmd/api
`

const agentProviderMain = `package main

import "agentprovider/internal/agent"

func main() {
	agent.Serve()
}
`

const agentProviderServer = `package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
)

// ChatRequest/Attachment mirror the inbound contract documented in
// docs/manifest-schema.md#agent_provider.
type ChatRequest struct {
	ConversationID string       ` + "`json:\"conversation_id\"`" + `
	Message        string       ` + "`json:\"message\"`" + `
	Model          string       ` + "`json:\"model\"`" + `
	Attachments    []Attachment ` + "`json:\"attachments\"`" + `
}

type Attachment struct {
	Name     string ` + "`json:\"name\"`" + `
	MimeType string ` + "`json:\"mime_type\"`" + `
	Data     string ` + "`json:\"data\"`" + `
}

// Serve starts the inbound HTTP+SSE server on :8080 (must match
// specs.json's agent_provider.chat_port / docker.ports[].container).
func Serve() {
	http.HandleFunc("/barrakuda/v1/health", handleHealth)
	http.HandleFunc("/barrakuda/v1/chat", handleChat)
	fmt.Println("agent listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("server error:", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// ---------------------------------------------------------------------------
// Adapting to a real agent framework
// ---------------------------------------------------------------------------
//
// Replace this demo body with a real call into whatever upstream agent
// framework this mod wraps (its non-interactive/scripted invocation mode,
// or its own local API if it has one) — driving that framework's own LLM
// calls at RELAY_BASE_URL (see README.md), and translating its native
// streaming output into writeDelta calls as it arrives.
// ---------------------------------------------------------------------------

func handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	writer := bufio.NewWriter(w)
	writeDelta(writer, "echo: "+req.Message)
	flusher.Flush()
	writeDone(writer)
	flusher.Flush()
}

func writeDelta(w *bufio.Writer, delta string) {
	writeEvent(w, map[string]any{"delta": delta})
}

func writeDone(w *bufio.Writer) {
	writeEvent(w, map[string]any{"done": true})
}

func writeEvent(w *bufio.Writer, v map[string]any) {
	out, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", out)
	w.Flush()
}
`

const agentProviderDockerfile = `FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /agent ./cmd/api

FROM alpine:3.20
COPY --from=build /agent /agent
CMD ["/agent"]
`
