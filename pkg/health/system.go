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
	"fmt"
	"runtime"
	"syscall"
)

const (
	// Thresholds for system health
	MemoryThresholdHealthy  = 70.0  // 70%
	MemoryThresholdDegraded = 85.0  // 85%
	DiskThresholdHealthy    = 70.0  // 70%
	DiskThresholdDegraded   = 85.0  // 85%
)

// CheckSystem checks the health of system resources
func CheckSystem() *SystemHealth {
	health := &SystemHealth{
		Status: StatusHealthy,
	}

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	health.MemoryUsedMB = m.Alloc / 1024 / 1024
	health.MemoryTotalMB = m.Sys / 1024 / 1024

	if health.MemoryTotalMB > 0 {
		health.MemoryPercent = float64(health.MemoryUsedMB) / float64(health.MemoryTotalMB) * 100
	}

	// Get number of goroutines
	health.GoRoutines = runtime.NumGoroutine()

	// Get disk stats (current working directory)
	var stat syscall.Statfs_t
	err := syscall.Statfs(".", &stat)
	if err == nil {
		// Calculate disk usage
		totalBytes := stat.Blocks * uint64(stat.Bsize)
		freeBytes := stat.Bfree * uint64(stat.Bsize)
		usedBytes := totalBytes - freeBytes

		health.DiskTotalGB = totalBytes / 1024 / 1024 / 1024
		health.DiskUsedGB = usedBytes / 1024 / 1024 / 1024

		if health.DiskTotalGB > 0 {
			health.DiskPercent = float64(health.DiskUsedGB) / float64(health.DiskTotalGB) * 100
		}
	} else {
		health.Error = fmt.Sprintf("Failed to get disk stats: %v", err)
	}

	// Determine overall status
	if health.MemoryPercent >= MemoryThresholdDegraded || health.DiskPercent >= DiskThresholdDegraded {
		health.Status = StatusUnhealthy
	} else if health.MemoryPercent >= MemoryThresholdHealthy || health.DiskPercent >= DiskThresholdHealthy {
		health.Status = StatusDegraded
	}

	return health
}
