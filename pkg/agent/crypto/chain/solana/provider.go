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


package solana

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/sage-x-project/sage/pkg/agent/crypto/chain"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// Provider implements ChainProvider for Solana
type Provider struct{}

// NewProvider creates a new Solana chain provider
func NewProvider() chain.ChainProvider {
	return &Provider{}
}

// ChainType returns the blockchain type
func (p *Provider) ChainType() chain.ChainType {
	return chain.ChainTypeSolana
}

// SupportedNetworks returns the list of supported networks
func (p *Provider) SupportedNetworks() []chain.Network {
	return []chain.Network{
		chain.NetworkSolanaMainnet,
		chain.NetworkSolanaDevnet,
		chain.NetworkSolanaTestnet,
	}
}

// GenerateAddress generates a Solana address from a public key
func (p *Provider) GenerateAddress(publicKey crypto.PublicKey, network chain.Network) (*chain.Address, error) {
	// Solana uses Ed25519 keys
	ed25519PubKey, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return nil, chain.ErrInvalidPublicKey
	}

	// Validate network
	if !p.isNetworkSupported(network) {
		return nil, chain.ErrNetworkNotSupported
	}

	// Solana addresses are base58 encoded public keys
	address := base58Encode(ed25519PubKey)

	return &chain.Address{
		Value:     address,
		Chain:     chain.ChainTypeSolana,
		Network:   network,
		PublicKey: publicKey,
	}, nil
}

// GetPublicKeyFromAddress retrieves the public key from an address
func (p *Provider) GetPublicKeyFromAddress(ctx context.Context, address string, network chain.Network) (crypto.PublicKey, error) {
	// Validate network
	if !p.isNetworkSupported(network) {
		return nil, chain.ErrNetworkNotSupported
	}

	// Decode base58 address to get public key
	pubKeyBytes, err := base58Decode(address)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base58 encoding", chain.ErrInvalidAddress)
	}

	// Verify it's a valid Ed25519 public key (32 bytes)
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: invalid public key size", chain.ErrInvalidAddress)
	}

	return ed25519.PublicKey(pubKeyBytes), nil
}

// ValidateAddress checks if an address is valid
func (p *Provider) ValidateAddress(address string, network chain.Network) error {
	// Validate network
	if !p.isNetworkSupported(network) {
		return chain.ErrNetworkNotSupported
	}

	// Try to decode as base58
	pubKeyBytes, err := base58Decode(address)
	if err != nil {
		return fmt.Errorf("%w: invalid base58 encoding", chain.ErrInvalidAddress)
	}

	// Check if it's 32 bytes (Ed25519 public key size)
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: invalid length", chain.ErrInvalidAddress)
	}

	return nil
}

// SignTransaction signs a transaction using a key pair
func (p *Provider) SignTransaction(keyPair sagecrypto.KeyPair, transaction interface{}) ([]byte, error) {
	// Check key type
	if keyPair.Type() != sagecrypto.KeyTypeEd25519 {
		return nil, fmt.Errorf("%w: Solana requires Ed25519 keys", chain.ErrInvalidPublicKey)
	}

	// Transaction signing would require full Solana transaction implementation
	// This is a placeholder for the actual implementation
	return nil, fmt.Errorf("transaction signing not yet implemented")
}

// VerifySignature verifies a signature
func (p *Provider) VerifySignature(publicKey crypto.PublicKey, message []byte, signature []byte) error {
	// Solana uses Ed25519 signatures
	ed25519PubKey, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return chain.ErrInvalidPublicKey
	}

	// Verify Ed25519 signature
	if !ed25519.Verify(ed25519PubKey, message, signature) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

func (p *Provider) isNetworkSupported(network chain.Network) bool {
	for _, n := range p.SupportedNetworks() {
		if n == network {
			return true
		}
	}
	return false
}

// base58Encode encodes bytes to base58 (Solana format)
func base58Encode(data []byte) string {
	return base58.Encode(data)
}

// base58Decode decodes base58 string to bytes
func base58Decode(s string) ([]byte, error) {
	return base58.Decode(s)
}

// init registers the provider
func init() {
	chain.RegisterProvider(NewProvider())
}
