// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package auth0

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
)

var keyPath = "./testdata/"

func LoadOrCreateKeyPair(suffix string) (kp sagecrypto.KeyPair, privPEM, pubPEM []byte, err error) {
    privPath := filepath.Join(filepath.Dir(keyPath), "private_"+suffix+".pem")
    pubPath := filepath.Join(filepath.Dir(keyPath), "public_"+suffix+".pem")

    if data, err2 := os.ReadFile(privPath); err2 == nil {
        kp, err = formats.NewPEMImporter().Import(data, sagecrypto.KeyFormatPEM)
        if err != nil {
            return nil, nil, nil, fmt.Errorf("import existing key: %w", err)
        }
        pubPath := filepath.Join(filepath.Dir(keyPath), "public_"+suffix+".pem")
        if pubData, err3 := os.ReadFile(pubPath); err3 == nil {
            return kp, data, pubData, nil
        }
        fmt.Println("import existing key")
        return kp, data, nil, nil
    }

    kp, err = keys.GenerateRSAKeyPair()
    if err != nil {
        return nil, nil, nil, fmt.Errorf("generate key pair: %w", err)
    }

    privPEM, err = formats.NewPEMExporter().Export(kp, sagecrypto.KeyFormatPEM)
    if err != nil {
        return nil, nil, nil, fmt.Errorf("export private key: %w", err)
    }

    if err = os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
        return nil, nil, nil, fmt.Errorf("mkdir for key file: %w", err)
    }

    
    if err = os.WriteFile(privPath, privPEM, 0o600); err != nil {
        return nil, nil, nil, fmt.Errorf("write private key: %w", err)
    }

    pubDER, err := x509.MarshalPKIXPublicKey(kp.PublicKey())
    if err != nil {
        return nil, nil, nil, fmt.Errorf("marshal public key: %w", err)
    }
    pubBlock := &pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}
    pubPEM = pem.EncodeToMemory(pubBlock)

    if err = os.WriteFile(pubPath, pubPEM, 0o644); err != nil {
        return nil, nil, nil, fmt.Errorf("write public key PEM: %w", err)
    }

    return kp, privPEM, pubPEM, nil
}
