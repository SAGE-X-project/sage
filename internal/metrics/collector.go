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


package metrics

import (
	"sync"
	"time"
)

// MetricsCollector collects metrics for SAGE operations
type MetricsCollector struct {
	mu sync.RWMutex

	// Counters
	SignatureCount      int64
	VerificationCount   int64
	SuccessfulVerifies  int64
	FailedVerifies      int64
	DIDResolutions      int64
	CacheHits           int64
	CacheMisses         int64
	BlockchainCalls     int64
	BlockchainErrors    int64

	// Timing metrics (in microseconds)
	SignatureTimes      []int64
	VerificationTimes   []int64
	BlockchainLatencies []int64
	DIDResolutionTimes  []int64

	// Start time for uptime calculation
	startTime time.Time

	// Configuration
	maxTimingSamples int
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:        time.Now(),
		maxTimingSamples: 1000, // Keep last 1000 samples for each timing metric
	}
}

// RecordSignature records a signature operation
func (mc *MetricsCollector) RecordSignature(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.SignatureCount++
	mc.recordTiming(&mc.SignatureTimes, duration)
}

// RecordVerification records a verification operation
func (mc *MetricsCollector) RecordVerification(success bool, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.VerificationCount++
	if success {
		mc.SuccessfulVerifies++
	} else {
		mc.FailedVerifies++
	}
	mc.recordTiming(&mc.VerificationTimes, duration)
}

// RecordDIDResolution records a DID resolution
func (mc *MetricsCollector) RecordDIDResolution(cached bool, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.DIDResolutions++
	if cached {
		mc.CacheHits++
	} else {
		mc.CacheMisses++
	}
	mc.recordTiming(&mc.DIDResolutionTimes, duration)
}

// RecordBlockchainCall records a blockchain call
func (mc *MetricsCollector) RecordBlockchainCall(success bool, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.BlockchainCalls++
	if !success {
		mc.BlockchainErrors++
	}
	mc.recordTiming(&mc.BlockchainLatencies, duration)
}

// recordTiming records a timing sample
func (mc *MetricsCollector) recordTiming(timings *[]int64, duration time.Duration) {
	microseconds := duration.Microseconds()
	*timings = append(*timings, microseconds)

	// Keep only last N samples
	if len(*timings) > mc.maxTimingSamples {
		*timings = (*timings)[len(*timings)-mc.maxTimingSamples:]
	}
}

// GetSnapshot returns a snapshot of current metrics
func (mc *MetricsCollector) GetSnapshot() *MetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &MetricsSnapshot{
		Timestamp:           time.Now(),
		Uptime:              time.Since(mc.startTime),
		SignatureCount:      mc.SignatureCount,
		VerificationCount:   mc.VerificationCount,
		SuccessfulVerifies:  mc.SuccessfulVerifies,
		FailedVerifies:      mc.FailedVerifies,
		DIDResolutions:      mc.DIDResolutions,
		CacheHits:           mc.CacheHits,
		CacheMisses:         mc.CacheMisses,
		BlockchainCalls:     mc.BlockchainCalls,
		BlockchainErrors:    mc.BlockchainErrors,
		AvgSignatureTime:    calculateAverage(mc.SignatureTimes),
		AvgVerificationTime: calculateAverage(mc.VerificationTimes),
		AvgBlockchainTime:   calculateAverage(mc.BlockchainLatencies),
		AvgDIDResolutionTime: calculateAverage(mc.DIDResolutionTimes),
		P95SignatureTime:    calculatePercentile(mc.SignatureTimes, 95),
		P95VerificationTime: calculatePercentile(mc.VerificationTimes, 95),
		P95BlockchainTime:   calculatePercentile(mc.BlockchainLatencies, 95),
		P95DIDResolutionTime: calculatePercentile(mc.DIDResolutionTimes, 95),
	}
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.SignatureCount = 0
	mc.VerificationCount = 0
	mc.SuccessfulVerifies = 0
	mc.FailedVerifies = 0
	mc.DIDResolutions = 0
	mc.CacheHits = 0
	mc.CacheMisses = 0
	mc.BlockchainCalls = 0
	mc.BlockchainErrors = 0

	mc.SignatureTimes = nil
	mc.VerificationTimes = nil
	mc.BlockchainLatencies = nil
	mc.DIDResolutionTimes = nil

	mc.startTime = time.Now()
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	Timestamp time.Time
	Uptime    time.Duration

	// Counters
	SignatureCount      int64
	VerificationCount   int64
	SuccessfulVerifies  int64
	FailedVerifies      int64
	DIDResolutions      int64
	CacheHits           int64
	CacheMisses         int64
	BlockchainCalls     int64
	BlockchainErrors    int64

	// Timing averages (microseconds)
	AvgSignatureTime     float64
	AvgVerificationTime  float64
	AvgBlockchainTime    float64
	AvgDIDResolutionTime float64

	// 95th percentile timings (microseconds)
	P95SignatureTime     int64
	P95VerificationTime  int64
	P95BlockchainTime    int64
	P95DIDResolutionTime int64
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (ms *MetricsSnapshot) GetCacheHitRate() float64 {
	total := ms.CacheHits + ms.CacheMisses
	if total == 0 {
		return 0
	}
	return float64(ms.CacheHits) / float64(total) * 100
}

// GetVerificationSuccessRate returns the verification success rate as a percentage
func (ms *MetricsSnapshot) GetVerificationSuccessRate() float64 {
	if ms.VerificationCount == 0 {
		return 0
	}
	return float64(ms.SuccessfulVerifies) / float64(ms.VerificationCount) * 100
}

// GetBlockchainErrorRate returns the blockchain error rate as a percentage
func (ms *MetricsSnapshot) GetBlockchainErrorRate() float64 {
	if ms.BlockchainCalls == 0 {
		return 0
	}
	return float64(ms.BlockchainErrors) / float64(ms.BlockchainCalls) * 100
}

// Helper functions

func calculateAverage(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum int64
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func calculatePercentile(values []int64, percentile int) int64 {
	if len(values) == 0 {
		return 0
	}

	// Simple implementation - for production, use a proper percentile algorithm
	// This is an approximation
	index := len(values) * percentile / 100
	if index >= len(values) {
		index = len(values) - 1
	}

	// Create a copy and sort (simple bubble sort for small datasets)
	sorted := make([]int64, len(values))
	copy(sorted, values)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted[index]
}

// Global metrics collector instance
var globalCollector = NewMetricsCollector()

// GetGlobalCollector returns the global metrics collector
func GetGlobalCollector() *MetricsCollector {
	return globalCollector
}
