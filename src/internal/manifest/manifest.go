// Package manifest defines the mod manifest shape shared by every scaffold
// (mcp/skill/agent/theme) and the runtime that consumes it (barrakuda-software).
// See docs/manifest-schema.md for the full field-by-field spec.
package manifest

import (
	"encoding/json"
	"fmt"
)

// Type discriminates what a mod does. The runtime picks its execution path
// (spawn an MCP subprocess, build/run a Docker container, render markdown,
// apply theme tokens) based on this field alone.
type Type string

const (
	TypeMCP      Type = "mcp"
	TypeSkill    Type = "skill"
	TypeAgent    Type = "agent"
	TypeTheme    Type = "theme"
	TypeProvider Type = "provider"
)

// Manifest is the full mod manifest. Fields not relevant to a given Type are
// simply omitted from the JSON (all type-specific fields are optional).
type Manifest struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Banners     []string `json:"banners,omitempty"`
	Author      string   `json:"author,omitempty"`
	License     string   `json:"license,omitempty"`
	Type        Type     `json:"type"`
	Tags        []string `json:"tags,omitempty"`

	// ManifestVersion gates the structured-capability checks in Validate:
	// absent/0/1 = legacy shape (tools[].capabilities not required), 2 = new
	// shape (every mcp tool must declare capabilities, even if empty).
	ManifestVersion int `json:"manifest_version,omitempty"`

	// mcp/provider-only
	Server                  string            `json:"server,omitempty"`
	Entry                   string            `json:"entry,omitempty"`
	Platforms               []string          `json:"platforms,omitempty"`
	Tools                   []Tool            `json:"tools,omitempty"`
	RequiresPermission      string            `json:"requires_permission,omitempty"`
	RequiresPermissionLabel string            `json:"requires_permission_label,omitempty"`
	Dependencies            map[string]string `json:"dependencies,omitempty"`
	Sell                    *Sell             `json:"sell,omitempty"`

	// provider-only
	ProviderID      string `json:"provider_id,omitempty"`
	ProtocolVersion int    `json:"protocol_version,omitempty"`

	// mcp/agent share a single doc page; skill has many (Skills below)
	Skill string `json:"skill,omitempty"`

	// agent-only
	Docker *DockerConfig `json:"docker,omitempty"`

	// skill-only
	Skills []ModDoc `json:"skills,omitempty"`

	// theme-only — colors/fonts: token name -> CSS value; icons: token
	// name -> inline SVG markup (no <svg> wrapper), sanitized by the app
	// before rendering.
	Colors map[string]string `json:"colors,omitempty"`
	Fonts  map[string]string `json:"fonts,omitempty"`
	Icons  map[string]string `json:"icons,omitempty"`

	// Accepted for forward-compatibility, no execution engine reads these
	// yet — see docs/manifest-schema.md "contributions/shortcuts" section.
	Contributions []Contribution `json:"contributions,omitempty"`
	Shortcuts     []Shortcut     `json:"shortcuts,omitempty"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema,omitempty"`

	// Capabilities is the structured replacement for the free-text
	// "[fsRead]"/"[fsWrite]" prefix convention in Description — required
	// (may be an explicit empty slice) once ManifestVersion >= 2. Values
	// must come from the closed vocabulary below: it's deliberately coarse,
	// matching what an OS-level sandbox can actually enforce, not a
	// free-form permission string.
	//
	// Deliberately no `omitempty`: it would make Marshal drop an explicit
	// `[]` back down to absent, indistinguishable from "not declared" on
	// the next Unmarshal — the whole point of nil vs [] in Validate.
	Capabilities []string `json:"capabilities"`
}

// Closed capability vocabulary for Tool.Capabilities. Kept small on purpose:
// each value must map to something a sandbox can actually scope (a folder
// ACL, a capability SID, ...), not an arbitrary free-form string.
const (
	CapFSRead       = "fs:read"
	CapFSWrite      = "fs:write"
	CapFSDelete     = "fs:delete"
	CapNetwork      = "network"
	CapShellExec    = "shell:exec"
	CapProcessSpawn = "process:spawn"
)

var knownCapabilities = map[string]bool{
	CapFSRead:       true,
	CapFSWrite:      true,
	CapFSDelete:     true,
	CapNetwork:      true,
	CapShellExec:    true,
	CapProcessSpawn: true,
}

type Sell struct {
	Price   int64  `json:"price"`
	Coin    string `json:"coin"`
	Monthly bool   `json:"monthly"`
}

