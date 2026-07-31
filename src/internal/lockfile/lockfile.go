// Package lockfile computes a content-integrity lock for a mod: a hash
// covering its specs.json, config.yaml (if present), icon.svg (if present),
// and every source file, so that ANY change to any of those inputs changes
// the lock's hash. This is a different concern from internal/signing, which
// signs a built release zip for publisher provenance and deliberately
// EXCLUDES config.yaml (mutable by design, overwritten locally by the
// client). The lock instead exists to catch drift for CI ("did anything in
// this mod change since the committed lock was generated?"), so it
// deliberately INCLUDES config.yaml — a config edit should bust the lock.
package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"barrakudaModKit/internal/modlayout"
)

const (
	Algorithm   = "sha256"
	LockVersion = 1
	fileName    = "mod.lock"
)

// Component describes one named input (specs, config, icon) that fed into
// the lock. Hash is empty when Included is false.
type Component struct {
	Path     string `json:"path,omitempty"`
	Included bool   `json:"included"`
	Hash     string `json:"hash,omitempty"`
}

// FileHash is one file's contribution to the source component.
type FileHash struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// SourceComponent covers every source file under the mod root, excluding
// the named components above and the lock file itself.
type SourceComponent struct {
	Included  bool       `json:"included"`
	FileCount int        `json:"file_count"`
	Hash      string     `json:"hash,omitempty"`
	Files     []FileHash `json:"files,omitempty"`
}

type components struct {
	Specs  Component       `json:"specs"`
	Config Component       `json:"config"`
	Icon   Component       `json:"icon"`
	Source SourceComponent `json:"source"`
}

// Lock is the full content of a mod.lock file.
type Lock struct {
	Algorithm   string     `json:"algorithm"`
	LockVersion int        `json:"lock_version"`
	Components  components `json:"components"`
	Hash        string     `json:"hash"`
}

