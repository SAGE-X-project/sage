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

package crypto_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

func TestGetKeyTypeFromPublicKey(t *testing.T) {
	t.Run("Ed25519 public key", func(t *testing.T) {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		keyType, err := sagecrypto.GetKeyTypeFromPublicKey(pub)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeEd25519, keyType)
	})

	t.Run("ECDSA P-256 public key", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		keyType, err := sagecrypto.GetKeyTypeFromPublicKey(&priv.PublicKey)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeP256, keyType)
	})

	t.Run("ECDSA secp256k1 public key", func(t *testing.T) {
		kp, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		keyType, err := sagecrypto.GetKeyTypeFromPublicKey(kp.PublicKey())
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyType)
	})

	t.Run("ECDSA unsupported curve", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		require.NoError(t, err)

		_, err = sagecrypto.GetKeyTypeFromPublicKey(&priv.PublicKey)
		assert.Error(t, err)
	})

	t.Run("RSA public key", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		keyType, err := sagecrypto.GetKeyTypeFromPublicKey(&priv.PublicKey)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeRSA, keyType)
	})

	t.Run("Unsupported key type", func(t *testing.T) {
		_, err := sagecrypto.GetKeyTypeFromPublicKey("not a key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
}

func TestValidateAlgorithmForPublicKey(t *testing.T) {
	t.Run("Valid Ed25519 algorithm", func(t *testing.T) {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = sagecrypto.ValidateAlgorithmForPublicKey(pub, "ed25519")
		assert.NoError(t, err)
	})

	t.Run("Empty algorithm is allowed", func(t *testing.T) {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = sagecrypto.ValidateAlgorithmForPublicKey(pub, "")
		assert.NoError(t, err)
	})

	t.Run("Mismatched algorithm", func(t *testing.T) {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = sagecrypto.ValidateAlgorithmForPublicKey(pub, "es256k")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mismatch")
	})

	t.Run("Unsupported algorithm", func(t *testing.T) {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = sagecrypto.ValidateAlgorithmForPublicKey(pub, "unknown-algorithm")
		assert.Error(t, err)
	})

	t.Run("Valid ECDSA algorithm", func(t *testing.T) {
		p256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		assert.NoError(t, sagecrypto.ValidateAlgorithmForPublicKey(&p256.PublicKey, "ecdsa-p256-sha256"))
		assert.Error(t, sagecrypto.ValidateAlgorithmForPublicKey(&p256.PublicKey, "es256k"), "a P-256 key must not pass as secp256k1")

		k1, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		assert.NoError(t, sagecrypto.ValidateAlgorithmForPublicKey(k1.PublicKey(), "es256k"))
		assert.Error(t, sagecrypto.ValidateAlgorithmForPublicKey(k1.PublicKey(), "ecdsa-p256-sha256"))
	})

	t.Run("Valid RSA algorithm", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		err = sagecrypto.ValidateAlgorithmForPublicKey(&priv.PublicKey, "rsa-pss-sha256")
		assert.NoError(t, err)
	})
}
