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
	"errors"
	"fmt"
	"time"

	"github.com/sage-x-project/sage/core/message"
	"github.com/sage-x-project/sage/core/message/dedupe"
	"github.com/sage-x-project/sage/core/message/nonce"
	"github.com/sage-x-project/sage/core/message/order"
)

// MessageValidator provides comprehensive validation for RequestMessage packets
type MessageValidator struct {
    config            *ValidatorConfig
    nonceManager      *nonce.Manager
    duplicateDetector *dedupe.Detector
    orderManager      *order.Manager
}

// NewMessageValidator creates a new message validator with the given configuration
func NewMessageValidator(config *ValidatorConfig) *MessageValidator {
    if config == nil {
        config = DefaultConfig()
    }
    
    return &MessageValidator{
        config:            config,
        nonceManager:      nonce.NewManager(config.NonceTTL, config.CleanupInterval),
        duplicateDetector: dedupe.NewDetector(config.DuplicateTTL, config.CleanupInterval),
        orderManager:      order.NewManager(),
    }
}

// ValidateMessage performs comprehensive validation on a RequestMessage
func (mv *MessageValidator) ValidateMessage(msg message.ControlHeader, sessionId string, msgId string) *ValidationResult {
    result := &ValidationResult{
        IsValid: true,
    }
    
    if err := mv.validateTimestamp(msg); err != nil {
        result.IsValid = false
        result.Error = err
        return result
    }
    
    if mv.nonceManager.IsNonceUsed(msg.GetNonce()) {
        result.IsValid = false
        result.IsReplay = true
        result.Error = errors.New("nonce has been used before (replay attack detected)")
        return result
    }
    
    if mv.duplicateDetector.IsDuplicate(msg) {
        result.IsValid = false
        result.IsDuplicate = true
        result.Error = errors.New("duplicate packet detected")
        return result
    }
    
    err := mv.orderManager.ProcessMessage(msg, sessionId)
    if err != nil {
        result.IsValid = false
        result.Error = fmt.Errorf("order validation failed: %w", err)
        return result
    }
    
    // If all validations pass, mark the message as processed
    mv.nonceManager.MarkNonceUsed(msg.GetNonce())
    mv.duplicateDetector.MarkPacketSeen(msg)
    
    return result
}

// validateTimestamp checks if the timestamp is within acceptable range
func (mv *MessageValidator) validateTimestamp(msg message.ControlHeader) error {
    msgTime := msg.GetTimestamp()
    if msgTime.IsZero() {
        return fmt.Errorf("empty timestamp")
    }
    
    now := time.Now()
    timeDiff := now.Sub(msgTime)
    if timeDiff < 0 {
        timeDiff = -timeDiff
    }
    
    if timeDiff > mv.config.TimestampTolerance {
        return fmt.Errorf("timestamp outside tolerance window: %v", timeDiff)
    }
    
    return nil
}

// GetStats returns current validation statistics
func (mv *MessageValidator) GetStats() map[string]interface{} {
    return map[string]interface{}{
        "tracked_nonces":     mv.nonceManager.GetUsedNonceCount(),
        "tracked_packets":    mv.duplicateDetector.GetSeenPacketCount(),
        "nonce_ttl_seconds":  int(mv.config.NonceTTL.Seconds()),
        "duplicate_ttl_seconds": int(mv.config.DuplicateTTL.Seconds()),
    }
}
