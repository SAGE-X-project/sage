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


package validator

import (
	"time"
)

// ValidationResult contains the result of packet validation
type ValidationResult struct {
    IsValid         bool
    Error           error
    IsReplay        bool
    IsDuplicate     bool
    IsOutOfOrder    bool
    ProcessedSeq    uint64
}

// ValidatorConfig holds configuration for the message validator
type ValidatorConfig struct {
    NonceTTL            time.Duration // How long to remember nonces
    DuplicateTTL        time.Duration // How long to remember packet hashes
    TimestampTolerance  time.Duration // How much timestamp drift to allow
    MaxOutOfOrderWindow time.Duration // Maximum time window for out-of-order messages
    CleanupInterval     time.Duration // How often to run cleanup
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *ValidatorConfig {
    return &ValidatorConfig{
        NonceTTL:            5 * time.Minute,
        DuplicateTTL:        3 * time.Minute,
        TimestampTolerance:  30 * time.Second,
        MaxOutOfOrderWindow: 2 * time.Minute,
        CleanupInterval:     1 * time.Minute,
    }
}
