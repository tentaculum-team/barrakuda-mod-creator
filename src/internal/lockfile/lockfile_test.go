package lockfile

import (
	"os"
	"path/filepath"
	"testing"
)

// newMod writes a minimal .barrakuda-layout mod under a fresh temp dir and
// returns its root.
func newMod(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".barrakuda", "specs.json"), `{"name":"test","version":"0.1.0"}`)
	mustWrite(t, filepath.Join(dir, ".barrakuda", "config.yaml"), "key: value\n")
	mustWrite(t, filepath.Join(dir, "src", "main.go"), "package main\n")
	return dir
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestGenerateDeterministic(t *testing.T) {
	dir := newMod(t)
	a, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	b, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate (2nd): %v", err)
	}
	if a.Hash != b.Hash {
		t.Fatalf("same input produced different hashes: %s vs %s", a.Hash, b.Hash)
	}
}

func TestGenerateRequiresSpecs(t *testing.T) {
	dir := t.TempDir()
	if _, err := Generate(dir); err == nil {
		t.Fatal("Generate: expected error for a dir with no specs.json")
	}
}

func TestConfigAndIconOptional(t *testing.T) {
	dir := newMod(t)
	lock, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !lock.Components.Config.Included {
		t.Error("config.yaml is present but marked not included")
	}
	if lock.Components.Icon.Included {
		t.Error("icon.svg is absent but marked included")
	}
	if lock.Components.Icon.Hash != "" {
		t.Errorf("icon.svg is absent but has a hash: %q", lock.Components.Icon.Hash)
	}
}

func TestAnyComponentChangeBustsHash(t *testing.T) {
	cases := []struct {
		name  string
		mutFn func(dir string)
	}{
		{"specs", func(dir string) {
			mustWrite(t, filepath.Join(dir, ".barrakuda", "specs.json"), `{"name":"test","version":"0.2.0"}`)
		}},
		{"config", func(dir string) {
			mustWrite(t, filepath.Join(dir, ".barrakuda", "config.yaml"), "key: changed\n")
		}},
		{"source", func(dir string) {
			mustWrite(t, filepath.Join(dir, "src", "main.go"), "package main // changed\n")
		}},
		{"icon added", func(dir string) {
			mustWrite(t, filepath.Join(dir, ".barrakuda", "icon.svg"), "<svg/>")
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := newMod(t)
			before, err := Generate(dir)
			if err != nil {
				t.Fatalf("Generate (before): %v", err)
			}
			tc.mutFn(dir)
			after, err := Generate(dir)
			if err != nil {
				t.Fatalf("Generate (after): %v", err)
			}
			if before.Hash == after.Hash {
				t.Fatalf("%s: hash unchanged after mutation", tc.name)
			}
		})
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	dir := newMod(t)
	lock, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := Write(dir, lock); err != nil {
		t.Fatalf("Write: %v", err)
	}
	loaded, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if loaded.Hash != lock.Hash {
		t.Fatalf("round-tripped hash %q != original %q", loaded.Hash, lock.Hash)
	}

	path, err := Path(dir)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock file not written at expected path %s: %v", path, err)
	}
}

func TestVerifyDetectsDrift(t *testing.T) {
	dir := newMod(t)
	committed, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := Write(dir, committed); err != nil {
		t.Fatalf("Write: %v", err)
	}

	mustWrite(t, filepath.Join(dir, "src", "main.go"), "package main // drifted\n")

	fresh, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate (fresh): %v", err)
	}
	committed, err = Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	findings := Diff(committed, fresh)
	if len(findings) == 0 {
		t.Fatal("Diff: expected findings after drifting src/main.go, got none")
	}
	found := false
	for _, f := range findings {
		if f == "source: src/main.go changed" {
			found = true
		}
	}
	if !found {
		t.Errorf("Diff: expected a finding naming src/main.go, got %v", findings)
	}
}

func TestLockIgnore(t *testing.T) {
	dir := newMod(t)
	mustWrite(t, filepath.Join(dir, "vendored.bin"), "binary-ish content")
	mustWrite(t, filepath.Join(dir, ".barrakuda", "lockignore"), "vendored.bin\n")

	lock, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, f := range lock.Components.Source.Files {
		if f.Path == "vendored.bin" {
			t.Fatal("vendored.bin matched by lockignore should be excluded from source files")
		}
	}
}

func TestPathNormalizationCrossPlatform(t *testing.T) {
	dir := newMod(t)
	mustWrite(t, filepath.Join(dir, "nested", "deep", "file.txt"), "content")

	lock, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	found := false
	for _, f := range lock.Components.Source.Files {
		if f.Path == "nested/deep/file.txt" {
			found = true
		}
		if filepath.Separator != '/' && containsBackslash(f.Path) {
			t.Errorf("path %q contains a backslash, expected forward-slash-normalized", f.Path)
		}
	}
	if !found {
		t.Errorf("expected forward-slash path nested/deep/file.txt in %v", lock.Components.Source.Files)
	}
}

func containsBackslash(s string) bool {
	for _, c := range s {
		if c == '\\' {
			return true
		}
	}
	return false
}
