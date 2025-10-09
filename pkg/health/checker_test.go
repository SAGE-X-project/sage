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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthChecker(t *testing.T) {
	t.Run("RegisterAndCheck", func(t *testing.T) {
		checker := NewHealthChecker(1 * time.Second)

		// Register a healthy check
		checker.RegisterCheck("test_healthy", func(ctx context.Context) error {
			return nil
		})

		// Register an unhealthy check
		checker.RegisterCheck("test_unhealthy", func(ctx context.Context) error {
			return errors.New("service unavailable")
		})

		// Check healthy service
		result, err := checker.Check(context.Background(), "test_healthy")
		require.NoError(t, err)
		assert.Equal(t, StatusHealthy, result.Status)
		assert.Equal(t, "test_healthy", result.Name)
		assert.Empty(t, result.Message)

		// Check unhealthy service
		result, err = checker.Check(context.Background(), "test_unhealthy")
		require.NoError(t, err)
		assert.Equal(t, StatusUnhealthy, result.Status)
		assert.Equal(t, "test_unhealthy", result.Name)
		assert.Equal(t, "service unavailable", result.Message)
	})

	t.Run("CheckNonExistent", func(t *testing.T) {
		checker := NewHealthChecker(1 * time.Second)

		_, err := checker.Check(context.Background(), "non_existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "health check not found")
	})

	t.Run("CheckWithTimeout", func(t *testing.T) {
		checker := NewHealthChecker(100 * time.Millisecond)

		// Register a slow check
		checker.RegisterCheck("slow_check", func(ctx context.Context) error {
			select {
			case <-time.After(200 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

		// Should timeout
		result, err := checker.Check(context.Background(), "slow_check")
		require.NoError(t, err)
		assert.Equal(t, StatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "context deadline exceeded")
	})

	t.Run("CheckAll", func(t *testing.T) {
		checker := NewHealthChecker(1 * time.Second)

		// Register multiple checks
		checker.RegisterCheck("check1", func(ctx context.Context) error {
			return nil
		})
		checker.RegisterCheck("check2", func(ctx context.Context) error {
			return errors.New("failed")
		})
		checker.RegisterCheck("check3", func(ctx context.Context) error {
			return nil
		})

		results := checker.CheckAll(context.Background())

		assert.Len(t, results, 3)
		assert.Equal(t, StatusHealthy, results["check1"].Status)
		assert.Equal(t, StatusUnhealthy, results["check2"].Status)
		assert.Equal(t, StatusHealthy, results["check3"].Status)
	})

	t.Run("GetOverallStatus", func(t *testing.T) {
		checker := NewHealthChecker(1 * time.Second)

		// All healthy
		checker.RegisterCheck("healthy1", func(ctx context.Context) error {
			return nil
		})
		checker.RegisterCheck("healthy2", func(ctx context.Context) error {
			return nil
		})

		status := checker.GetOverallStatus(context.Background())
		assert.Equal(t, StatusHealthy, status)

		// Add unhealthy check
		checker.RegisterCheck("unhealthy", func(ctx context.Context) error {
			return errors.New("error")
		})

		status = checker.GetOverallStatus(context.Background())
		assert.Equal(t, StatusUnhealthy, status)

		// Remove unhealthy check
		checker.UnregisterCheck("unhealthy")

		status = checker.GetOverallStatus(context.Background())
		assert.Equal(t, StatusHealthy, status)
	})

	t.Run("Caching", func(t *testing.T) {
		checker := NewHealthChecker(1 * time.Second)
		checker.SetCacheTTL(100 * time.Millisecond)

		callCount := 0
		checker.RegisterCheck("cached_check", func(ctx context.Context) error {
			callCount++
			return nil
		})

		// First call should execute the check
		result1, err := checker.Check(context.Background(), "cached_check")
		require.NoError(t, err)
		assert.Equal(t, StatusHealthy, result1.Status)
		assert.Equal(t, 1, callCount)

		// Second call should use cache
		result2, err := checker.Check(context.Background(), "cached_check")
		require.NoError(t, err)
		assert.Equal(t, StatusHealthy, result2.Status)
		assert.Equal(t, 1, callCount) // Should not increment

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Third call should execute the check again
		result3, err := checker.Check(context.Background(), "cached_check")
		require.NoError(t, err)
		assert.Equal(t, StatusHealthy, result3.Status)
		assert.Equal(t, 2, callCount) // Should increment
	})

	t.Run("ClearCache", func(t *testing.T) {
		checker := NewHealthChecker(1 * time.Second)
		checker.SetCacheTTL(1 * time.Hour) // Long TTL

		callCount := 0
		checker.RegisterCheck("cached_check", func(ctx context.Context) error {
			callCount++
			return nil
		})

		// First call
		checker.Check(context.Background(), "cached_check")
		assert.Equal(t, 1, callCount)

		// Second call (should use cache)
		checker.Check(context.Background(), "cached_check")
		assert.Equal(t, 1, callCount)

		// Clear cache
		checker.ClearCache()

		// Third call (should execute again)
		checker.Check(context.Background(), "cached_check")
		assert.Equal(t, 2, callCount)
	})

	t.Run("GetSystemHealth", func(t *testing.T) {
		checker := NewHealthChecker(1 * time.Second)

		checker.RegisterCheck("database", func(ctx context.Context) error {
			return nil
		})
		checker.RegisterCheck("blockchain", func(ctx context.Context) error {
			return errors.New("connection failed")
		})

		health := checker.GetSystemHealth(context.Background())

		assert.Equal(t, StatusUnhealthy, health.Status)
		assert.Len(t, health.Checks, 2)
		assert.Equal(t, StatusHealthy, health.Checks["database"].Status)
		assert.Equal(t, StatusUnhealthy, health.Checks["blockchain"].Status)
		assert.NotZero(t, health.Timestamp)
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		checker := NewHealthChecker(1 * time.Second)

		// Register checks concurrently
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				name := "check_" + string(rune('0'+idx))
				checker.RegisterCheck(name, func(ctx context.Context) error {
					return nil
				})
			}(i)
		}
		wg.Wait()

		// Check all concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results := checker.CheckAll(context.Background())
				assert.Len(t, results, 10)
			}()
		}
		wg.Wait()

		// Unregister concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				name := "check_" + string(rune('0'+idx))
				checker.UnregisterCheck(name)
			}(i)
		}
		wg.Wait()

		results := checker.CheckAll(context.Background())
		assert.Len(t, results, 0)
	})
}

