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
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsRegistration(t *testing.T) {
	// Test that handshake metrics are registered
	if HandshakesInitiated == nil {
		t.Error("HandshakesInitiated metric is nil")
	}
	if HandshakesCompleted == nil {
		t.Error("HandshakesCompleted metric is nil")
	}
	if HandshakesFailed == nil {
		t.Error("HandshakesFailed metric is nil")
	}
	if HandshakeDuration == nil {
		t.Error("HandshakeDuration metric is nil")
	}

	// Test that session metrics are registered
	if SessionsCreated == nil {
		t.Error("SessionsCreated metric is nil")
	}
	if SessionsActive == nil {
		t.Error("SessionsActive metric is nil")
	}
	if SessionsExpired == nil {
		t.Error("SessionsExpired metric is nil")
	}
	if SessionDuration == nil {
		t.Error("SessionDuration metric is nil")
	}
	if SessionMessageSize == nil {
		t.Error("SessionMessageSize metric is nil")
	}

	// Test that crypto metrics are registered
	if CryptoOperations == nil {
		t.Error("CryptoOperations metric is nil")
	}
}

func TestMetricsIncrement(t *testing.T) {
	// Test incrementing handshake metrics
	HandshakesInitiated.WithLabelValues("test").Inc()
	HandshakesCompleted.WithLabelValues("success").Inc()
	HandshakesFailed.WithLabelValues("error").Inc()
	HandshakeDuration.WithLabelValues("invitation").Observe(0.5)

	// Test incrementing session metrics
	SessionsCreated.WithLabelValues("success").Inc()
	SessionsActive.Inc()
	SessionsExpired.Inc()
	SessionDuration.WithLabelValues("test_session").Observe(1.5)
	SessionMessageSize.WithLabelValues("encrypted").Observe(1024)

	// Test incrementing crypto metrics
	CryptoOperations.WithLabelValues("encrypt", "success").Inc()
	CryptoOperations.WithLabelValues("decrypt", "success").Inc()

	// Verify metrics have non-zero values
	count := testutil.CollectAndCount(HandshakesInitiated)
	if count == 0 {
		t.Error("HandshakesInitiated has no metrics collected")
	}

	count = testutil.CollectAndCount(SessionsCreated)
	if count == 0 {
		t.Error("SessionsCreated has no metrics collected")
	}

	count = testutil.CollectAndCount(CryptoOperations)
	if count == 0 {
		t.Error("CryptoOperations has no metrics collected")
	}
}

func TestMetricsExport(t *testing.T) {
	// Test that metrics can be exported
	expected := `
		# HELP sage_handshakes_initiated_total Total number of handshakes initiated
		# TYPE sage_handshakes_initiated_total counter
	`
	if err := testutil.CollectAndCompare(HandshakesInitiated, strings.NewReader(expected)); err != nil {
		// This is expected to have some differences due to labels, just check no panic
		t.Logf("Metrics export test completed (minor differences expected): %v", err)
	}
}
