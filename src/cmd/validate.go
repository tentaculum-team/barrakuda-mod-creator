package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"barrakudaModKit/internal/manifest"
	"barrakudaModKit/internal/mcpclient"

	"github.com/spf13/cobra"
)

var validateLiveBin string

var validateCmd = &cobra.Command{
	Use:   "validate <path>",
	Short: "Validate a mod's manifest.json before zipping/publishing it",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			path = filepath.Join(path, "manifest.json")
		}

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("failed to read %s: %v\n", path, err)
			os.Exit(1)
		}

		var m manifest.Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			fmt.Printf("invalid JSON in %s: %v\n", path, err)
			os.Exit(1)
		}

		warnings, err := m.Validate()
		if err != nil {
			fmt.Printf("%s is invalid: %v\n", path, err)
			os.Exit(1)
		}
		for _, w := range warnings {
			fmt.Printf("warning: %s\n", w)
		}

		if validateLiveBin != "" {
			if m.Type != manifest.TypeMCP {
				fmt.Printf("--live only applies to type mcp manifests, got %q\n", m.Type)
				os.Exit(1)
			}

			liveTools, err := mcpclient.ListTools(validateLiveBin)
			if err != nil {
				fmt.Printf("--live: failed to introspect %s: %v\n", validateLiveBin, err)
				os.Exit(1)
			}

			findings := diffLiveTools(toolNames(m.Tools), toolSummaryNames(liveTools))
			if len(findings) > 0 {
				for _, f := range findings {
					fmt.Printf("drift: %s\n", f)
				}
				os.Exit(1)
			}
			fmt.Println("--live: manifest tools match what the running server reports")
		}

		fmt.Printf("%s is a valid %s manifest\n", path, m.Type)
	},
}

func toolNames(tools []manifest.Tool) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}

func toolSummaryNames(tools []mcpclient.ToolSummary) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}

// diffLiveTools compares a manifest's declared tool names against what a
// running server actually reports via tools/list, returning one finding per
// name present on only one side. Order of findings follows declared first,
// then live-only names, both in their original order.
func diffLiveTools(declared, live []string) []string {
	declaredSet := map[string]bool{}
	for _, n := range declared {
		declaredSet[n] = true
	}
	liveSet := map[string]bool{}
	for _, n := range live {
		liveSet[n] = true
	}

	var findings []string
	for _, n := range declared {
		if !liveSet[n] {
			findings = append(findings, fmt.Sprintf("tool %q is declared in the manifest but never reported by the running server", n))
		}
	}
	for _, n := range live {
		if !declaredSet[n] {
			findings = append(findings, fmt.Sprintf("tool %q is reported by the running server but not declared in the manifest", n))
		}
	}
	return findings
}

func init() {
	validateCmd.Flags().StringVar(&validateLiveBin, "live", "", "path to a built mcp entry binary to introspect via tools/list and diff against the manifest")
	rootCmd.AddCommand(validateCmd)
}
