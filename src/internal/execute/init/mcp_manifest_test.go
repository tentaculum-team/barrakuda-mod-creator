package execute_init

import (
	"encoding/json"
	"testing"

	"barrakudaModKit/internal/manifest"
)

func TestMcpManifestJSON_DeclaresVersion2AndToolCapabilities(t *testing.T) {
	tools := []manifest.Tool{
		{Name: "greet", Description: "Greets someone.", Capabilities: []string{}},
	}

	raw, err := mcpManifestJSON("gomcptest", tools)
	if err != nil {
		t.Fatalf("mcpManifestJSON: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("generated manifest.json is not valid JSON: %v", err)
	}

	if m.ManifestVersion != 2 {
		t.Fatalf("expected manifest_version 2, got %d", m.ManifestVersion)
	}
	if len(m.Tools) != 1 || m.Tools[0].Name != "greet" {
		t.Fatalf("expected the passed-in tools to round-trip, got %+v", m.Tools)
	}
	if m.Tools[0].Capabilities == nil {
		t.Fatalf("expected an explicit empty capabilities slice to survive the JSON round-trip as non-nil, got nil")
	}

	if warnings, err := m.Validate(); err != nil || len(warnings) != 0 {
		t.Fatalf("generated manifest should validate clean under its own manifest_version: warnings=%v err=%v", warnings, err)
	}
}
