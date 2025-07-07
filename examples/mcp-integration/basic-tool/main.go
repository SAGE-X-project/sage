// MCP Calculator Tool Server with SAGE Security
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Create calculator tool with SAGE
	tool, err := NewCalculatorTool()
	if err != nil {
		log.Fatalf("Failed to create calculator tool: %v", err)
	}

	// MCP endpoints
	http.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		// List available tools
		tools := []map[string]interface{}{
			tool.GetToolDefinition(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tools": tools,
		})
	})

	http.HandleFunc("/tools/calculator/execute", tool.HandleRequest)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("🔐 SAGE-secured MCP Calculator Tool Server")
	fmt.Println("📍 Listening on http://localhost:8080")
	fmt.Println("")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /tools                    - List available tools")
	fmt.Println("  POST /tools/calculator/execute - Execute calculation (requires SAGE signature)")
	fmt.Println("  GET  /health                   - Health check")
	fmt.Println("")
	fmt.Println("Security features:")
	fmt.Println("  ✅ All requests must be signed with SAGE")
	fmt.Println("  ✅ Agent identity verified via blockchain DID")
	fmt.Println("  ✅ Agent capabilities checked before execution")
	fmt.Println("  ✅ Responses are signed for authenticity")

	log.Fatal(http.ListenAndServe(":8080", nil))
}