func TestCommonHealthChecks(t *testing.T) {
	t.Run("BlockchainHealthCheck", func(t *testing.T) {
		// Test successful check
		check := BlockchainHealthCheck(func(ctx context.Context) error {
			return nil
		})

		err := check(context.Background())
		assert.NoError(t, err)

		// Test failed check
		check = BlockchainHealthCheck(func(ctx context.Context) error {
			return errors.New("blockchain error")
		})

		err = check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blockchain error")

		// Test nil checker
		check = BlockchainHealthCheck(nil)
		err = check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not configured")
	})

	t.Run("KeyStoreHealthCheck", func(t *testing.T) {
		// Test successful check
		check := KeyStoreHealthCheck(func() error {
			return nil
		})

		err := check(context.Background())
		assert.NoError(t, err)

		// Test failed check
		check = KeyStoreHealthCheck(func() error {
			return errors.New("keystore error")
		})

		err = check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "keystore error")

		// Test context cancellation
		check = KeyStoreHealthCheck(func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err = check(ctx)
		assert.Error(t, err)
	})

	t.Run("DatabaseHealthCheck", func(t *testing.T) {
		// Test successful check
		check := DatabaseHealthCheck(func(ctx context.Context) error {
			return nil
		})

		err := check(context.Background())
		assert.NoError(t, err)

		// Test failed check
		check = DatabaseHealthCheck(func(ctx context.Context) error {
			return errors.New("connection refused")
		})

		err = check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("ServiceHealthCheck", func(t *testing.T) {
		// Test successful check
		check := ServiceHealthCheck("https://api.example.com", func(ctx context.Context, url string) error {
			assert.Equal(t, "https://api.example.com", url)
			return nil
		})

		err := check(context.Background())
		assert.NoError(t, err)

		// Test failed check
		check = ServiceHealthCheck("https://api.example.com", func(ctx context.Context, url string) error {
			return errors.New("service unavailable")
		})

		err = check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service unavailable")
	})
}

func BenchmarkHealthChecker(b *testing.B) {
	checker := NewHealthChecker(1 * time.Second)

	// Register some checks
	for i := 0; i < 10; i++ {
		name := "check_" + string(rune('0'+i))
		checker.RegisterCheck(name, func(ctx context.Context) error {
			// Simulate some work
			time.Sleep(1 * time.Microsecond)
			return nil
		})
	}

	b.Run("SingleCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			checker.Check(context.Background(), "check_0")
		}
	})

	b.Run("CheckAll", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			checker.CheckAll(context.Background())
		}
	})

	b.Run("GetOverallStatus", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			checker.GetOverallStatus(context.Background())
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		checker.SetCacheTTL(1 * time.Second)
		for i := 0; i < b.N; i++ {
			checker.Check(context.Background(), "check_0")
		}
	})
}
