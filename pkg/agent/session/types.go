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

import (
	"time"
)

// Session is the part of *SecureSession that callers outside this package
// use. Manager methods return *SecureSession directly; keep this interface
// for code that stored the previous return type.
//
// Deprecated: use *SecureSession.
type Session interface {
	GetID() string
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	SignCovered(covered []byte) []byte
	VerifyCovered(covered, sig []byte) error
}

var _ Session = (*SecureSession)(nil)

// Config defines session policies and limits
type Config struct {
	MaxAge      time.Duration `json:"maxAge"`      // absolute expiration (ex: 1 hour)
	IdleTimeout time.Duration `json:"idleTimeout"` // idle timeout (ex: 10munutes)
	MaxMessages int           `json:"maxMessages"`
	// RekeyInterval is the number of messages per direction after which the
	// AEAD key is rotated (see the wire format in session.go). 0 disables
	// rotation; Manager substitutes DefaultRekeyInterval.
	RekeyInterval uint64 `json:"rekeyInterval"`
}

// Status provides information about session status
type Status struct {
	TotalSessions   int `json:"totalSessions"`
	ActiveSessions  int `json:"activeSessions"`
	ExpiredSessions int `json:"expiredSessions"`
}
