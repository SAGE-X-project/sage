package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sage-x-project/sage/health"
)

func main() {
	fmt.Println("🏥 Starting SAGE Health Check Server")

	// Define health checks
	checks := map[string]health.HealthCheck{
		"blockchain": func(ctx context.Context) health.HealthResult {
			// Check blockchain connectivity
			client, err := ethclient.DialContext(ctx, "http://localhost:8545")
			if err != nil {
				return health.HealthResult{
					Status: "unhealthy",
					Error:  err,
					Details: map[string]interface{}{
						"rpc_url": "http://localhost:8545",
					},
				}
			}
			defer client.Close()

			// Get block number
			blockNum, err := client.BlockNumber(ctx)
			if err != nil {
				return health.HealthResult{
					Status: "unhealthy",
					Error:  err,
				}
			}

			return health.HealthResult{
				Status: "healthy",
				Details: map[string]interface{}{
					"block_number": blockNum,
					"rpc_url":      "http://localhost:8545",
				},
			}
		},
		"keystore": func(ctx context.Context) health.HealthResult {
			// Check keystore accessibility
			keyDir := ".sage/keys"
			if _, err := os.Stat(keyDir); os.IsNotExist(err) {
				return health.HealthResult{
					Status: "unhealthy",
					Error:  fmt.Errorf("keystore directory not found: %s", keyDir),
				}
			}

			return health.HealthResult{
				Status: "healthy",
				Details: map[string]interface{}{
					"key_directory": keyDir,
				},
			}
		},
		"did_resolver": func(ctx context.Context) health.HealthResult {
			// Placeholder for DID resolver check
			return health.HealthResult{
				Status: "healthy",
				Details: map[string]interface{}{
					"resolver": "ethereum",
					"cache":    "enabled",
				},
			}
		},
	}

	// Start health server
	server, err := health.StartHealthServer(8080, checks)
	if err != nil {
		log.Fatalf("Failed to start health server: %v", err)
	}

	fmt.Println("✅ Health check server started on port 8080")
	fmt.Println("")
	fmt.Println("Available endpoints:")
	fmt.Println("  - http://localhost:8080/health        - Full health check")
	fmt.Println("  - http://localhost:8080/health/live   - Liveness probe")
	fmt.Println("  - http://localhost:8080/health/ready  - Readiness probe")
	fmt.Println("  - http://localhost:8080/metrics       - Metrics")
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Shutting down health server...")
	if err := server.Stop(context.Background()); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	fmt.Println("👋 Goodbye!")
}