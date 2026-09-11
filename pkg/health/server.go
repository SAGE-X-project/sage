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
	"log/slog"
	"net/http"
	"time"
)

// Logger is the logging surface this package needs. *slog.Logger satisfies
// it; any logger with the same two methods can be passed instead.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Server represents the health check HTTP server
type Server struct {
	checker        *Checker
	logger         Logger
	port           int
	server         *http.Server
	metricsHandler http.Handler
}

// NewServer creates a new health check server. A nil logger selects
// slog.Default(). The /metrics endpoint answers 404 until SetMetricsHandler
// installs a handler (for example metrics.Handler() from pkg/telemetry/metrics).
func NewServer(checker *Checker, logger Logger, port int) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		checker: checker,
		logger:  logger,
		port:    port,
	}
}

// SetMetricsHandler installs the handler served at /metrics.
func (s *Server) SetMetricsHandler(h http.Handler) {
	s.metricsHandler = h
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

	s.logger.Info("starting health check server", "port", s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("health check server error", "error", err)
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
	switch status.Status {
	case StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	case StatusDegraded:
		w.WriteHeader(http.StatusOK) // 200 but with degraded status in body
	default:
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

// handleMetrics serves the installed metrics handler, or 404 when none is set.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.metricsHandler == nil {
		http.Error(w, "metrics handler not configured", http.StatusNotFound)
		return
	}
	s.metricsHandler.ServeHTTP(w, r)
}

// StartHealthServer is a convenience function to start a health server
func StartHealthServer(port int, rpcURL string) (*Server, error) {
	// Create health checker
	checker := NewChecker(rpcURL)

	// Create and start server with the default logger and no metrics endpoint
	server := NewServer(checker, nil, port)
	if err := server.Start(); err != nil {
		return nil, err
	}

	return server, nil
}
