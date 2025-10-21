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
	"os"

	"github.com/sage-x-project/sage/deployments/config"
	"github.com/sage-x-project/sage/pkg/health"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "health":
		runHealthCheck()
	case "blockchain":
		runBlockchainCheck()
	case "system":
		runSystemCheck()
	case "version", "--version", "-v":
		fmt.Printf("sage-verify version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("SAGE System Verification Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sage-verify <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  health      - Run all health checks (blockchain + system)")
	fmt.Println("  blockchain  - Check blockchain connection status")
	fmt.Println("  system      - Check system resources (memory, CPU, disk)")
	fmt.Println("  version     - Show version information")
	fmt.Println("  help        - Show this help message")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --json      - Output results in JSON format")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  sage-verify health")
	fmt.Println("  sage-verify blockchain --json")
	fmt.Println("  sage-verify system")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  SAGE_NETWORK   - Network to connect to (default: local)")
	fmt.Println("  SAGE_RPC_URL   - Override blockchain RPC URL")
}

func runHealthCheck() {
	jsonOutput := hasJSONFlag()

	// Load network configuration
	network := os.Getenv("SAGE_NETWORK")
	if network == "" {
		network = "local"
	}

	rpcURL := os.Getenv("SAGE_RPC_URL")
	if rpcURL == "" {
		// Try to load from config
		cfg, err := config.LoadConfig(network)
		if err == nil && cfg.NetworkRPC != "" {
			rpcURL = cfg.NetworkRPC
		} else {
			rpcURL = "http://localhost:8545" // Default
		}
	}

	// Run health check
	checker := health.NewChecker(rpcURL)
	status := checker.CheckAll()

	if jsonOutput {
		outputJSON(status)
		if status.Status != health.StatusHealthy {
			os.Exit(1)
		}
		return
	}

	// Pretty print
	printHealthStatus(status, network, rpcURL)

	if status.Status != health.StatusHealthy {
		os.Exit(1)
	}
}

func runBlockchainCheck() {
	jsonOutput := hasJSONFlag()

	network := os.Getenv("SAGE_NETWORK")
	if network == "" {
		network = "local"
	}

	rpcURL := os.Getenv("SAGE_RPC_URL")
	if rpcURL == "" {
		cfg, err := config.LoadConfig(network)
		if err == nil && cfg.NetworkRPC != "" {
			rpcURL = cfg.NetworkRPC
		} else {
			rpcURL = "http://localhost:8545"
		}
	}

	blockchainStatus := health.CheckBlockchain(rpcURL)

	if jsonOutput {
		outputJSON(blockchainStatus)
		if blockchainStatus.Status != health.StatusHealthy {
			os.Exit(1)
		}
		return
	}

	// Pretty print
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  SAGE Blockchain Connection Check")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Network:    %s\n", network)
	fmt.Printf("RPC URL:    %s\n", rpcURL)
	fmt.Println()

	if blockchainStatus.Connected {
		fmt.Println("✓ Status:     CONNECTED")
		fmt.Printf("  Chain ID:   %s\n", blockchainStatus.ChainID)
		fmt.Printf("  Block:      %d\n", blockchainStatus.BlockNumber)
		fmt.Printf("  Latency:    %s\n", blockchainStatus.Latency)
		
		statusColor := getStatusSymbol(blockchainStatus.Status)
		fmt.Printf("\n%s Overall:    %s\n", statusColor, blockchainStatus.Status)
	} else {
		fmt.Println("✗ Status:     DISCONNECTED")
		fmt.Printf("  Error:      %s\n", blockchainStatus.Error)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	if blockchainStatus.Status != health.StatusHealthy {
		os.Exit(1)
	}
}

func runSystemCheck() {
	jsonOutput := hasJSONFlag()

	systemStatus := health.CheckSystem()

	if jsonOutput {
		outputJSON(systemStatus)
		if systemStatus.Status != health.StatusHealthy {
			os.Exit(1)
		}
		return
	}

	// Pretty print
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  SAGE System Resource Check")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Memory:      %d MB / %d MB (%.1f%%)\n",
		systemStatus.MemoryUsedMB, systemStatus.MemoryTotalMB, systemStatus.MemoryPercent)
	fmt.Printf("Disk:        %d GB / %d GB (%.1f%%)\n",
		systemStatus.DiskUsedGB, systemStatus.DiskTotalGB, systemStatus.DiskPercent)
	fmt.Printf("Goroutines:  %d\n", systemStatus.GoRoutines)
	
	statusColor := getStatusSymbol(systemStatus.Status)
	fmt.Printf("\n%s Overall:    %s\n", statusColor, systemStatus.Status)

	if systemStatus.Error != "" {
		fmt.Printf("  Warning:    %s\n", systemStatus.Error)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	if systemStatus.Status != health.StatusHealthy {
		os.Exit(1)
	}
}

func printHealthStatus(status *health.HealthStatus, network, rpcURL string) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  SAGE Health Check")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Network:     %s\n", network)
	fmt.Printf("RPC URL:     %s\n", rpcURL)
	fmt.Printf("Timestamp:   %s\n", status.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Blockchain status
	if status.BlockchainStatus != nil {
		fmt.Println("Blockchain:")
		if status.BlockchainStatus.Connected {
			fmt.Printf("  ✓ Connected   Chain ID: %s, Block: %d\n",
				status.BlockchainStatus.ChainID, status.BlockchainStatus.BlockNumber)
			fmt.Printf("    Latency:    %s\n", status.BlockchainStatus.Latency)
		} else {
			fmt.Printf("  ✗ Disconnected\n")
			fmt.Printf("    Error:      %s\n", status.BlockchainStatus.Error)
		}
		fmt.Println()
	}

	// System status
	if status.SystemStatus != nil {
		fmt.Println("System:")
		fmt.Printf("  Memory:      %d MB / %d MB (%.1f%%)\n",
			status.SystemStatus.MemoryUsedMB, status.SystemStatus.MemoryTotalMB,
			status.SystemStatus.MemoryPercent)
		fmt.Printf("  Disk:        %d GB / %d GB (%.1f%%)\n",
			status.SystemStatus.DiskUsedGB, status.SystemStatus.DiskTotalGB,
			status.SystemStatus.DiskPercent)
		fmt.Printf("  Goroutines:  %d\n", status.SystemStatus.GoRoutines)
		fmt.Println()
	}

	// Overall status
	statusSymbol := getStatusSymbol(status.Status)
	fmt.Printf("%s Overall Status: %s\n", statusSymbol, status.Status)

	if len(status.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range status.Errors {
			fmt.Printf("  • %s\n", err)
		}
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
}

func getStatusSymbol(status health.Status) string {
	switch status {
	case health.StatusHealthy:
		return "✓"
	case health.StatusDegraded:
		return "⚠"
	case health.StatusUnhealthy:
		return "✗"
	default:
		return "?"
	}
}

func hasJSONFlag() bool {
	for _, arg := range os.Args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func outputJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}
