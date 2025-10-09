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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CryptoOperations tracks crypto operations
	CryptoOperations = promauto.With(Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "crypto",
			Name:      "operations_total",
			Help:      "Total number of cryptographic operations",
		},
		[]string{"operation", "algorithm"}, // sign/verify/encrypt/decrypt, ed25519/secp256k1/chacha20
	)

	// CryptoErrors tracks crypto errors
	CryptoErrors = promauto.With(Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "crypto",
			Name:      "errors_total",
			Help:      "Total number of cryptographic errors",
		},
		[]string{"operation"}, // sign, verify, encrypt, decrypt
	)

	// CryptoOperationDuration tracks crypto operation durations
	CryptoOperationDuration = promauto.With(Registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "crypto",
			Name:      "operation_duration_seconds",
			Help:      "Cryptographic operation duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.00001, 2, 15), // 10µs to 163ms
		},
		[]string{"operation", "algorithm"}, // sign/verify/encrypt/decrypt, ed25519/secp256k1/chacha20
	)
)
