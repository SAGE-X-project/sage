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

package keys

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// KeyID derives the identifier every KeyPair in this package reports from its
// public key bytes: the first 8 bytes of SHA-256(publicKey), hex-encoded.
// It is the single definition of how key ids are formed.
func KeyID(publicKey []byte) string {
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:8])
}

// EthereumAddress derives the Ethereum account address of a secp256k1 public
// key: the last 20 bytes of Keccak-256(X || Y), as a lowercase 0x-prefixed hex
// string. It is the single address derivation used by the DID and chain
// packages.
func EthereumAddress(pub *ecdsa.PublicKey) (string, error) {
	if pub == nil || !IsSecp256k1Curve(pub.Curve) {
		return "", fmt.Errorf("ethereum address derivation requires a secp256k1 public key")
	}
	return "0x" + hex.EncodeToString(ethcrypto.PubkeyToAddress(*pub).Bytes()), nil
}
