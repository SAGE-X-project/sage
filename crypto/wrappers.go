package crypto

// This file provides wrapper functions that will be implemented by a separate
// initialization package to avoid circular dependencies.

var (
	// generateEd25519KeyPair is the implementation function for Ed25519 key generation
	generateEd25519KeyPair func() (KeyPair, error)
	
	// generateSecp256k1KeyPair is the implementation function for Secp256k1 key generation
	generateSecp256k1KeyPair func() (KeyPair, error)
	
	// newMemoryKeyStorage is the implementation function for memory storage creation
	newMemoryKeyStorage func() KeyStorage
	
	// newJWKExporter is the implementation function for JWK exporter creation
	newJWKExporter func() KeyExporter
	
	// newPEMExporter is the implementation function for PEM exporter creation
	newPEMExporter func() KeyExporter
	
	// newJWKImporter is the implementation function for JWK importer creation
	newJWKImporter func() KeyImporter
	
	// newPEMImporter is the implementation function for PEM importer creation
	newPEMImporter func() KeyImporter
)

// SetKeyGenerators sets the key generation functions
func SetKeyGenerators(ed25519Gen, secp256k1Gen func() (KeyPair, error)) {
	generateEd25519KeyPair = ed25519Gen
	generateSecp256k1KeyPair = secp256k1Gen
}

// SetStorageConstructors sets the storage constructor functions
func SetStorageConstructors(memoryStorage func() KeyStorage) {
	newMemoryKeyStorage = memoryStorage
}

// SetFormatConstructors sets the format constructor functions
func SetFormatConstructors(jwkExp, pemExp func() KeyExporter, jwkImp, pemImp func() KeyImporter) {
	newJWKExporter = jwkExp
	newPEMExporter = pemExp
	newJWKImporter = jwkImp
	newPEMImporter = pemImp
}

// NewEd25519KeyPair generates a new Ed25519 key pair
func NewEd25519KeyPair() (KeyPair, error) {
	if generateEd25519KeyPair == nil {
		panic("Ed25519 key generator not initialized")
	}
	return generateEd25519KeyPair()
}

// NewSecp256k1KeyPair generates a new Secp256k1 key pair
func NewSecp256k1KeyPair() (KeyPair, error) {
	if generateSecp256k1KeyPair == nil {
		panic("Secp256k1 key generator not initialized")
	}
	return generateSecp256k1KeyPair()
}

// GenerateEd25519KeyPair is an alias for NewEd25519KeyPair
func GenerateEd25519KeyPair() (KeyPair, error) {
	return NewEd25519KeyPair()
}

// GenerateSecp256k1KeyPair is an alias for NewSecp256k1KeyPair
func GenerateSecp256k1KeyPair() (KeyPair, error) {
	return NewSecp256k1KeyPair()
}

// NewMemoryKeyStorage creates a new memory key storage
func NewMemoryKeyStorage() KeyStorage {
	if newMemoryKeyStorage == nil {
		panic("Memory key storage constructor not initialized")
	}
	return newMemoryKeyStorage()
}

// NewJWKExporter creates a new JWK exporter
func NewJWKExporter() KeyExporter {
	if newJWKExporter == nil {
		panic("JWK exporter constructor not initialized")
	}
	return newJWKExporter()
}

// NewPEMExporter creates a new PEM exporter
func NewPEMExporter() KeyExporter {
	if newPEMExporter == nil {
		panic("PEM exporter constructor not initialized")
	}
	return newPEMExporter()
}

// NewJWKImporter creates a new JWK importer
func NewJWKImporter() KeyImporter {
	if newJWKImporter == nil {
		panic("JWK importer constructor not initialized")
	}
	return newJWKImporter()
}

// NewPEMImporter creates a new PEM importer
func NewPEMImporter() KeyImporter {
	if newPEMImporter == nil {
		panic("PEM importer constructor not initialized")
	}
	return newPEMImporter()
}