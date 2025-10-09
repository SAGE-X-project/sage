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

const GeneralPrefix = "session"

// Session represents an active cryptographic session between two agents
type Session interface {
    // Identification
    GetID() string
    GetCreatedAt() time.Time
    GetLastUsedAt() time.Time
    
    // Lifecycle
    IsExpired() bool
    UpdateLastUsed()
    Close() error
    
    // Cryptographic operations  
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
    EncryptAndSign(plaintext []byte, covered []byte) ([]byte, []byte, error)
    DecryptAndVerify(cipher []byte, covered []byte, mac []byte) ([]byte, error)
    SignCovered(covered []byte) []byte
    VerifyCovered(covered, sig []byte) error
    // Statistics
    GetMessageCount() int
    GetConfig() Config
}

// Config defines session policies and limits
type Config struct {
    MaxAge       time.Duration `json:"maxAge"`       // absolute expiration (ex: 1 hour)
    IdleTimeout  time.Duration `json:"idleTimeout"`  // idle timeout (ex: 10munutes) 
    MaxMessages  int           `json:"maxMessages"`
}


// Status provides information about session status
type Status struct {
    TotalSessions   int `json:"totalSessions"`
    ActiveSessions  int `json:"activeSessions"`
    ExpiredSessions int `json:"expiredSessions"`
}
