package execute_init

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// TestGenerators only checks that the expected files exist and that .go
// files parse (go/parser.ParseFile — pure syntax, no imports resolved, no
// network, no `go mod download`), so it stays runnable offline in CI.
//
// It does NOT catch a wrong import path or a misused mcp-go SDK call — only
// `go build` against the real dependency graph does that. After touching the
// Go MCP scaffold templates (mcp_go.go), also generate a throwaway project
// and run `go build ./...` / `go vet ./...` inside it by hand (or via
// `barrakuda-mod-creator init --name x --out <tmp> --type mcp-go`) whenever
// Go's module proxy is reachable.
func TestGenerators(t *testing.T) {
	base := t.TempDir()

	cases := []struct {
		name  string
		fn    func(string) (string, error)
		files []string
	}{
		// CreateGoMcpAt is exercised directly (not CreateGoMcp) with a
		// separate, valid Go module name ("gomcptest") instead of reusing
		// the temp directory's absolute path as the module name: on
		// Windows that path lives under C:\Users\... and a literal
		// "\U" inside a generated Go import string is parsed as the start
		// of a \Uxxxxxxxx unicode escape, breaking the parse. CreateGoMcp
		// itself (used by the TUI) still conflates dir and module name —
		// harmless there since users type ordinary mod names, not paths.
		{"go", func(root string) (string, error) { return CreateGoMcpAt(root, "gomcptest", false) }, []string{"specs.json", "go.mod", "cmd/api/main.go", "internal/domain/greeting.go", "internal/repository/greeting_repository.go", "internal/service/greeting_service.go", "internal/mcp/server.go", "README.md"}},
		{"ts", CreateTypescriptMcp, []string{"specs.json", "package.json", "tsconfig.json", "src/domain/greeting.ts", "src/repository/greetingRepository.ts", "src/service/greetingService.ts", "src/mcp/server.ts", "src/index.ts", "README.md"}},
		{"py", CreatePythonMcp, []string{"specs.json", "main.py", "app/domain/greeting.py", "app/repository/greeting_repository.py", "app/service/greeting_service.py", "app/mcp/server.py", "README.md"}},
		{"agent", CreateAgent, []string{"specs.json", "docker/Dockerfile", "README.md"}},
		{"agent-provider", func(root string) (string, error) { return CreateAgentProviderAt(root, "agentprovidertest", false) }, []string{"specs.json", "go.mod", "cmd/api/main.go", "internal/agent/server.go", "docker/Dockerfile", "README.md"}},
		{"provider", func(root string) (string, error) { return CreateGoProviderAt(root, "goprovidertest", false) }, []string{"specs.json", "go.mod", "cmd/api/main.go", "internal/provider/server.go", "README.md"}},
		{"theme", CreateTheme, []string{"specs.json"}},
		{"skill", CreateSkill, []string{"specs.json", "skill.md"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := filepath.Join(base, c.name)

			path, err := c.fn(root)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}
			if path != root {
				t.Fatalf("expected path %q, got %q", root, path)
			}

			for _, rel := range c.files {
				full := filepath.Join(root, rel)

				info, err := os.Stat(full)
				if err != nil {
					t.Fatalf("expected file %s: %v", rel, err)
				}
				if info.Size() == 0 {
					t.Fatalf("file %s is empty", rel)
				}

				if filepath.Ext(full) == ".go" {
					if _, err := parser.ParseFile(token.NewFileSet(), full, nil, parser.AllErrors); err != nil {
						t.Fatalf("generated Go file %s does not parse: %v", rel, err)
					}
				}
			}
		})
	}
}
