// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
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
	if err := listCmd.MarkFlagRequired("storage-dir"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}
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
	_, _ = fmt.Fprintf(w, "KEY ID\tTYPE\tFINGERPRINT\n")
	_, _ = fmt.Fprintf(w, "------\t----\t-----------\n")

	// Load each key to get details
	for _, id := range keyIDs {
		keyPair, err := keyStorage.Load(id)
		if err != nil {
			_, _ = fmt.Fprintf(w, "%s\t<error>\t%v\n", id, err)
			continue
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", id, keyPair.Type(), keyPair.ID())
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}

	fmt.Printf("\nTotal keys: %d\n", len(keyIDs))
	fmt.Printf("Storage location: %s\n", storageDir)

	return nil
}
