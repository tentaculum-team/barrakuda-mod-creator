package cmd

import (
	"fmt"
	"os"

	"barrakudaModKit/internal/signing"

	"github.com/spf13/cobra"
)

var (
	signZipPath      string
	signManifestPath string
	signKeyPath      string
	signPublisherID  string
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign a release zip + manifest.json with a publisher's Ed25519 key",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if signPublisherID == "" {
			fmt.Println("--publisher-id is required")
			os.Exit(1)
		}

		zipBytes, err := os.ReadFile(signZipPath)
		if err != nil {
			fmt.Printf("failed to read %s: %v\n", signZipPath, err)
			os.Exit(1)
		}
		manifestBytes, err := os.ReadFile(signManifestPath)
		if err != nil {
			fmt.Printf("failed to read %s: %v\n", signManifestPath, err)
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

		sig, err := signing.Sign(priv, zipBytes, manifestBytes)
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
	signCmd.Flags().StringVar(&signManifestPath, "manifest", "", "path to manifest.json (required)")
	signCmd.Flags().StringVar(&signKeyPath, "key", "", "path to the publisher's private key file, from `keygen` (required)")
	signCmd.Flags().StringVar(&signPublisherID, "publisher-id", "", "publisher id registered with the catalog (required)")
	_ = signCmd.MarkFlagRequired("zip")
	_ = signCmd.MarkFlagRequired("manifest")
	_ = signCmd.MarkFlagRequired("key")
	rootCmd.AddCommand(signCmd)
}
