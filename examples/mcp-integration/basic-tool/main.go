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

	fmt.Println(" SAGE-secured MCP Calculator Tool Server")
	fmt.Println("📍 Listening on http://localhost:8080")
	fmt.Println("")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /tools                    - List available tools")
	fmt.Println("  POST /tools/calculator/execute - Execute calculation (requires SAGE signature)")
	fmt.Println("  GET  /health                   - Health check")
	fmt.Println("")
	fmt.Println("Security features:")
	fmt.Println("   All requests must be signed with SAGE")
	fmt.Println("   Agent identity verified via blockchain DID")
	fmt.Println("   Agent capabilities checked before execution")
	fmt.Println("   Responses are signed for authenticity")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
