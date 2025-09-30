package main

import (
	"fmt"

	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/rotation"
	"github.com/sage-x-project/sage/crypto/storage"
	"github.com/spf13/cobra"
)

var (
	keepOldKeys bool
)

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate a key in storage",
	Long: `Rotate a key in storage, generating a new key of the same type.

Key rotation is important for security best practices. This command will:
  1. Load the existing key from storage
  2. Generate a new key of the same type
  3. Store the new key with the same ID
  4. Optionally keep the old key with a special ID`,
	Example: `  # Rotate a key and discard the old one
  sage-crypto rotate --storage-dir ./keys --key-id mykey

  # Rotate a key and keep the old one
  sage-crypto rotate --storage-dir ./keys --key-id mykey --keep-old`,
	RunE: runRotate,
}

func init() {
	rootCmd.AddCommand(rotateCmd)

	rotateCmd.Flags().StringVarP(&storageDir, "storage-dir", "s", "", "Storage directory (required)")
	rotateCmd.Flags().StringVarP(&keyID, "key-id", "k", "", "Key ID to rotate (required)")
	rotateCmd.Flags().BoolVar(&keepOldKeys, "keep-old", false, "Keep old keys after rotation")

	rotateCmd.MarkFlagRequired("storage-dir")
	rotateCmd.MarkFlagRequired("key-id")
}

func runRotate(cmd *cobra.Command, args []string) error {
	// Create storage
	keyStorage, err := storage.NewFileKeyStorage(storageDir)
	if err != nil {
		return fmt.Errorf("failed to create key storage: %w", err)
	}

	// Load existing key to get its info
	oldKeyPair, err := keyStorage.Load(keyID)
	if err != nil {
		return fmt.Errorf("failed to load existing key: %w", err)
	}

	// Create rotator
	rotator := rotation.NewKeyRotator(keyStorage)
	
	// Configure rotation
	rotator.SetRotationConfig(sagecrypto.KeyRotationConfig{
		KeepOldKeys: keepOldKeys,
	})

	// Perform rotation
	newKeyPair, err := rotator.Rotate(keyID)
	if err != nil {
		return fmt.Errorf("failed to rotate key: %w", err)
	}

	fmt.Println(" Key rotation successful!")
	fmt.Printf("\nRotation details:\n")
	fmt.Printf("  Key ID: %s\n", keyID)
	fmt.Printf("  Key Type: %s\n", newKeyPair.Type())
	fmt.Printf("  Old Key Fingerprint: %s\n", oldKeyPair.ID())
	fmt.Printf("  New Key Fingerprint: %s\n", newKeyPair.ID())
	
	if keepOldKeys {
		fmt.Printf("  Old Key Stored As: %s.old.%s\n", keyID, oldKeyPair.ID())
	}

	// Show rotation history
	history, err := rotator.GetRotationHistory(keyID)
	if err == nil && len(history) > 0 {
		fmt.Printf("\nRotation history (%d rotations):\n", len(history))
		for i, event := range history {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(history)-5)
				break
			}
			fmt.Printf("  %s: %s → %s (%s)\n",
				event.Timestamp.Format("2006-01-02 15:04:05"),
				event.OldKeyID[:8],
				event.NewKeyID[:8],
				event.Reason)
		}
	}

	return nil
}