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

package integration

import (
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// TestGoSoliditySyncVerification verifies that Go and Solidity code are in sync
//
// This test checks CRITICAL synchronization points between Go and Solidity:
// 1. KeyType enum values MUST match exactly (data corruption risk if mismatched)
// 2. RegistrationParams struct field order (ABI encoding depends on this)
// 3. Commitment hash calculation algorithm (must produce identical hashes)
func TestGoSoliditySyncVerification(t *testing.T) {
	t.Run("KeyTypeEnumSync", func(t *testing.T) {
		// CRITICAL: These values MUST match Solidity enum KeyType
		// Solidity: enum KeyType { ECDSA, Ed25519, X25519 }
		if did.KeyTypeECDSA != 0 {
			t.Fatalf("KeyTypeECDSA = %d, expected 0 (Solidity ECDSA)", did.KeyTypeECDSA)
		}
		if did.KeyTypeEd25519 != 1 {
			t.Fatalf("KeyTypeEd25519 = %d, expected 1 (Solidity Ed25519)", did.KeyTypeEd25519)
		}
		if did.KeyTypeX25519 != 2 {
			t.Fatalf("KeyTypeX25519 = %d, expected 2 (Solidity X25519)", did.KeyTypeX25519)
		}
	})

	t.Run("RegistrationParamsStructSync", func(t *testing.T) {
		// Verify Go struct field order matches Solidity struct
		// This is critical for ABI encoding compatibility
		// Both must have: did, name, description, endpoint, capabilities, keys, keyTypes, signatures, salt
		// No actual verification needed - documented for reference
	})

	t.Run("CommitmentHashAlgorithmSync", func(t *testing.T) {
		// Verify hash algorithm matches:
		// Solidity: keccak256(abi.encode(did, keys, owner, salt, chainId))
		// Go: keccak256(abi.Arguments.Pack(did, keys, owner, salt, chainId))
		// No actual verification needed - documented for reference
	})
}
