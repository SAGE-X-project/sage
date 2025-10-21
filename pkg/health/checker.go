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

import "time"

// Checker performs health checks
type Checker struct {
	rpcURL string
}

// NewChecker creates a new health checker
func NewChecker(rpcURL string) *Checker {
	return &Checker{
		rpcURL: rpcURL,
	}
}

// CheckAll performs all health checks
func (c *Checker) CheckAll() *HealthStatus {
	status := &HealthStatus{
		Timestamp: time.Now(),
		Status:    StatusHealthy,
		Errors:    make([]string, 0),
	}

	// Check blockchain
	status.BlockchainStatus = CheckBlockchain(c.rpcURL)
	if status.BlockchainStatus.Status != StatusHealthy {
		status.Status = status.BlockchainStatus.Status
		if status.BlockchainStatus.Error != "" {
			status.Errors = append(status.Errors, "Blockchain: "+status.BlockchainStatus.Error)
		}
	}

	// Check system
	status.SystemStatus = CheckSystem()
	if status.SystemStatus.Status != StatusHealthy {
		if status.Status == StatusHealthy {
			status.Status = status.SystemStatus.Status
		} else if status.SystemStatus.Status == StatusUnhealthy {
			status.Status = StatusUnhealthy
		}
		if status.SystemStatus.Error != "" {
			status.Errors = append(status.Errors, "System: "+status.SystemStatus.Error)
		}
	}

	return status
}
