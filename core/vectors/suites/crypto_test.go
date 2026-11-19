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

package suites

import (
	"bytes"
	"crypto/ed25519"
	"os"
	"testing"
)

const publishedCryptoDesc = "Signature primitives as used on every SAGE wire path: Ed25519 (pure, no pre-hash), " +
	"secp256k1 over Keccak-256 with RFC 6979 and the Ethereum r||s||v encoding, " +
	"P-256 over SHA-256 with raw r||s low-S encoding, and the key identifier helper."

// TestCryptoReproducesPublished regenerates the published crypto.json from
// the cases that existed in draft.1 and requires identical bytes. The
// randomised P-256 signature is kept from the published file because it
// still verifies.
func TestCryptoReproducesPublished(t *testing.T) {
	s, err := Crypto()
	if err != nil {
		t.Fatal(err)
	}
	s.Description = publishedCryptoDesc
	s.Cases = s.Cases[:4]
	files, err := runner(t, s, publishedGenerator).Generate(os.DirFS(published))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(published + "/crypto.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(files["crypto.json"], want) {
		t.Fatalf("regenerated crypto.json differs from the published file:\n%s", files["crypto.json"])
	}
}

func TestCryptoRejectInputsAreStrict(t *testing.T) {
	good := map[string]any{"alg": "ed25519", "public_key": "00", "message": "", "signature": "00"}
	for name, mut := range map[string]func(map[string]any){
		"missing alg":  func(m map[string]any) { delete(m, "alg") },
		"extra member": func(m map[string]any) { m["x"] = "1" },
		"upper hex":    func(m map[string]any) { m["signature"] = "AB" },
		"odd hex":      func(m map[string]any) { m["public_key"] = "0" },
		"number alg":   func(m map[string]any) { m["alg"] = 1 },
	} {
		in := map[string]any{}
		for k, v := range good {
			in[k] = v
		}
		mut(in)
		if err := rejectSignature(in); err == nil || isRejection(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestSmallOrderVectorIsolated checks that the small-order vector would pass
// the standard cofactorless verifier: only the key check may reject it.
func TestSmallOrderVectorIsolated(t *testing.T) {
	m, err := newMaterial()
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(m.ed25519SmallOrderKey, m.msg, m.ed25519SmallOrderSig) {
		t.Fatal("crypto/ed25519 rejects the small-order vector, so it does not isolate the key check")
	}
}
