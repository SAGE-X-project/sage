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
	"encoding/json"
	"net/http"
	"time"
)

// SnapshotHandler serves the global MetricsCollector snapshot as JSON. It is
// the handler pkg/health used to serve at /metrics; install it with
// health.Server.SetMetricsHandler(metrics.SnapshotHandler()), or use
// Handler() for the Prometheus exposition format.
func SnapshotHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		snapshot := GetGlobalCollector().GetSnapshot()
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
	})
}
