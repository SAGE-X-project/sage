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

package session

// Metrics receives the events this package emits. It is declared here, on the
// consumer side, so that the session layer has no dependency on a metrics
// backend; pkg/telemetry/metrics provides a Prometheus implementation
// (PrometheusSessionMetrics) that the composition root passes to
// Manager.SetMetrics. The default is NopMetrics.
type Metrics interface {
	// SessionCreated is called once per creation attempt.
	SessionCreated(success bool)
	// ActiveSessions adjusts the number of live sessions by delta (+1 / -1).
	ActiveSessions(delta int)
	// SessionExpired is called when the cleanup loop removes an expired session.
	SessionExpired()
	// CryptoOperation is called per Encrypt/Decrypt with op "encrypt" or
	// "decrypt" and result "success", "failure" or "expired".
	CryptoOperation(op, result string)
	// MessageSize reports the size of a processed message; direction is
	// "encrypted" (ciphertext produced) or "decrypted" (plaintext recovered).
	MessageSize(direction string, bytes int)
}

// NopMetrics discards every event. It is the default for Manager and for
// sessions created outside a Manager.
type NopMetrics struct{}

func (NopMetrics) SessionCreated(bool)            {}
func (NopMetrics) ActiveSessions(int)             {}
func (NopMetrics) SessionExpired()                {}
func (NopMetrics) CryptoOperation(string, string) {}
func (NopMetrics) MessageSize(string, int)        {}
