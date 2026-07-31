// Package modlayout resolves where a mod's declared files actually live on
// disk. Mods have moved through two directory shapes over time — a legacy
// flat layout (specs at manifest.json/specs.json, icon.svg at the repo
// root) and the current .barrakuda/ layout (.barrakuda/specs.json,
// .barrakuda/config.yaml, .barrakuda/icon.svg) — so every caller that needs
// to find these files goes through the same fallback order instead of
// re-implementing it.
package modlayout

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveSpecsPath finds a mod's manifest file under dir, trying
// .barrakuda/specs.json, then specs.json, then the legacy manifest.json.
// legacy is true only for the manifest.json fallback. Returns an error if
// none of the three exist.
func ResolveSpecsPath(dir string) (path string, legacy bool, err error) {
	path = filepath.Join(dir, ".barrakuda", "specs.json")
	if _, statErr := os.Stat(path); statErr == nil {
		return path, false, nil
	}
	path = filepath.Join(dir, "specs.json")
	if _, statErr := os.Stat(path); statErr == nil {
		return path, false, nil
	}
	path = filepath.Join(dir, "manifest.json")
	if _, statErr := os.Stat(path); statErr == nil {
		return path, true, nil
	}
	return "", false, fmt.Errorf("no specs.json, .barrakuda/specs.json, or manifest.json found under %s", dir)
}

// ResolveConfigPath finds a mod's config.yaml, trying .barrakuda/config.yaml
// then config.yaml at the root. found is false if neither exists — a mod
// without runtime config is valid, unlike a mod without specs.
func ResolveConfigPath(dir string) (path string, found bool) {
	path = filepath.Join(dir, ".barrakuda", "config.yaml")
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	path = filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}

// ResolveIconPath finds a mod's icon.svg, trying .barrakuda/icon.svg then
// icon.svg at the root (the legacy layout, e.g. barrakuda-mod-claude-cli,
// keeps it there). found is false if neither exists.
func ResolveIconPath(dir string) (path string, found bool) {
	path = filepath.Join(dir, ".barrakuda", "icon.svg")
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	path = filepath.Join(dir, "icon.svg")
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}
