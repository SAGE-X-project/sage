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
	// HandshakesInitiated tracks handshakes started
	HandshakesInitiated = promauto.With(Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "handshakes",
			Name:      "initiated_total",
			Help:      "Total number of handshakes initiated",
		},
		[]string{"role"}, // client, server
	)

	// HandshakesCompleted tracks completed handshakes
	HandshakesCompleted = promauto.With(Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "handshakes",
			Name:      "completed_total",
			Help:      "Total number of handshakes completed",
		},
		[]string{"status"}, // success, failure
	)

	// HandshakesFailed tracks failed handshakes by error type
	HandshakesFailed = promauto.With(Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "handshakes",
			Name:      "failed_total",
			Help:      "Total number of failed handshakes by error type",
		},
		[]string{"error_type"}, // timeout, invalid, network
	)

	// HandshakeDuration tracks handshake stage durations
	HandshakeDuration = promauto.With(Registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "handshakes",
			Name:      "duration_seconds",
			Help:      "Handshake stage duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to 4s
		},
		[]string{"stage"}, // init, process, finalize
	)
)
