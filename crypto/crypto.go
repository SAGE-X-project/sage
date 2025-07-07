// Package crypto provides cryptographic operations for the SAGE system.
package crypto

// This file is intentionally minimal to avoid circular dependencies.
// The actual implementations are in the subpackages:
// - crypto/keys: Key pair generation and operations
// - crypto/storage: Key storage implementations
// - crypto/formats: Key import/export formats
// - crypto/chain: Blockchain integration
// - crypto/rotation: Key rotation support