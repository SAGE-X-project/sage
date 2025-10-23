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

	// 출력
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  SAGE 블록체인 연결 확인")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("네트워크:    %s\n", network)
	fmt.Printf("RPC URL:    %s\n", rpcURL)
	fmt.Println()

	if blockchainStatus.Connected {
		fmt.Println("✓ 상태:      연결됨 (CONNECTED)")
		fmt.Printf("  Chain ID:   %s\n", blockchainStatus.ChainID)
		fmt.Printf("  Block:      %d\n", blockchainStatus.BlockNumber)
		fmt.Printf("  지연시간:    %s\n", blockchainStatus.Latency)

		statusColor := getStatusSymbol(blockchainStatus.Status)
		fmt.Printf("\n%s 전체 상태:  %s\n", statusColor, blockchainStatus.Status)
	} else {
		fmt.Println("✗ 상태:      연결 끊김 (DISCONNECTED)")
		fmt.Printf("  에러:      %s\n", blockchainStatus.Error)
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

	// 출력
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  SAGE 시스템 리소스 확인")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("메모리:       %d MB / %d MB (%.1f%%)\n",
		systemStatus.MemoryUsedMB, systemStatus.MemoryTotalMB, systemStatus.MemoryPercent)
	fmt.Printf("디스크:       %d GB / %d GB (%.1f%%)\n",
		systemStatus.DiskUsedGB, systemStatus.DiskTotalGB, systemStatus.DiskPercent)
	fmt.Printf("Goroutines:  %d\n", systemStatus.GoRoutines)

	statusColor := getStatusSymbol(systemStatus.Status)
	fmt.Printf("\n%s 전체 상태:  %s\n", statusColor, systemStatus.Status)

	if systemStatus.Error != "" {
		fmt.Printf("  경고:       %s\n", systemStatus.Error)
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
	fmt.Println("  SAGE 헬스체크")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("네트워크:     %s\n", network)
	fmt.Printf("RPC URL:     %s\n", rpcURL)
	fmt.Printf("타임스탬프:   %s\n", status.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// 블록체인 상태
	if status.BlockchainStatus != nil {
		fmt.Println("블록체인:")
		if status.BlockchainStatus.Connected {
			fmt.Printf("  ✓ 연결됨   Chain ID: %s, Block: %d\n",
				status.BlockchainStatus.ChainID, status.BlockchainStatus.BlockNumber)
			fmt.Printf("    지연시간:    %s\n", status.BlockchainStatus.Latency)
		} else {
			fmt.Printf("  ✗ 연결 끊김 (Disconnected)\n")
			fmt.Printf("    에러:      %s\n", status.BlockchainStatus.Error)
		}
		fmt.Println()
	}

	// 시스템 상태
	if status.SystemStatus != nil {
		fmt.Println("시스템:")
		fmt.Printf("  메모리:       %d MB / %d MB (%.1f%%)\n",
			status.SystemStatus.MemoryUsedMB, status.SystemStatus.MemoryTotalMB,
			status.SystemStatus.MemoryPercent)
		fmt.Printf("  디스크:       %d GB / %d GB (%.1f%%)\n",
			status.SystemStatus.DiskUsedGB, status.SystemStatus.DiskTotalGB,
			status.SystemStatus.DiskPercent)
		fmt.Printf("  Goroutines:  %d\n", status.SystemStatus.GoRoutines)
		fmt.Println()
	}

	// 전체 상태
	statusSymbol := getStatusSymbol(status.Status)
	fmt.Printf("%s 전체 상태: %s\n", statusSymbol, status.Status)

	if len(status.Errors) > 0 {
		fmt.Println("\n에러 목록:")
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
