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

// SAGE Metrics Demo
// This program demonstrates the metrics collection and HTTP export.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sage-x-project/sage/internal/metrics"
	"github.com/sage-x-project/sage/pkg/agent/session"
)

func main() {
	fmt.Println("🚀 SAGE Metrics Demo Server")
	fmt.Println("==============================")
	fmt.Println()

	// Start metrics HTTP server
	metricsAddr := ":9090"
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())

	server := &http.Server{
		Addr:              metricsAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		fmt.Printf("📊 Metrics server listening on http://localhost%s/metrics\n", metricsAddr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Metrics server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	fmt.Println()

	// Simulate some activity to generate metrics
	fmt.Println("📈 Generating sample metrics...")
	fmt.Println()

	simulateActivity()

	// Print access instructions
	fmt.Println()
	fmt.Println("✅ Demo running! Access metrics at:")
	fmt.Printf("   http://localhost%s/metrics\n", metricsAddr)
	fmt.Println()
	fmt.Println("📋 Sample queries:")
	fmt.Printf("   curl localhost%s/metrics | grep sage_handshakes\n", metricsAddr)
	fmt.Printf("   curl localhost%s/metrics | grep sage_sessions\n", metricsAddr)
	fmt.Printf("   curl localhost%s/metrics | grep sage_crypto\n", metricsAddr)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop...")
	fmt.Println()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	fmt.Println("👋 Goodbye!")
}

func simulateActivity() {
	// Simulate handshake metrics
	fmt.Println("  🤝 Simulating handshakes...")
	for i := 0; i < 5; i++ {
		metrics.HandshakesInitiated.WithLabelValues("client").Inc()
		metrics.HandshakeDuration.WithLabelValues("invitation").Observe(0.1)
		metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	}

	// Simulate 2 failed handshakes
	metrics.HandshakesInitiated.WithLabelValues("server").Inc()
	metrics.HandshakesFailed.WithLabelValues("signature_error").Inc()

	metrics.HandshakesInitiated.WithLabelValues("server").Inc()
	metrics.HandshakesFailed.WithLabelValues("timeout").Inc()

	// Simulate session creation
	fmt.Println("  💼 Creating test sessions...")
	mgr := session.NewManager()
	defer func() { _ = mgr.Close() }()

	for i := 0; i < 3; i++ {
		sessionID := fmt.Sprintf("test-session-%d", i)
		sharedSecret := []byte(fmt.Sprintf("secret-%d", i))

		sess, err := mgr.CreateSession(sessionID, sharedSecret)
		if err != nil {
			fmt.Printf("    ⚠️  Failed to create session %d: %v\n", i, err)
			continue
		}

		// Simulate encryption/decryption
		plaintext := []byte(fmt.Sprintf("Hello from session %d", i))

		// Encrypt
		ciphertext, err := sess.Encrypt(plaintext)
		if err != nil {
			fmt.Printf("    ⚠️  Encryption failed: %v\n", err)
			continue
		}

		// Decrypt
		_, err = sess.Decrypt(ciphertext)
		if err != nil {
			fmt.Printf("    ⚠️  Decryption failed: %v\n", err)
		}
	}

	fmt.Println("  ✅ Sample metrics generated!")
	fmt.Println()
	fmt.Println("📊 Current metrics summary:")
	fmt.Println("   - Handshakes initiated: 7 (5 client, 2 server)")
	fmt.Println("   - Handshakes completed: 5")
	fmt.Println("   - Handshakes failed: 2")
	fmt.Println("   - Sessions created: 3")
	fmt.Println("   - Sessions active: 3")
	fmt.Println("   - Crypto operations: 6 (3 encrypt, 3 decrypt)")
}
