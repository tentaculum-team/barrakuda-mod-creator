package execute_init

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoScaffoldHasNoSDKDependency proves the Go MCP scaffold no longer pins
// any external module: framework-free skeletons are the whole point of this
// scaffold, matching the TS/Python scaffolds which were already SDK-free.
func TestGoScaffoldHasNoSDKDependency(t *testing.T) {
	root := filepath.Join(t.TempDir(), "gomcptest")
	if _, err := CreateGoMcpAt(root, "gomcptest", false); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if strings.Contains(string(goMod), "require") {
		t.Fatalf("go.mod declares a dependency, expected none:\n%s", goMod)
	}
	if strings.Contains(string(goMod), "mark3labs") {
		t.Fatalf("go.mod still references the mcp-go SDK:\n%s", goMod)
	}

	if _, err := os.Stat(filepath.Join(root, "go.sum")); !os.IsNotExist(err) {
		t.Fatalf("expected no go.sum for a zero-dependency module, stat returned: %v", err)
	}

	server, err := os.ReadFile(filepath.Join(root, "internal", "mcp", "server.go"))
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	if strings.Contains(string(server), "mark3labs") {
		t.Fatalf("server.go still imports the mcp-go SDK:\n%s", server)
	}
}

// TestGoScaffoldBuildsOffline generates a project and runs `go build` with
// GOPROXY=off, proving the zero-dependency claim actually holds — not just
// that go.mod happens to omit a require line.
func TestGoScaffoldBuildsOffline(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}

	root := filepath.Join(t.TempDir(), "gomcptest")
	if _, err := CreateGoMcpAt(root, "gomcptest", false); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	cmd := exec.Command("go", "build", "-buildvcs=false", "./...")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed with GOPROXY=off:\n%s", out)
	}
}
