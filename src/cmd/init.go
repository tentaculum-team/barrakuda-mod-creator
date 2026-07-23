package cmd

import (
	"fmt"
	"os"

	execute_init "barrakudaModKit/internal/execute/init"
	tui "barrakudaModKit/internal/tui/init"

	bubbleTea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var (
	initName  string
	initType  string
	initOut   string
	initForce bool
)

var initCmd = &cobra.Command{
	Use: "init",
	Run: func(cmd *cobra.Command, args []string) {
		// Any of --name/--type/--out/--force skips the TUI entirely and
		// scaffolds directly — the wizard is only for the fully-interactive
		// path (no flags passed).
		if initName != "" || initType != "" || initOut != "" || initForce {
			runNonInteractiveInit()
			return
		}

		player := bubbleTea.NewProgram(tui.ModTypeInitialModel())
		if _, err := player.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

func runNonInteractiveInit() {
	if initName == "" {
		fmt.Println("--name is required when using non-interactive flags")
		os.Exit(1)
	}

	out := initOut
	if out == "" {
		out = "./" + initName
	}

	var (
		path string
		err  error
	)

	switch initType {
	case "", "mcp-go":
		// mcp-go is the only non-interactive type supported today besides
		// provider-go; other mod types (typescript-mcp, python-mcp,
		// application, theme, skill) remain TUI-only until they get the
		// same treatment.
		path, err = execute_init.CreateGoMcpAt(out, initName, initForce)
	case "provider-go":
		path, err = execute_init.CreateGoProviderAt(out, initName, initForce)
	default:
		fmt.Printf("unsupported --type %q (supported: mcp-go, provider-go)\n", initType)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scaffolded %s at %s\n", initName, path)
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&initName, "name", "", "mod/module name")
	initCmd.Flags().StringVar(&initType, "type", "", "mod type (mcp-go, provider-go)")
	initCmd.Flags().StringVar(&initOut, "out", "", "output directory (default ./<name>)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite a non-empty output directory")
}
