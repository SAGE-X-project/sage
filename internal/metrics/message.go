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
	// MessagesProcessed tracks processed messages
	MessagesProcessed = promauto.With(Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "messages",
			Name:      "processed_total",
			Help:      "Total number of messages processed",
		},
		[]string{"type", "status"}, // text/binary, success/failure
	)

	// ReplayAttacksDetected tracks detected replay attacks
	ReplayAttacksDetected = promauto.With(Registry).NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "messages",
			Name:      "replay_attacks_detected_total",
			Help:      "Total number of replay attacks detected",
		},
	)

	// NonceValidations tracks nonce validations
	NonceValidations = promauto.With(Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "messages",
			Name:      "nonce_validations_total",
			Help:      "Total number of nonce validations",
		},
		[]string{"status"}, // valid, invalid, expired
	)

	// MessageProcessingDuration tracks message processing duration
	MessageProcessingDuration = promauto.With(Registry).NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "messages",
			Name:      "processing_duration_seconds",
			Help:      "Message processing duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 12), // 0.1ms to 409ms
		},
	)

	// MessageSize tracks message sizes
	MessageSize = promauto.With(Registry).NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "messages",
			Name:      "size_bytes",
			Help:      "Message size in bytes",
			Buckets:   prometheus.ExponentialBuckets(64, 4, 10), // 64B to 16MB
		},
	)
)
