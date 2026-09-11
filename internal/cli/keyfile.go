// Package cli holds helpers shared by the SAGE command-line tools. It is
// internal: binaries depend on it, library packages must not.
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
)

// keyFileWrapper is the envelope written by `sage-crypto generate`:
// {"private_key": <jwk>, "public_key": <jwk>, "key_id": "...", "key_type": "..."}.
type keyFileWrapper struct {
	PrivateKey json.RawMessage `json:"private_key"`
	PublicKey  json.RawMessage `json:"public_key"`
	KeyID      string          `json:"key_id"`
	KeyType    string          `json:"key_type"`
}

// LoadKeyPair loads a key pair either from a key file (JWK or PEM, including the
// sage-crypto wrapper envelope) or from a file key storage entry. Exactly one
// source must be given: keyFile+keyFormat, or storageDir+keyID.
func LoadKeyPair(keyFile, keyFormat, storageDir, keyID string) (crypto.KeyPair, error) {
	if storageDir != "" && keyID != "" {
		store, err := storage.NewFileKeyStorage(storageDir)
		if err != nil {
			return nil, fmt.Errorf("failed to open key storage: %w", err)
		}
		kp, err := store.Load(keyID)
		if err != nil {
			return nil, fmt.Errorf("failed to load key %q from storage: %w", keyID, err)
		}
		return kp, nil
	}
	if keyFile == "" {
		return nil, fmt.Errorf("no key source specified: use --key with --key-format, or --storage-dir with --key-id")
	}

	// #nosec G304 -- the path is supplied by the operator running the CLI
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	var importer crypto.KeyImporter
	var format crypto.KeyFormat
	switch keyFormat {
	case "jwk", "":
		importer = formats.NewJWKImporter()
		format = crypto.KeyFormatJWK
		var w keyFileWrapper
		if err := json.Unmarshal(data, &w); err == nil && len(w.PrivateKey) > 0 {
			data = w.PrivateKey
		}
	case "pem":
		importer = formats.NewPEMImporter()
		format = crypto.KeyFormatPEM
	default:
		return nil, fmt.Errorf("unsupported key format: %s (use jwk or pem)", keyFormat)
	}

	kp, err := importer.Import(data, format)
	if err != nil {
		return nil, fmt.Errorf("failed to import %s key from %s: %w", keyFormat, keyFile, err)
	}
	return kp, nil
}
