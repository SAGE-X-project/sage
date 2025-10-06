// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


// AI Agent Client with SAGE Security
package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("AI Agent with SAGE Security")
	fmt.Println("================================")
	
	// Create SAGE client - just one line!
	client, err := NewSAGEClient("did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f7F1a")
	if err != nil {
		log.Fatalf("Failed to create SAGE client: %v", err)
	}

	// Example 1: Addition
	fmt.Println("\nCalling Calculator Tool - Addition")
	result, err := client.CallTool("http://localhost:8080/tools/calculator/execute", map[string]interface{}{
		"tool":      "calculator",
		"operation": "add",
		"arguments": map[string]interface{}{
			"a": 10,
			"b": 20,
		},
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	// Example 2: Division
	fmt.Println("\nCalling Calculator Tool - Division")
	result, err = client.CallTool("http://localhost:8080/tools/calculator/execute", map[string]interface{}{
		"tool":      "calculator",
		"operation": "divide",
		"arguments": map[string]interface{}{
			"a": 100,
			"b": 5,
		},
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	// Example 3: Multiplication
	fmt.Println("\nCalling Calculator Tool - Multiplication")
	result, err = client.CallTool("http://localhost:8080/tools/calculator/execute", map[string]interface{}{
		"tool":      "calculator",
		"operation": "multiply",
		"arguments": map[string]interface{}{
			"a": 7,
			"b": 8,
		},
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	fmt.Println("\nAll tool calls were cryptographically signed and verified!")
}