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
	checker *HealthChecker
	logger  logger.Logger
	port    int
	server  *http.Server
}

// NewServer creates a new health check server
func NewServer(checker *HealthChecker, logger logger.Logger, port int) *Server {
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
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
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
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	results := s.checker.CheckAll(ctx)

	// Determine overall status
	overallHealthy := true
	var errors []string

	for name, result := range results {
		if result.Status != "healthy" {
			overallHealthy = false
			if result.Message != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", name, result.Message))
			} else {
				errors = append(errors, fmt.Sprintf("%s: unhealthy", name))
			}
		}
	}

	// Prepare response
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    results,
	}

	if !overallHealthy {
		response["status"] = "unhealthy"
		response["errors"] = errors
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
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
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check only critical services for readiness
	criticalChecks := []string{"blockchain", "keystore"}
	ready := true
	var errors []string

	// Run critical checks
	allResults := s.checker.CheckAll(ctx)
	for _, checkName := range criticalChecks {
		if result, exists := allResults[checkName]; exists {
			if result.Status != "healthy" {
				ready = false
				if result.Message != "" {
					errors = append(errors, fmt.Sprintf("%s: %s", checkName, result.Message))
				} else {
					errors = append(errors, fmt.Sprintf("%s: unhealthy", checkName))
				}
			}
		}
	}

	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if !ready {
		response["errors"] = errors
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
func StartHealthServer(port int, checks map[string]HealthCheck) (*Server, error) {
	// Create health checker
	checker := NewHealthChecker(5 * time.Second)
	for name, check := range checks {
		checker.RegisterCheck(name, check)
	}

	// Create logger
	log := logger.NewLogger(os.Stdout, logger.InfoLevel)

	// Create and start server
	server := NewServer(checker, log, port)
	if err := server.Start(); err != nil {
		return nil, err
	}

	return server, nil
}
