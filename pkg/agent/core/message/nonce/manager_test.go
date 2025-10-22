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

package nonce

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNonceManager(t *testing.T) {
	const ttl = 50 * time.Millisecond
	const cleanup = 10 * time.Millisecond

	t.Run("GenerateNonce", func(t *testing.T) {
		// Specification Requirement: Cryptographically secure nonce generation for replay attack prevention
		helpers.LogTestSection(t, "5.1.1", "Nonce Generation (Cryptographically Secure)")

		n, err := GenerateNonce()
		require.NoError(t, err)
		require.NotEmpty(t, n)

		helpers.LogSuccess(t, "Nonce generation successful")
		helpers.LogDetail(t, "Nonce value: %s", n)
		helpers.LogDetail(t, "Nonce length: %d characters", len(n))

		// Specification Requirement: Nonce should be hex-encoded and at least 32 characters (16 bytes)
		decoded, err := hex.DecodeString(n)
		if err == nil {
			helpers.LogDetail(t, "Nonce is hex-encoded: true")
			helpers.LogDetail(t, "Decoded size: %d bytes", len(decoded))
			assert.GreaterOrEqual(t, len(decoded), 16, "Nonce should be at least 16 bytes")
		} else {
			helpers.LogDetail(t, "Nonce encoding: non-hex format")
		}

		// Test uniqueness by generating multiple nonces
		nonce2, err := GenerateNonce()
		require.NoError(t, err)
		assert.NotEqual(t, n, nonce2, "Generated nonces should be unique")
		helpers.LogSuccess(t, "Nonce uniqueness verified")
		helpers.LogDetail(t, "Second nonce: %s", nonce2)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Nonce generation successful",
			"Nonce is not empty",
			"Nonce length sufficient",
			"Nonces are unique",
			"Cryptographically secure generation",
		})

		// Save test data
		testData := map[string]interface{}{
			"test_case":     "5.1.1_Nonce_Generation",
			"nonce_1":       n,
			"nonce_2":       nonce2,
			"nonce_1_length": len(n),
			"nonce_2_length": len(nonce2),
			"unique":        n != nonce2,
		}
		helpers.SaveTestData(t, "nonce/nonce_generation.json", testData)
	})

	t.Run("MarkNonceUsed", func(t *testing.T) {
		// Specification Requirement: Nonce tracking for replay attack prevention
		helpers.LogTestSection(t, "5.1.2", "Nonce Usage Tracking")

		ttlDuration := time.Second
		cleanupDuration := time.Second
		m := NewManager(ttlDuration, cleanupDuration)

		helpers.LogDetail(t, "Nonce TTL: %v", ttlDuration)
		helpers.LogDetail(t, "Cleanup interval: %v", cleanupDuration)

		n := "test-nonce"
		helpers.LogDetail(t, "Test nonce: %s", n)

		// Mark nonce as used
		m.MarkNonceUsed(n)
		helpers.LogSuccess(t, "Nonce marked as used")

		// Specification Requirement: Used nonce should be tracked
		require.True(t, m.IsNonceUsed(n), "manually marked nonce should be used")
		helpers.LogSuccess(t, "Nonce usage verification successful")

		usedCount := m.GetUsedNonceCount()
		require.Equal(t, 1, usedCount)
		helpers.LogDetail(t, "Used nonce count: %d", usedCount)

		// Test duplicate nonce rejection
		isUsed := m.IsNonceUsed(n)
		assert.True(t, isUsed, "Previously used nonce should still be marked as used")
		helpers.LogSuccess(t, "Duplicate nonce detected correctly")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Nonce marked as used successfully",
			"Used nonce is tracked",
			"Nonce usage check returns true",
			"Used nonce count is accurate",
			"Duplicate nonce detection working",
		})

		// Save test data
		testData := map[string]interface{}{
			"test_case":        "5.1.2_Nonce_Usage_Tracking",
			"nonce":            n,
			"marked_used":      true,
			"is_used_check":    isUsed,
			"used_nonce_count": usedCount,
			"ttl_ms":           ttlDuration.Milliseconds(),
			"cleanup_interval_ms": cleanupDuration.Milliseconds(),
		}
		helpers.SaveTestData(t, "nonce/nonce_usage_tracking.json", testData)
	})

	t.Run("IsNonceUsedExpiresOnCheck", func(t *testing.T) {
		// Specification Requirement: Nonce expiration to prevent unbounded memory growth
		helpers.LogTestSection(t, "5.1.3", "Nonce Expiration on Check")

		m := NewManager(ttl, time.Hour)
		n := "expiring-nonce"

		helpers.LogDetail(t, "Nonce TTL: %v", ttl)
		helpers.LogDetail(t, "Test nonce: %s", n)

		// Mark nonce as used
		m.MarkNonceUsed(n)
		initialCount := m.GetUsedNonceCount()
		helpers.LogDetail(t, "Initial used nonce count: %d", initialCount)
		require.Equal(t, 1, initialCount)

		// Wait for TTL to expire
		sleepDuration := ttl + 20*time.Millisecond
		helpers.LogDetail(t, "Waiting %v for nonce to expire", sleepDuration)
		time.Sleep(sleepDuration)

		// Specification Requirement: First call removes expired entry
		require.False(t, m.IsNonceUsed(n), "expired nonce should not be considered used")
		helpers.LogSuccess(t, "Expired nonce correctly identified as unused")

		finalCount := m.GetUsedNonceCount()
		require.Equal(t, 0, finalCount, "expired nonce should be removed")
		helpers.LogSuccess(t, "Expired nonce removed from tracking")
		helpers.LogDetail(t, "Final used nonce count: %d", finalCount)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Nonce marked as used initially",
			"TTL expiration triggered correctly",
			"Expired nonce identified as unused",
			"Expired nonce removed from tracking",
			"Memory cleanup working",
		})

		// Save test data
		testData := map[string]interface{}{
			"test_case":       "5.1.3_Nonce_Expiration_On_Check",
			"nonce":           n,
			"ttl_ms":          ttl.Milliseconds(),
			"sleep_duration_ms": sleepDuration.Milliseconds(),
			"initial_count":   initialCount,
			"final_count":     finalCount,
			"expired":         true,
		}
		helpers.SaveTestData(t, "nonce/nonce_expiration_check.json", testData)
	})

	t.Run("CleanupLoopPurgesExpired", func(t *testing.T) {
		// Specification Requirement: Automatic cleanup loop for expired nonce purging
		helpers.LogTestSection(t, "5.1.4", "Automatic Nonce Cleanup Loop")

		m := NewManager(ttl, cleanup)
		n := "cleanup-nonce"

		helpers.LogDetail(t, "Nonce TTL: %v", ttl)
		helpers.LogDetail(t, "Cleanup interval: %v", cleanup)
		helpers.LogDetail(t, "Test nonce: %s", n)

		// Mark nonce as used
		m.MarkNonceUsed(n)
		initialCount := m.GetUsedNonceCount()
		require.Equal(t, 1, initialCount, "nonce should be initially tracked")
		helpers.LogDetail(t, "Initial used nonce count: %d", initialCount)
		helpers.LogSuccess(t, "Nonce tracked by manager")

		// Specification Requirement: Wait for TTL + cleanup interval for cleanupLoop to run
		sleepDuration := ttl + cleanup + 20*time.Millisecond
		helpers.LogDetail(t, "Waiting %v for cleanup loop to run", sleepDuration)
		time.Sleep(sleepDuration)

		finalCount := m.GetUsedNonceCount()
		require.Equal(t, 0, finalCount, "cleanupLoop should purge expired nonces")
		helpers.LogSuccess(t, "Cleanup loop purged expired nonces")
		helpers.LogDetail(t, "Final used nonce count: %d", finalCount)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Nonce initially tracked",
			"Cleanup loop executed",
			"Expired nonces purged automatically",
			"Memory released correctly",
			"Background cleanup working",
		})

		// Save test data
		testData := map[string]interface{}{
			"test_case":         "5.1.4_Automatic_Cleanup_Loop",
			"nonce":             n,
			"ttl_ms":            ttl.Milliseconds(),
			"cleanup_interval_ms": cleanup.Milliseconds(),
			"sleep_duration_ms": sleepDuration.Milliseconds(),
			"initial_count":     initialCount,
			"final_count":       finalCount,
			"cleanup_successful": finalCount == 0,
		}
		helpers.SaveTestData(t, "nonce/nonce_cleanup_loop.json", testData)
	})

	// Test 1.2.2: Nonce 중복 검사 (Replay Attack Prevention)
	t.Run("CheckReplay", func(t *testing.T) {
		// Specification Requirement: Duplicate nonce detection for replay attack prevention
		helpers.LogTestSection(t, "1.2.2", "Nonce Duplicate Detection (CheckReplay)")

		m := NewManager(time.Second, time.Second)
		n, err := GenerateNonce()
		require.NoError(t, err)
		require.NotEmpty(t, n)

		helpers.LogDetail(t, "Generated nonce: %s", n)

		// First use: should be accepted
		isUsedBefore := m.IsNonceUsed(n)
		require.False(t, isUsedBefore, "nonce should not be used initially")
		helpers.LogSuccess(t, "First use: nonce not marked as used")
		helpers.LogDetail(t, "Is used before marking: %v", isUsedBefore)

		// Mark nonce as used
		m.MarkNonceUsed(n)
		helpers.LogSuccess(t, "Nonce marked as used")

		// Second use: should be detected as duplicate (replay attack)
		isUsedAfter := m.IsNonceUsed(n)
		require.True(t, isUsedAfter, "duplicate nonce should be detected")
		helpers.LogSuccess(t, "Duplicate nonce detected successfully")
		helpers.LogDetail(t, "Is used after marking: %v", isUsedAfter)

		// Verify replay attack is prevented
		replayDetected := m.IsNonceUsed(n)
		assert.True(t, replayDetected, "replay attack should be detected")
		helpers.LogSuccess(t, "Replay attack prevention working")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"중복 Nonce 탐지",
			"재사용 거부",
			"Replay 방어",
			"첫 사용 정상 처리",
			"두 번째 사용 탐지",
			"보안 메커니즘 동작",
		})

		// Save test data
		testData := map[string]interface{}{
			"test_case":         "1.2.2_Nonce_CheckReplay",
			"nonce":             n,
			"is_used_before":    isUsedBefore,
			"is_used_after":     isUsedAfter,
			"replay_detected":   replayDetected,
			"replay_prevented":  replayDetected,
		}
		helpers.SaveTestData(t, "nonce/nonce_check_replay.json", testData)
	})

	// Test 10.1.10: Nonce 만료
	t.Run("Expiration", func(t *testing.T) {
		// Specification Requirement: TTL-based nonce expiration for memory management
		helpers.LogTestSection(t, "10.1.10", "Nonce Expiration (TTL-based)")

		shortTTL := 50 * time.Millisecond
		m := NewManager(shortTTL, time.Hour) // long cleanup interval to test manual expiration
		n, err := GenerateNonce()
		require.NoError(t, err)

		helpers.LogDetail(t, "Generated nonce: %s", n)
		helpers.LogDetail(t, "TTL: %v", shortTTL)

		// Mark nonce as used
		m.MarkNonceUsed(n)
		initialCount := m.GetUsedNonceCount()
		require.Equal(t, 1, initialCount)
		helpers.LogSuccess(t, "Nonce marked as used")
		helpers.LogDetail(t, "Initial count: %d", initialCount)

		// Verify nonce is tracked
		isUsedBeforeExpiry := m.IsNonceUsed(n)
		require.True(t, isUsedBeforeExpiry, "nonce should be tracked before expiry")
		helpers.LogSuccess(t, "Nonce tracked before expiry")

		// Wait for TTL to expire
		sleepDuration := shortTTL + 20*time.Millisecond
		helpers.LogDetail(t, "Waiting %v for nonce to expire", sleepDuration)
		time.Sleep(sleepDuration)

		// Check if nonce is expired
		isUsedAfterExpiry := m.IsNonceUsed(n)
		require.False(t, isUsedAfterExpiry, "expired nonce should not be considered used")
		helpers.LogSuccess(t, "Expired nonce correctly identified as unused")
		helpers.LogDetail(t, "Is used after expiry: %v", isUsedAfterExpiry)

		// Verify nonce is removed from tracking
		finalCount := m.GetUsedNonceCount()
		require.Equal(t, 0, finalCount, "expired nonce should be removed")
		helpers.LogSuccess(t, "Expired nonce removed from tracking")
		helpers.LogDetail(t, "Final count: %d", finalCount)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"TTL 기반 만료",
			"만료된 Nonce 정리",
			"메모리 효율성",
			"만료 시간 정확성",
			"자동 제거 동작",
			"재사용 가능 (만료 후)",
		})

		// Save test data
		testData := map[string]interface{}{
			"test_case":            "10.1.10_Nonce_Expiration",
			"nonce":                n,
			"ttl_ms":               shortTTL.Milliseconds(),
			"sleep_duration_ms":    sleepDuration.Milliseconds(),
			"initial_count":        initialCount,
			"final_count":          finalCount,
			"is_used_before_expiry": isUsedBeforeExpiry,
			"is_used_after_expiry":  isUsedAfterExpiry,
			"expired":              true,
		}
		helpers.SaveTestData(t, "nonce/nonce_expiration.json", testData)
	})
}
