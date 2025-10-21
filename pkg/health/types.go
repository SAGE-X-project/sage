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

// Status represents the overall health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// HealthStatus represents the complete health status of the system
type HealthStatus struct {
	Status           Status               `json:"status"`
	Timestamp        time.Time            `json:"timestamp"`
	BlockchainStatus *BlockchainHealth    `json:"blockchain,omitempty"`
	SystemStatus     *SystemHealth        `json:"system,omitempty"`
	Errors           []string             `json:"errors,omitempty"`
}

// BlockchainHealth represents blockchain connection health
type BlockchainHealth struct {
	Status      Status    `json:"status"`
	Connected   bool      `json:"connected"`
	ChainID     string    `json:"chain_id,omitempty"`
	BlockNumber uint64    `json:"block_number,omitempty"`
	NetworkRPC  string    `json:"network_rpc,omitempty"`
	Latency     string    `json:"latency,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// SystemHealth represents system resource health
type SystemHealth struct {
	Status         Status  `json:"status"`
	MemoryUsedMB   uint64  `json:"memory_used_mb"`
	MemoryTotalMB  uint64  `json:"memory_total_mb"`
	MemoryPercent  float64 `json:"memory_percent"`
	CPUPercent     float64 `json:"cpu_percent"`
	DiskUsedGB     uint64  `json:"disk_used_gb"`
	DiskTotalGB    uint64  `json:"disk_total_gb"`
	DiskPercent    float64 `json:"disk_percent"`
	GoRoutines     int     `json:"goroutines"`
	Error          string  `json:"error,omitempty"`
}
