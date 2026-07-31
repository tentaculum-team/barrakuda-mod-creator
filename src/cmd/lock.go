package cmd

import (
	"fmt"
	"os"

	"barrakudaModKit/internal/lockfile"

	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Generate or verify a mod's content-integrity lock file",
}

var lockGenerateCmd = &cobra.Command{
	Use:   "generate <path>",
	Short: "Compute mod.lock (hash of specs.json + config.yaml + icon.svg + source) and write it",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]
		lock, err := lockfile.Generate(dir)
		if err != nil {
			fmt.Printf("failed to generate lock: %v\n", err)
			os.Exit(1)
		}
		if err := lockfile.Write(dir, lock); err != nil {
			fmt.Printf("failed to write lock: %v\n", err)
			os.Exit(1)
		}
		path, _ := lockfile.Path(dir)
		fmt.Printf("wrote %s\n", path)
		fmt.Printf("hash: %s\n", lock.Hash)
	},
}

var lockVerifyCmd = &cobra.Command{
	Use:   "verify <path>",
	Short: "Recompute the mod's lock and fail if it doesn't match the committed mod.lock (CI drift check)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]

		committed, err := lockfile.Read(dir)
		if err != nil {
			fmt.Printf("failed to read committed lock: %v\n", err)
			fmt.Println("run `barrakudaModKit lock generate` first and commit the result")
			os.Exit(1)
		}

		fresh, err := lockfile.Generate(dir)
		if err != nil {
			fmt.Printf("failed to generate fresh lock: %v\n", err)
			os.Exit(1)
		}

		findings := lockfile.Diff(committed, fresh)
		if len(findings) > 0 {
			fmt.Println("mod.lock is out of date:")
			for _, f := range findings {
				fmt.Printf("  - %s\n", f)
			}
			fmt.Println("\nrun `barrakudaModKit lock generate` and commit the updated mod.lock")
			os.Exit(1)
		}

		fmt.Println("mod.lock is up to date")
	},
}

func init() {
	lockCmd.AddCommand(lockGenerateCmd, lockVerifyCmd)
	rootCmd.AddCommand(lockCmd)
}
