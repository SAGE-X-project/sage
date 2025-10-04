// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sage-x-project/sage/internal/logger"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Name      string                 `json:"name"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthCheck represents a single health check function
type HealthCheck func(ctx context.Context) error

// HealthChecker manages multiple health checks
type HealthChecker struct {
	checks   map[string]HealthCheck
	timeout  time.Duration
	mu       sync.RWMutex
	logger   logger.Logger
	cacheTTL time.Duration
	cache    map[string]*cachedResult
}

// cachedResult stores a cached health check result
type cachedResult struct {
	result    *CheckResult
	expiresAt time.Time
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(timeout time.Duration) *HealthChecker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &HealthChecker{
		checks:   make(map[string]HealthCheck),
		timeout:  timeout,
		logger:   logger.GetDefaultLogger(),
		cacheTTL: 10 * time.Second,
		cache:    make(map[string]*cachedResult),
	}
}

// SetLogger sets the logger for the health checker
func (h *HealthChecker) SetLogger(l logger.Logger) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.logger = l
}

// SetCacheTTL sets the cache TTL for health check results
func (h *HealthChecker) SetCacheTTL(ttl time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cacheTTL = ttl
}

// RegisterCheck registers a new health check
func (h *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.checks[name] = check
	h.logger.Info("Health check registered", logger.String("name", name))
}

// UnregisterCheck removes a health check
func (h *HealthChecker) UnregisterCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.checks, name)
	delete(h.cache, name)
	h.logger.Info("Health check unregistered", logger.String("name", name))
}

// Check performs a single health check
func (h *HealthChecker) Check(ctx context.Context, name string) (*CheckResult, error) {
	h.mu.RLock()
	check, exists := h.checks[name]
	h.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("health check not found: %s", name)
	}

	// Check cache
	if cached := h.getCachedResult(name); cached != nil {
		return cached, nil
	}

	// Perform the check with timeout
	checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	start := time.Now()
	err := check(checkCtx)
	duration := time.Since(start)

	result := &CheckResult{
		Name:      name,
		Timestamp: time.Now(),
		Duration:  duration,
	}

	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = err.Error()
		h.logger.Warn("Health check failed",
			logger.String("name", name),
			logger.Error(err),
			logger.Duration("duration", duration),
		)
	} else {
		result.Status = StatusHealthy
		h.logger.Debug("Health check passed",
			logger.String("name", name),
			logger.Duration("duration", duration),
		)
	}

	// Cache the result
	h.cacheResult(name, result)

	return result, nil
}

// CheckAll performs all registered health checks
func (h *HealthChecker) CheckAll(ctx context.Context) map[string]*CheckResult {
	h.mu.RLock()
	checkNames := make([]string, 0, len(h.checks))
	for name := range h.checks {
		checkNames = append(checkNames, name)
	}
	h.mu.RUnlock()

	results := make(map[string]*CheckResult)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	for _, name := range checkNames {
		wg.Add(1)
		go func(checkName string) {
			defer wg.Done()

			result, err := h.Check(ctx, checkName)
			if err != nil {
				result = &CheckResult{
					Name:      checkName,
					Status:    StatusUnhealthy,
					Message:   fmt.Sprintf("Check failed: %v", err),
					Timestamp: time.Now(),
				}
			}

			resultsMu.Lock()
			results[checkName] = result
			resultsMu.Unlock()
		}(name)
	}

	wg.Wait()
	return results
}

// GetOverallStatus returns the overall health status
func (h *HealthChecker) GetOverallStatus(ctx context.Context) Status {
	results := h.CheckAll(ctx)

	if len(results) == 0 {
		return StatusHealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}

	return StatusHealthy
}

// getCachedResult retrieves a cached result if it's still valid
func (h *HealthChecker) getCachedResult(name string) *CheckResult {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cached, exists := h.cache[name]
	if !exists || time.Now().After(cached.expiresAt) {
		return nil
	}

	return cached.result
}

// cacheResult stores a result in the cache
func (h *HealthChecker) cacheResult(name string, result *CheckResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cache[name] = &cachedResult{
		result:    result,
		expiresAt: time.Now().Add(h.cacheTTL),
	}
}

// ClearCache clears all cached results
func (h *HealthChecker) ClearCache() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cache = make(map[string]*cachedResult)
	h.logger.Debug("Health check cache cleared")
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status    Status                    `json:"status"`
	Timestamp time.Time                 `json:"timestamp"`
	Checks    map[string]*CheckResult   `json:"checks"`
	Details   map[string]interface{}    `json:"details,omitempty"`
}

// GetSystemHealth returns comprehensive system health information
func (h *HealthChecker) GetSystemHealth(ctx context.Context) *SystemHealth {
	checks := h.CheckAll(ctx)
	status := h.GetOverallStatus(ctx)

	return &SystemHealth{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
	}
}

// Common health check implementations

// BlockchainHealthCheck creates a health check for blockchain connectivity
func BlockchainHealthCheck(checker func(context.Context) error) HealthCheck {
	return func(ctx context.Context) error {
		if checker == nil {
			return fmt.Errorf("blockchain checker not configured")
		}
		return checker(ctx)
	}
}

// KeyStoreHealthCheck creates a health check for key store availability
func KeyStoreHealthCheck(checker func() error) HealthCheck {
	return func(ctx context.Context) error {
		if checker == nil {
			return fmt.Errorf("keystore checker not configured")
		}

		// Run checker with context support
		done := make(chan error, 1)
		go func() {
			done <- checker()
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		}
	}
}

// DatabaseHealthCheck creates a health check for database connectivity
func DatabaseHealthCheck(ping func(context.Context) error) HealthCheck {
	return func(ctx context.Context) error {
		if ping == nil {
			return fmt.Errorf("database ping function not configured")
		}
		return ping(ctx)
	}
}

// ServiceHealthCheck creates a health check for external service availability
func ServiceHealthCheck(url string, checker func(context.Context, string) error) HealthCheck {
	return func(ctx context.Context) error {
		if checker == nil {
			return fmt.Errorf("service checker not configured")
		}
		return checker(ctx, url)
	}
}