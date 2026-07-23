package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"barrakudaModKit/internal/signing"

	"github.com/spf13/cobra"
)

var keygenOutDir string

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate an Ed25519 publisher key pair for signing mod releases",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		pub, priv, err := signing.GenerateKey()
		if err != nil {
			fmt.Printf("failed to generate key pair: %v\n", err)
			os.Exit(1)
		}

		if err := os.MkdirAll(keygenOutDir, 0o755); err != nil {
			fmt.Printf("failed to create %s: %v\n", keygenOutDir, err)
			os.Exit(1)
		}

		privPath := filepath.Join(keygenOutDir, "publisher.key")
		pubPath := filepath.Join(keygenOutDir, "publisher.pub")

		if err := os.WriteFile(privPath, []byte(signing.EncodePrivateKey(priv)), 0o600); err != nil {
			fmt.Printf("failed to write %s: %v\n", privPath, err)
			os.Exit(1)
		}
		if err := os.WriteFile(pubPath, []byte(signing.EncodePublicKey(pub)), 0o644); err != nil {
			fmt.Printf("failed to write %s: %v\n", pubPath, err)
			os.Exit(1)
		}

		fmt.Printf("wrote %s (keep this secret, never commit it)\n", privPath)
		fmt.Printf("wrote %s\n", pubPath)
		fmt.Printf("public key (send this to be registered as a known publisher):\n%s\n", signing.EncodePublicKey(pub))
	},
}

func init() {
	keygenCmd.Flags().StringVar(&keygenOutDir, "out", ".", "directory to write publisher.key/publisher.pub into")
	rootCmd.AddCommand(keygenCmd)
}