// Path returns where dir's mod.lock lives: next to specs.json, whichever
// layout that resolved to (.barrakuda/mod.lock or a root mod.lock for the
// legacy layout).
func Path(dir string) (string, error) {
	specsPath, _, err := modlayout.ResolveSpecsPath(dir)
	if err != nil {
		return "", err
	}
	if filepath.Dir(specsPath) == filepath.Join(dir, ".barrakuda") {
		return filepath.Join(dir, ".barrakuda", fileName), nil
	}
	return filepath.Join(dir, fileName), nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func resolveComponent(path string, found bool, dir string) (Component, error) {
	if !found {
		return Component{Included: false}, nil
	}
	hash, err := hashFile(path)
	if err != nil {
		return Component{}, err
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		rel = path
	}
	return Component{Path: filepath.ToSlash(rel), Included: true, Hash: hash}, nil
}

// Generate computes a fresh Lock for the mod rooted at dir. specs.json (or
// its legacy equivalent) must exist; config.yaml and icon.svg are hashed
// only if present — a mod without them is valid, the lock just marks that
// component Included: false. Adding either file later naturally flips
// Included to true and changes the hash, exactly like editing an existing
// input does.
func Generate(dir string) (*Lock, error) {
	specsPath, _, err := modlayout.ResolveSpecsPath(dir)
	if err != nil {
		return nil, fmt.Errorf("lockfile: %w", err)
	}
	specsComp, err := resolveComponent(specsPath, true, dir)
	if err != nil {
		return nil, fmt.Errorf("lockfile: hashing specs: %w", err)
	}

	configPath, configFound := modlayout.ResolveConfigPath(dir)
	configComp, err := resolveComponent(configPath, configFound, dir)
	if err != nil {
		return nil, fmt.Errorf("lockfile: hashing config: %w", err)
	}

	iconPath, iconFound := modlayout.ResolveIconPath(dir)
	iconComp, err := resolveComponent(iconPath, iconFound, dir)
	if err != nil {
		return nil, fmt.Errorf("lockfile: hashing icon: %w", err)
	}

	lockPath, err := Path(dir)
	if err != nil {
		return nil, fmt.Errorf("lockfile: %w", err)
	}
	exclude := map[string]bool{
		filepath.Clean(specsPath): true,
		filepath.Clean(lockPath):  true,
	}
	if configFound {
		exclude[filepath.Clean(configPath)] = true
	}
	if iconFound {
		exclude[filepath.Clean(iconPath)] = true
	}

	files, err := walkSource(dir, exclude)
	if err != nil {
		return nil, fmt.Errorf("lockfile: walking source: %w", err)
	}

	sourceHash := ""
	if len(files) > 0 {
		sourceHash = hashFileList(files)
	}

	lock := &Lock{
		Algorithm:   Algorithm,
		LockVersion: LockVersion,
		Components: components{
			Specs:  specsComp,
			Config: configComp,
			Icon:   iconComp,
			Source: SourceComponent{
				Included:  len(files) > 0,
				FileCount: len(files),
				Hash:      sourceHash,
				Files:     files,
			},
		},
	}

	hash, err := hashLockBody(lock)
	if err != nil {
		return nil, fmt.Errorf("lockfile: hashing body: %w", err)
	}
	lock.Hash = hash
	return lock, nil
}

// hashFileList combines per-file hashes into the source component's hash:
// one "relpath:hexhash" line per file (files is already sorted by path by
// the caller), joined with "\n" and hashed — deterministic regardless of
// filesystem walk order or OS path separator.
func hashFileList(files []FileHash) string {
	var b strings.Builder
	for _, f := range files {
		b.WriteString(f.Path)
		b.WriteByte(':')
		b.WriteString(f.Hash)
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// hashLockBody hashes the lock's full JSON body with Hash cleared, so the
// top-level hash covers every included component's data — changing any
// component changes this hash. encoding/json preserves Go struct field
// order, so this is deterministic across runs and platforms.
func hashLockBody(lock *Lock) (string, error) {
	body := *lock
	body.Hash = ""
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Write serializes lock as indented JSON to dir's mod.lock path.
func Write(dir string, lock *Lock) error {
	path, err := Path(dir)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("lockfile: marshaling: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("lockfile: creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("lockfile: writing %s: %w", path, err)
	}
	return nil
}

// Read loads dir's committed mod.lock.
func Read(dir string) (*Lock, error) {
	path, err := Path(dir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lockfile: reading %s: %w", path, err)
	}
	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("lockfile: parsing %s: %w", path, err)
	}
	return &lock, nil
}

// Diff reports every component whose hash/inclusion differs between want
// (the committed lock) and got (freshly generated) as a human-readable
// finding. An empty result means the lock is up to date.
func Diff(want, got *Lock) []string {
	var findings []string
	check := func(name string, w, g Component) {
		if w.Included != g.Included {
			findings = append(findings, fmt.Sprintf("%s: included changed from %v to %v", name, w.Included, g.Included))
			return
		}
		if w.Hash != g.Hash {
			findings = append(findings, fmt.Sprintf("%s: hash changed (%s -> %s)", name, w.Hash, g.Hash))
		}
	}
	check("specs", want.Components.Specs, got.Components.Specs)
	check("config", want.Components.Config, got.Components.Config)
	check("icon", want.Components.Icon, got.Components.Icon)

	if want.Components.Source.Hash != got.Components.Source.Hash {
		findings = append(findings, diffSourceFiles(want.Components.Source, got.Components.Source)...)
	}
	if want.Hash != got.Hash {
		findings = append(findings, "combined hash mismatch")
	}
	return findings
}

func diffSourceFiles(want, got SourceComponent) []string {
	wantByPath := make(map[string]string, len(want.Files))
	for _, f := range want.Files {
		wantByPath[f.Path] = f.Hash
	}
	gotByPath := make(map[string]string, len(got.Files))
	for _, f := range got.Files {
		gotByPath[f.Path] = f.Hash
	}

	var findings []string
	for path, hash := range wantByPath {
		gotHash, ok := gotByPath[path]
		if !ok {
			findings = append(findings, fmt.Sprintf("source: %s was removed", path))
			continue
		}
		if gotHash != hash {
			findings = append(findings, fmt.Sprintf("source: %s changed", path))
		}
	}
	for path := range gotByPath {
		if _, ok := wantByPath[path]; !ok {
			findings = append(findings, fmt.Sprintf("source: %s was added", path))
		}
	}
	sort.Strings(findings)
	return findings
}
