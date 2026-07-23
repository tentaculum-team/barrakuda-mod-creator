package execute_init

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGoProviderScaffoldBuildsOffline generates a project and runs `go build`
// with GOPROXY=off, proving the zero-dependency claim actually holds — same
// check the MCP Go scaffold gets in mcp_go_no_sdk_test.go.
func TestGoProviderScaffoldBuildsOffline(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}

	root := filepath.Join(t.TempDir(), "goprovidertest")
	if _, err := CreateGoProviderAt(root, "goprovidertest", false); err != nil {
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
