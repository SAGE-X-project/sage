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


package dedupe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/message"
)

// Detector detects duplicate packets based on content hash
type Detector struct {
    ttl             time.Duration
    mu              sync.RWMutex
    seenPackets     map[string]time.Time
    cleanupInterval time.Duration
}

// NewDetector creates a new duplicate detector
func NewDetector(ttl, cleanupInterval time.Duration) *Detector {
    dd := &Detector{
        ttl:             ttl,
        seenPackets:     make(map[string]time.Time),
        cleanupInterval: cleanupInterval,
    }
    go dd.cleanupLoop()
    return dd
}

// IsDuplicate checks if a packet is a duplicate based on its content
func (dd *Detector) IsDuplicate(msg message.ControlHeader) bool {
    hash := dd.calculatePacketHash(msg)
    
    dd.mu.RLock()
    timestamp, exists := dd.seenPackets[hash]
    dd.mu.RUnlock()
    
    if !exists {
        return false
    }
    
    // Check if entry is expired
    if time.Since(timestamp) > dd.ttl {
        dd.mu.Lock()
        delete(dd.seenPackets, hash)
        dd.mu.Unlock()
        return false
    }
    
    return true
}

// MarkPacketSeen records a packet as seen
func (dd *Detector) MarkPacketSeen(msg message.ControlHeader) {
    hash := dd.calculatePacketHash(msg)
    
    dd.mu.Lock()
    defer dd.mu.Unlock()
    dd.seenPackets[hash] = time.Now()
}

// calculatePacketHash generates a hash for duplicate detection
// Excludes ContextID and signature from hash calculation
func (dd *Detector) calculatePacketHash(msg message.ControlHeader) string {
    // Create a normalized version for hashing
    hashData := &message.MessageControlHeader{
        Sequence:  msg.GetSequence(),  
		Nonce:     msg.GetNonce(),
		Timestamp: msg.GetTimestamp(),  
    }
    data, _ := json.Marshal(hashData)
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

// GetSeenPacketCount returns the number of packets currently being tracked
func (dd *Detector) GetSeenPacketCount() int {
    dd.mu.RLock()
    defer dd.mu.RUnlock()
    return len(dd.seenPackets)
}

// cleanupLoop periodically removes expired packet hashes
func (dd *Detector) cleanupLoop() {
    ticker := time.NewTicker(dd.cleanupInterval)
    defer ticker.Stop()
    
    for range ticker.C {
        dd.performCleanup()
    }
}

// performCleanup removes expired packet hashes
func (dd *Detector) performCleanup() {
    dd.mu.Lock()
    defer dd.mu.Unlock()
    
    now := time.Now()
    for hash, timestamp := range dd.seenPackets {
        if now.Sub(timestamp) > dd.ttl {
            delete(dd.seenPackets, hash)
        }
    }
}
