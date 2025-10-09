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
	// SessionsCreated tracks total sessions created
	SessionsCreated = promauto.With(Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "sessions",
			Name:      "created_total",
			Help:      "Total number of sessions created",
		},
		[]string{"status"}, // success, failure
	)

	// SessionsActive tracks currently active sessions
	SessionsActive = promauto.With(Registry).NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "sessions",
			Name:      "active",
			Help:      "Number of currently active sessions",
		},
	)

	// SessionsExpired tracks expired sessions
	SessionsExpired = promauto.With(Registry).NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "sessions",
			Name:      "expired_total",
			Help:      "Total number of expired sessions",
		},
	)

	// SessionsClosed tracks closed sessions
	SessionsClosed = promauto.With(Registry).NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "sessions",
			Name:      "closed_total",
			Help:      "Total number of closed sessions",
		},
	)

	// SessionDuration tracks session operation duration
	SessionDuration = promauto.With(Registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "sessions",
			Name:      "duration_seconds",
			Help:      "Session operation duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 15), // 0.1ms to 1.6s
		},
		[]string{"operation"}, // create, encrypt, decrypt
	)

	// SessionMessageSize tracks message sizes
	SessionMessageSize = promauto.With(Registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "sessions",
			Name:      "message_size_bytes",
			Help:      "Size of messages processed by sessions",
			Buckets:   prometheus.ExponentialBuckets(64, 4, 10), // 64B to 16MB
		},
		[]string{"direction"}, // inbound, outbound
	)
)
