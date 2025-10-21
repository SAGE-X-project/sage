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
	"testing"
)

func TestChecker_CheckBlockchain(t *testing.T) {
	t.Run("InvalidRPC", func(t *testing.T) {
		health := CheckBlockchain("http://invalid:9999")
		if health.Connected {
			t.Error("Expected connection to fail with invalid RPC")
		}
		if health.Status == StatusHealthy {
			t.Error("Expected unhealthy status with invalid RPC")
		}
		if health.Error == "" {
			t.Error("Expected error message")
		}
	})

	t.Run("EmptyRPC", func(t *testing.T) {
		health := CheckBlockchain("")
		if health.Connected {
			t.Error("Expected connection to fail with empty RPC")
		}
		if health.Error != "RPC URL not configured" {
			t.Errorf("Expected 'RPC URL not configured', got: %s", health.Error)
		}
	})

	// Note: To test with real blockchain, set SAGE_TEST_RPC environment variable
	// t.Run("ValidRPC", func(t *testing.T) {
	//     rpc := os.Getenv("SAGE_TEST_RPC")
	//     if rpc == "" {
	//         t.Skip("SAGE_TEST_RPC not set, skipping live test")
	//     }
	//     health := CheckBlockchain(rpc)
	//     if !health.Connected {
	//         t.Errorf("Expected connection to succeed: %s", health.Error)
	//     }
	//     if health.ChainID == "" {
	//         t.Error("Expected chain ID to be set")
	//     }
	// })
}

func TestChecker_CheckSystem(t *testing.T) {
	health := CheckSystem()

	// Memory stats should be set (even if small values)
	if health.MemoryTotalMB == 0 {
		t.Error("Expected total memory > 0")
	}

	// Memory percent should be valid range
	if health.MemoryPercent < 0 || health.MemoryPercent > 100 {
		t.Errorf("Invalid memory percent: %.2f", health.MemoryPercent)
	}

	// Should have at least 1 goroutine (this test)
	if health.GoRoutines <= 0 {
		t.Error("Expected goroutines > 0")
	}

	// Status should be set
	if health.Status == "" {
		t.Error("Expected status to be set")
	}

	// Disk stats might fail in some environments, but shouldn't crash
	if health.Error != "" {
		t.Logf("Disk stats warning: %s", health.Error)
	}

	t.Logf("System health: Memory=%dMB/%dMB (%.1f%%), Goroutines=%d, Disk=%dGB/%dGB (%.1f%%)",
		health.MemoryUsedMB, health.MemoryTotalMB, health.MemoryPercent,
		health.GoRoutines,
		health.DiskUsedGB, health.DiskTotalGB, health.DiskPercent)
}

func TestChecker_CheckAll(t *testing.T) {
	checker := NewChecker("http://invalid:9999")
	status := checker.CheckAll()

	if status.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if status.BlockchainStatus == nil {
		t.Error("Expected blockchain status to be checked")
	}

	if status.SystemStatus == nil {
		t.Error("Expected system status to be checked")
	}

	// With invalid RPC, overall status should be degraded/unhealthy
	if status.Status == StatusHealthy {
		t.Error("Expected unhealthy status with invalid RPC")
	}

	if len(status.Errors) == 0 {
		t.Error("Expected at least one error with invalid RPC")
	}
}
