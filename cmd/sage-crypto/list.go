package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/sage-x-project/sage/crypto/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List keys in storage",
	Long:  `List all keys stored in the specified storage directory.`,
	Example: `  # List all keys in storage
  sage-crypto list --storage-dir ./keys`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&storageDir, "storage-dir", "s", "", "Storage directory (required)")
	listCmd.MarkFlagRequired("storage-dir")
}

func runList(cmd *cobra.Command, args []string) error {
	// Create storage
	keyStorage, err := storage.NewFileKeyStorage(storageDir)
	if err != nil {
		return fmt.Errorf("failed to create key storage: %w", err)
	}

	// List all keys
	keyIDs, err := keyStorage.List()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keyIDs) == 0 {
		fmt.Println("No keys found in storage")
		return nil
	}

	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "KEY ID\tTYPE\tFINGERPRINT\n")
	fmt.Fprintf(w, "------\t----\t-----------\n")

	// Load each key to get details
	for _, id := range keyIDs {
		keyPair, err := keyStorage.Load(id)
		if err != nil {
			fmt.Fprintf(w, "%s\t<error>\t%v\n", id, err)
			continue
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\n", id, keyPair.Type(), keyPair.ID())
	}

	w.Flush()
	
	fmt.Printf("\nTotal keys: %d\n", len(keyIDs))
	fmt.Printf("Storage location: %s\n", storageDir)

	return nil
}