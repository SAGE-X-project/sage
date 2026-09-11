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

// PrometheusSessionMetrics records session events into the Prometheus
// vectors of this package. It satisfies session.Metrics without this package
// importing session (Go interfaces are structural), so wiring is one line in
// the composition root:
//
//	mgr := session.NewManager()
//	mgr.SetMetrics(metrics.PrometheusSessionMetrics{})
type PrometheusSessionMetrics struct{}

func (PrometheusSessionMetrics) SessionCreated(success bool) {
	status := "failure"
	if success {
		status = "success"
	}
	SessionsCreated.WithLabelValues(status).Inc()
}

func (PrometheusSessionMetrics) ActiveSessions(delta int) {
	SessionsActive.Add(float64(delta))
}

func (PrometheusSessionMetrics) SessionExpired() {
	SessionsExpired.Inc()
}

func (PrometheusSessionMetrics) CryptoOperation(op, result string) {
	CryptoOperations.WithLabelValues(op, result).Inc()
}

func (PrometheusSessionMetrics) MessageSize(direction string, bytes int) {
	SessionMessageSize.WithLabelValues(direction).Observe(float64(bytes))
}
