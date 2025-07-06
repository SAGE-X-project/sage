// Package cryptoinit initializes the crypto package with implementations
// from subpackages to avoid circular dependencies.
package cryptoinit

import (
	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/crypto/storage"
)

func init() {
	// Register key generators
	crypto.SetKeyGenerators(
		func() (crypto.KeyPair, error) { return keys.GenerateEd25519KeyPair() },
		func() (crypto.KeyPair, error) { return keys.GenerateSecp256k1KeyPair() },
	)
	
	// Register storage constructors
	crypto.SetStorageConstructors(
		func() crypto.KeyStorage { return storage.NewMemoryKeyStorage() },
	)
	
	// Register format constructors
	crypto.SetFormatConstructors(
		func() crypto.KeyExporter { return formats.NewJWKExporter() },
		func() crypto.KeyExporter { return formats.NewPEMExporter() },
		func() crypto.KeyImporter { return formats.NewJWKImporter() },
		func() crypto.KeyImporter { return formats.NewPEMImporter() },
	)
}