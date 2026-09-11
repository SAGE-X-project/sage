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

package crypto

import "errors"

// This file provides wrapper functions that will be implemented by a separate
// initialization package to avoid circular dependencies.

// ErrNotConfigured is returned by the deprecated generator wrappers when no
// generator was registered with SetKeyGenerators.
var ErrNotConfigured = errors.New("crypto: no implementation registered; use the keys, storage and formats packages directly")

var (
	generateEd25519KeyPair   func() (KeyPair, error)
	generateSecp256k1KeyPair func() (KeyPair, error)
	generateP256KeyPair      func() (KeyPair, error)
	newMemoryKeyStorage      func() KeyStorage
	newJWKExporter           func() KeyExporter
	newPEMExporter           func() KeyExporter
	newJWKImporter           func() KeyImporter
	newPEMImporter           func() KeyImporter
)

// SetKeyGenerators registers the functions behind the deprecated generator
// wrappers.
//
// Deprecated: call keys.GenerateEd25519KeyPair, keys.GenerateSecp256k1KeyPair
// and keys.GenerateP256KeyPair directly. Kept for consumers that still
// register implementations; it will be removed two minor releases after
// v1.6.0.
func SetKeyGenerators(ed25519Gen, secp256k1Gen, p256Gen func() (KeyPair, error)) {
	generateEd25519KeyPair = ed25519Gen
	generateSecp256k1KeyPair = secp256k1Gen
	generateP256KeyPair = p256Gen
}

// SetStorageConstructors registers the function behind NewMemoryKeyStorage.
//
// Deprecated: call storage.NewMemoryKeyStorage directly.
func SetStorageConstructors(memoryStorage func() KeyStorage) {
	newMemoryKeyStorage = memoryStorage
}

// SetFormatConstructors registers the functions behind the format wrappers.
//
// Deprecated: call formats.NewJWKExporter, formats.NewPEMExporter,
// formats.NewJWKImporter and formats.NewPEMImporter directly.
func SetFormatConstructors(jwkExp, pemExp func() KeyExporter, jwkImp, pemImp func() KeyImporter) {
	newJWKExporter = jwkExp
	newPEMExporter = pemExp
	newJWKImporter = jwkImp
	newPEMImporter = pemImp
}

func generate(gen func() (KeyPair, error)) (KeyPair, error) {
	if gen == nil {
		return nil, ErrNotConfigured
	}
	return gen()
}

// NewEd25519KeyPair generates an Ed25519 key pair through the registered generator.
//
// Deprecated: use keys.GenerateEd25519KeyPair.
func NewEd25519KeyPair() (KeyPair, error) { return generate(generateEd25519KeyPair) }

// NewSecp256k1KeyPair generates a secp256k1 key pair through the registered generator.
//
// Deprecated: use keys.GenerateSecp256k1KeyPair.
func NewSecp256k1KeyPair() (KeyPair, error) { return generate(generateSecp256k1KeyPair) }

// NewP256KeyPair generates a P-256 key pair through the registered generator.
//
// Deprecated: use keys.GenerateP256KeyPair.
func NewP256KeyPair() (KeyPair, error) { return generate(generateP256KeyPair) }

// GenerateEd25519KeyPair is an alias of NewEd25519KeyPair.
//
// Deprecated: use keys.GenerateEd25519KeyPair.
func GenerateEd25519KeyPair() (KeyPair, error) { return NewEd25519KeyPair() }

// GenerateSecp256k1KeyPair is an alias of NewSecp256k1KeyPair.
//
// Deprecated: use keys.GenerateSecp256k1KeyPair.
func GenerateSecp256k1KeyPair() (KeyPair, error) { return NewSecp256k1KeyPair() }

// GenerateP256KeyPair is an alias of NewP256KeyPair.
//
// Deprecated: use keys.GenerateP256KeyPair.
func GenerateP256KeyPair() (KeyPair, error) { return NewP256KeyPair() }

// NewMemoryKeyStorage returns the registered in-memory key storage, or nil
// when none was registered.
//
// Deprecated: use storage.NewMemoryKeyStorage.
func NewMemoryKeyStorage() KeyStorage {
	if newMemoryKeyStorage == nil {
		return nil
	}
	return newMemoryKeyStorage()
}

// NewJWKExporter returns the registered JWK exporter, or nil.
//
// Deprecated: use formats.NewJWKExporter.
func NewJWKExporter() KeyExporter {
	if newJWKExporter == nil {
		return nil
	}
	return newJWKExporter()
}

// NewPEMExporter returns the registered PEM exporter, or nil.
//
// Deprecated: use formats.NewPEMExporter.
func NewPEMExporter() KeyExporter {
	if newPEMExporter == nil {
		return nil
	}
	return newPEMExporter()
}

// NewJWKImporter returns the registered JWK importer, or nil.
//
// Deprecated: use formats.NewJWKImporter.
func NewJWKImporter() KeyImporter {
	if newJWKImporter == nil {
		return nil
	}
	return newJWKImporter()
}

// NewPEMImporter returns the registered PEM importer, or nil.
//
// Deprecated: use formats.NewPEMImporter.
func NewPEMImporter() KeyImporter {
	if newPEMImporter == nil {
		return nil
	}
	return newPEMImporter()
}
