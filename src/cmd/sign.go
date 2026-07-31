package cmd

import (
	"fmt"
	"os"

	"barrakudaModKit/internal/signing"

	"github.com/spf13/cobra"
)

var (
	signZipPath     string
	signSpecsPath   string // written by --specs OR its deprecated alias --manifest
	signKeyPath     string
	signPublisherID string
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign a release zip + specs.json with a publisher's Ed25519 key",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if signPublisherID == "" {
			fmt.Println("--publisher-id is required")
			os.Exit(1)
		}
		if signSpecsPath == "" {
			fmt.Println("--specs (or its deprecated alias --manifest) is required")
			os.Exit(1)
		}

		zipBytes, err := os.ReadFile(signZipPath)
		if err != nil {
			fmt.Printf("failed to read %s: %v\n", signZipPath, err)
			os.Exit(1)
		}
		// deliberately config.yaml is never part of this: it's mutable by
		// design (the client overwrites it locally), signing it would be
		// meaningless proof of provenance — only specs.json (identity/
		// capabilities) is covered.
		specsBytes, err := os.ReadFile(signSpecsPath)
		if err != nil {
			fmt.Printf("failed to read %s: %v\n", signSpecsPath, err)
			os.Exit(1)
		}
		keyText, err := os.ReadFile(signKeyPath)
		if err != nil {
			fmt.Printf("failed to read %s: %v\n", signKeyPath, err)
			os.Exit(1)
		}

		priv, err := signing.DecodePrivateKey(string(keyText))
		if err != nil {
			fmt.Printf("invalid private key in %s: %v\n", signKeyPath, err)
			os.Exit(1)
		}

		sig, err := signing.Sign(priv, zipBytes, specsBytes)
		if err != nil {
			fmt.Printf("failed to sign: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("publisher_id: %s\n", signPublisherID)
		fmt.Printf("signature: %s\n", sig)
		fmt.Println("\npaste into the catalog submission's package.<os> block:")
		fmt.Printf("  \"publisher_id\": %q,\n", signPublisherID)
		fmt.Printf("  \"signature\": %q\n", sig)
	},
}

func init() {
	signCmd.Flags().StringVar(&signZipPath, "zip", "", "path to the built release zip (required)")
	signCmd.Flags().StringVar(&signSpecsPath, "specs", "", "path to specs.json (required, unless --manifest is used)")
	signCmd.Flags().StringVar(&signSpecsPath, "manifest", "", "deprecated alias for --specs, kept for backwards compatibility")
	signCmd.Flags().StringVar(&signKeyPath, "key", "", "path to the publisher's private key file, from `keygen` (required)")
	signCmd.Flags().StringVar(&signPublisherID, "publisher-id", "", "publisher id registered with the catalog (required)")
	_ = signCmd.MarkFlagRequired("zip")
	_ = signCmd.MarkFlagRequired("key")
	rootCmd.AddCommand(signCmd)
}
