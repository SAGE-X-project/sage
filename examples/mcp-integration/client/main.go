// AI Agent Client with SAGE Security
package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("🤖 AI Agent with SAGE Security")
	fmt.Println("================================")
	
	// Create SAGE client - just one line!
	client, err := NewSAGEClient("did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f7F1a")
	if err != nil {
		log.Fatalf("Failed to create SAGE client: %v", err)
	}

	// Example 1: Addition
	fmt.Println("\n📊 Calling Calculator Tool - Addition")
	result, err := client.CallTool("http://localhost:8080/tools/calculator/execute", map[string]interface{}{
		"tool":      "calculator",
		"operation": "add",
		"arguments": map[string]interface{}{
			"a": 10,
			"b": 20,
		},
	})
	
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Result: %v\n", result)
	}

	// Example 2: Division
	fmt.Println("\n📊 Calling Calculator Tool - Division")
	result, err = client.CallTool("http://localhost:8080/tools/calculator/execute", map[string]interface{}{
		"tool":      "calculator",
		"operation": "divide",
		"arguments": map[string]interface{}{
			"a": 100,
			"b": 5,
		},
	})
	
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Result: %v\n", result)
	}

	// Example 3: Multiplication
	fmt.Println("\n📊 Calling Calculator Tool - Multiplication")
	result, err = client.CallTool("http://localhost:8080/tools/calculator/execute", map[string]interface{}{
		"tool":      "calculator",
		"operation": "multiply",
		"arguments": map[string]interface{}{
			"a": 7,
			"b": 8,
		},
	})
	
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Result: %v\n", result)
	}

	fmt.Println("\n✨ All tool calls were cryptographically signed and verified!")
}