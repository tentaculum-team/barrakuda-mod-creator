package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		m       Manifest
		wantErr bool
	}{
		{"mcp valid", Manifest{Name: "n", Version: "0.0.1", Type: TypeMCP, Server: "n", Entry: "n"}, false},
		{"mcp missing server", Manifest{Name: "n", Version: "0.0.1", Type: TypeMCP, Entry: "n"}, true},
		{"mcp missing entry", Manifest{Name: "n", Version: "0.0.1", Type: TypeMCP, Server: "n"}, true},
		{"agent valid", Manifest{Name: "n", Version: "0.0.1", Type: TypeAgent, Docker: &DockerConfig{Dockerfile: "./Dockerfile"}}, false},
		{"agent missing docker", Manifest{Name: "n", Version: "0.0.1", Type: TypeAgent}, true},
		{"agent_provider chat_port matches docker.ports", Manifest{Name: "n", Version: "0.0.1", Type: TypeAgent,
			Docker:        &DockerConfig{Dockerfile: "./Dockerfile", Ports: []DockerPort{{Container: 8080}}},
			AgentProvider: &AgentProviderConfig{ChatPort: 8080}}, false},
		{"agent_provider missing chat_port", Manifest{Name: "n", Version: "0.0.1", Type: TypeAgent,
			Docker:        &DockerConfig{Dockerfile: "./Dockerfile", Ports: []DockerPort{{Container: 8080}}},
			AgentProvider: &AgentProviderConfig{}}, true},
		{"agent_provider chat_port not exposed in docker.ports", Manifest{Name: "n", Version: "0.0.1", Type: TypeAgent,
			Docker:        &DockerConfig{Dockerfile: "./Dockerfile", Ports: []DockerPort{{Container: 8080}}},
			AgentProvider: &AgentProviderConfig{ChatPort: 9999}}, true},
		{"skill valid", Manifest{Name: "n", Version: "0.0.1", Type: TypeSkill, Skills: []ModDoc{{Name: "p", Path: "./p.md"}}}, false},
		{"skill missing skills", Manifest{Name: "n", Version: "0.0.1", Type: TypeSkill}, true},
		{"theme valid", Manifest{Name: "n", Version: "0.0.1", Type: TypeTheme}, false},
		{"provider valid", Manifest{Name: "n", Version: "0.0.1", Type: TypeProvider, ProviderID: "p", Entry: "p", Platforms: []string{"windows"}, ProtocolVersion: 1}, false},
		{"provider missing provider_id", Manifest{Name: "n", Version: "0.0.1", Type: TypeProvider, Entry: "p", Platforms: []string{"windows"}, ProtocolVersion: 1}, true},
		{"provider missing entry", Manifest{Name: "n", Version: "0.0.1", Type: TypeProvider, ProviderID: "p", Platforms: []string{"windows"}, ProtocolVersion: 1}, true},
		{"provider missing platforms", Manifest{Name: "n", Version: "0.0.1", Type: TypeProvider, ProviderID: "p", Entry: "p", ProtocolVersion: 1}, true},
		{"provider missing protocol_version", Manifest{Name: "n", Version: "0.0.1", Type: TypeProvider, ProviderID: "p", Entry: "p", Platforms: []string{"windows"}}, true},
		{"missing name", Manifest{Version: "0.0.1", Type: TypeTheme}, true},
		{"missing version", Manifest{Name: "n", Type: TypeTheme}, true},
		{"unknown type", Manifest{Name: "n", Version: "0.0.1", Type: "bogus"}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.m.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidate_ManifestVersion2Capabilities(t *testing.T) {
	cases := []struct {
		name        string
		m           Manifest
		wantErr     bool
		wantWarning bool
	}{
		{
			name: "v2 mcp with declared capabilities passes clean",
			m: Manifest{
				Name: "n", Version: "0.0.1", Type: TypeMCP, Server: "n", Entry: "n",
				ManifestVersion: 2,
				Tools:           []Tool{{Name: "t", Capabilities: []string{CapFSRead}}},
			},
			wantErr: false, wantWarning: false,
		},
		{
			name: "v2 mcp tool declaring no capabilities explicitly (empty slice) passes clean",
			m: Manifest{
				Name: "n", Version: "0.0.1", Type: TypeMCP, Server: "n", Entry: "n",
				ManifestVersion: 2,
				Tools:           []Tool{{Name: "t", Capabilities: []string{}}},
			},
			wantErr: false, wantWarning: false,
		},
		{
			name: "v2 mcp tool with omitted capabilities (nil) errors",
			m: Manifest{
				Name: "n", Version: "0.0.1", Type: TypeMCP, Server: "n", Entry: "n",
				ManifestVersion: 2,
				Tools:           []Tool{{Name: "t"}},
			},
			wantErr: true,
		},
		{
			name: "v2 mcp tool with unknown capability errors",
			m: Manifest{
				Name: "n", Version: "0.0.1", Type: TypeMCP, Server: "n", Entry: "n",
				ManifestVersion: 2,
				Tools:           []Tool{{Name: "t", Capabilities: []string{"fs:teleport"}}},
			},
			wantErr: true,
		},
		{
			name: "legacy manifest (no manifest_version) with omitted capabilities only warns",
			m: Manifest{
				Name: "n", Version: "0.0.1", Type: TypeMCP, Server: "n", Entry: "n",
				Tools: []Tool{{Name: "t"}},
			},
			wantErr: false, wantWarning: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			warnings, err := c.m.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.wantWarning && len(warnings) == 0 {
				t.Fatalf("expected a warning, got none")
			}
			if !c.wantWarning && !c.wantErr && len(warnings) != 0 {
				t.Fatalf("expected no warnings, got: %v", warnings)
			}
		})
	}
}

func TestDeriveLegacyPermission(t *testing.T) {
	cases := []struct {
		name string
		m    Manifest
		want string
	}{
		{"write capability wins", Manifest{Tools: []Tool{{Capabilities: []string{CapFSRead}}, {Capabilities: []string{CapFSWrite}}}}, "fsWrite"},
		{"delete counts as write", Manifest{Tools: []Tool{{Capabilities: []string{CapFSDelete}}}}, "fsWrite"},
		{"read only", Manifest{Tools: []Tool{{Capabilities: []string{CapFSRead}}}}, "fsRead"},
		{"no fs capabilities", Manifest{Tools: []Tool{{Capabilities: []string{CapNetwork}}}}, ""},
		{"no tools", Manifest{}, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.m.DeriveLegacyPermission()
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestExamplesValidate proves the example manifests under ../../../examples
// are both strict JSON and pass Validate() for their declared type — the
// examples are meant to double as real fixtures, not just illustration.
func TestExamplesValidate(t *testing.T) {
	files := []string{
		"mod.barrakuda.agent.json",
		"mod.barrakuda.mcp.json",
		"mod.barrakuda.skill.json",
		"mod.barrakuda.provider.json",
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "examples", f)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatalf("%s is not strict JSON: %v", path, err)
			}

			if _, err := m.Validate(); err != nil {
				t.Fatalf("%s failed validation: %v", path, err)
			}
		})
	}
}