type DockerConfig struct {
	Dockerfile string            `json:"dockerfile"`
	Ports      []DockerPort      `json:"ports,omitempty"`
	Resources  DockerResources   `json:"resources,omitzero"`
	Env        map[string]string `json:"env,omitempty"`
}

type DockerPort struct {
	Host      int `json:"host"`
	Container int `json:"container"`
}

type DockerResources struct {
	CPU    float64 `json:"cpu,omitempty"`
	Memory string  `json:"memory,omitempty"`
}

// ModDoc is one named content page (skill-type's `skills[]`).
type ModDoc struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path"`
}

type Contribution struct {
	Type     string            `json:"type"`
	Label    string            `json:"label,omitempty"`
	Position string            `json:"position,omitempty"`
	CSS      map[string]string `json:"css,omitempty"`
	Actions  []Action          `json:"actions,omitempty"`
}

type Action struct {
	Type   string `json:"type"`
	Value  string `json:"value,omitempty"`
	Target string `json:"target,omitempty"`
}

type Shortcut struct {
	Name   string `json:"name"`
	Action string `json:"action"`
	Key    string `json:"key"`
}

// Validate checks the fields required for m.Type are present, returning any
// non-fatal warnings alongside a fatal error. It does not check files
// referenced by relative path (icon/banners/skill/skills[].path/
// docker.dockerfile) actually exist on disk — callers that have a project
// root available (e.g. the `validate` CLI command) should check those
// themselves.
func (m *Manifest) Validate() ([]string, error) {
	if m.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return nil, fmt.Errorf("version is required")
	}

	switch m.Type {
	case TypeMCP:
		if m.Server == "" {
			return nil, fmt.Errorf("mcp manifest requires server")
		}
		if m.Entry == "" {
			return nil, fmt.Errorf("mcp manifest requires entry")
		}
		return m.validateToolCapabilities()
	case TypeAgent:
		if m.Docker == nil || m.Docker.Dockerfile == "" {
			return nil, fmt.Errorf("agent manifest requires docker.dockerfile")
		}
	case TypeSkill:
		if len(m.Skills) == 0 {
			return nil, fmt.Errorf("skill manifest requires at least one entry in skills")
		}
	case TypeTheme:
		// no fields beyond the base ones are required
	case TypeProvider:
		if m.ProviderID == "" {
			return nil, fmt.Errorf("provider manifest requires provider_id")
		}
		if m.Entry == "" {
			return nil, fmt.Errorf("provider manifest requires entry")
		}
		if len(m.Platforms) == 0 {
			return nil, fmt.Errorf("provider manifest requires platforms")
		}
		if m.ProtocolVersion == 0 {
			return nil, fmt.Errorf("provider manifest requires protocol_version")
		}
	default:
		return nil, fmt.Errorf("unknown type %q: must be one of mcp, skill, agent, theme, provider", m.Type)
	}

	return nil, nil
}

// validateToolCapabilities enforces the structured tools[].capabilities
// convention for ManifestVersion >= 2, and warns (without failing) about
// manifests that haven't migrated yet.
func (m *Manifest) validateToolCapabilities() ([]string, error) {
	if m.ManifestVersion < 2 {
		return []string{"manifest_version < 2: capabilities not declared, falling back to legacy requires_permission gating"}, nil
	}

	for _, t := range m.Tools {
		if t.Capabilities == nil {
			return nil, fmt.Errorf("tool %q must declare capabilities (use [] if it needs none) once manifest_version is 2", t.Name)
		}
		for _, cap := range t.Capabilities {
			if !knownCapabilities[cap] {
				return nil, fmt.Errorf("tool %q declares unknown capability %q", t.Name, cap)
			}
		}
	}

	return nil, nil
}

// DeriveLegacyPermission computes the coarse requires_permission fallback
// from tools[].capabilities, so a manifest_version 2 manifest still gates
// correctly against a host that only reads the legacy field.
func (m *Manifest) DeriveLegacyPermission() string {
	sawWrite := false
	sawRead := false
	for _, t := range m.Tools {
		for _, cap := range t.Capabilities {
			switch cap {
			case CapFSWrite, CapFSDelete:
				sawWrite = true
			case CapFSRead:
				sawRead = true
			}
		}
	}
	switch {
	case sawWrite:
		return "fsWrite"
	case sawRead:
		return "fsRead"
	default:
		return ""
	}
}

// JSON renders m as indented JSON with a trailing newline, ready to write to
// a mod's manifest.json.
func (m *Manifest) JSON() (string, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
