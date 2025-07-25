package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	
	// Import chain providers to register them
	_ "github.com/sage-x-project/sage/crypto/chain/ethereum"
	_ "github.com/sage-x-project/sage/crypto/chain/solana"
)

var rootCmd = &cobra.Command{
	Use:   "sage-crypto",
	Short: "SAGE Crypto CLI - Key management and cryptographic operations",
	Long: `SAGE Crypto CLI provides tools for managing cryptographic keys and performing
cryptographic operations for the SAGE (Secure Agent Guarantee Engine) project.

This tool supports:
- Key pair generation (Ed25519, Secp256k1)
- Key export/import (JWK, PEM formats)
- Secure key storage
- Key rotation
- Message signing and verification`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Disable default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	
	// Note: Commands are registered in their respective files
	// - generate.go: generateCmd
	// - sign.go: signCmd
	// - verify.go: verifyCmd
	// - list.go: listCmd
	// - rotate.go: rotateCmd
	// - address.go: addressCmd
}