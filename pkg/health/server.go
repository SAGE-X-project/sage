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

package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sage-x-project/sage/internal/logger"
	"github.com/sage-x-project/sage/internal/metrics"
)

// Server represents the health check HTTP server
type Server struct {
	checker *Checker
	logger  logger.Logger
	port    int
	server  *http.Server
}

// NewServer creates a new health check server
func NewServer(checker *Checker, logger logger.Logger, port int) *Server {
	return &Server{
		checker: checker,
		logger:  logger,
		port:    port,
	}
}

// Start starts the health check server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/live", s.handleLiveness)
	mux.HandleFunc("/health/ready", s.handleReadiness)
	mux.HandleFunc("/metrics", s.handleMetrics)

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	s.logger.Info("Starting health check server")

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Health check server error: " + err.Error())
		}
	}()

	return nil
}

// Stop stops the health check server
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleHealth handles the main health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := s.checker.CheckAll()

	// Set HTTP status code based on health status
	if status.Status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if status.Status == StatusDegraded {
		w.WriteHeader(http.StatusOK) // 200 but with degraded status in body
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// handleLiveness handles the liveness probe endpoint
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - just return OK if the server is running
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// handleReadiness handles the readiness probe endpoint
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	status := s.checker.CheckAll()

	// Check critical component: blockchain must be healthy
	ready := status.BlockchainStatus != nil && status.BlockchainStatus.Connected

	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"blockchain": map[string]interface{}{
			"connected": status.BlockchainStatus != nil && status.BlockchainStatus.Connected,
			"status":    status.BlockchainStatus.Status,
		},
	}

	if !ready {
		response["errors"] = status.Errors
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleMetrics handles the metrics endpoint
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	collector := metrics.GetGlobalCollector()
	snapshot := collector.GetSnapshot()

	// Convert to JSON-friendly format
	response := map[string]interface{}{
		"timestamp": snapshot.Timestamp.UTC().Format(time.RFC3339),
		"uptime":    snapshot.Uptime.String(),
		"counters": map[string]int64{
			"signatures":          snapshot.SignatureCount,
			"verifications":       snapshot.VerificationCount,
			"successful_verifies": snapshot.SuccessfulVerifies,
			"failed_verifies":     snapshot.FailedVerifies,
			"did_resolutions":     snapshot.DIDResolutions,
			"cache_hits":          snapshot.CacheHits,
			"cache_misses":        snapshot.CacheMisses,
			"blockchain_calls":    snapshot.BlockchainCalls,
			"blockchain_errors":   snapshot.BlockchainErrors,
		},
		"timings": map[string]interface{}{
			"avg_signature_time_us":      snapshot.AvgSignatureTime,
			"avg_verification_time_us":   snapshot.AvgVerificationTime,
			"avg_blockchain_time_us":     snapshot.AvgBlockchainTime,
			"avg_did_resolution_time_us": snapshot.AvgDIDResolutionTime,
			"p95_signature_time_us":      snapshot.P95SignatureTime,
			"p95_verification_time_us":   snapshot.P95VerificationTime,
			"p95_blockchain_time_us":     snapshot.P95BlockchainTime,
			"p95_did_resolution_time_us": snapshot.P95DIDResolutionTime,
		},
		"rates": map[string]float64{
			"cache_hit_rate":            snapshot.GetCacheHitRate(),
			"verification_success_rate": snapshot.GetVerificationSuccessRate(),
			"blockchain_error_rate":     snapshot.GetBlockchainErrorRate(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// StartHealthServer is a convenience function to start a health server
func StartHealthServer(port int, rpcURL string) (*Server, error) {
	// Create health checker
	checker := NewChecker(rpcURL)

	// Create logger
	log := logger.NewLogger(os.Stdout, logger.InfoLevel)

	// Create and start server
	server := NewServer(checker, log, port)
	if err := server.Start(); err != nil {
		return nil, err
	}

	return server, nil
}
