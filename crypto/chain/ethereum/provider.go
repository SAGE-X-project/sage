package ethereum

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/sage-x-project/sage/crypto/chain"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"golang.org/x/crypto/sha3"
)

// Provider implements ChainProvider for Ethereum
type Provider struct{}

// NewProvider creates a new Ethereum chain provider
func NewProvider() chain.ChainProvider {
	return &Provider{}
}

// ChainType returns the blockchain type
func (p *Provider) ChainType() chain.ChainType {
	return chain.ChainTypeEthereum
}

// SupportedNetworks returns the list of supported networks
func (p *Provider) SupportedNetworks() []chain.Network {
	return []chain.Network{
		chain.NetworkEthereumMainnet,
		chain.NetworkEthereumGoerli,
		chain.NetworkEthereumSepolia,
	}
}

// GenerateAddress generates an Ethereum address from a public key
func (p *Provider) GenerateAddress(publicKey crypto.PublicKey, network chain.Network) (*chain.Address, error) {
	// Ethereum uses secp256k1 keys
	ecdsaPubKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, chain.ErrInvalidPublicKey
	}

	// Validate network
	if !p.isNetworkSupported(network) {
		return nil, chain.ErrNetworkNotSupported
	}

	// Convert public key to uncompressed format (remove 0x04 prefix if present)
	pubKeyBytes := make([]byte, 64)
	ecdsaPubKey.X.FillBytes(pubKeyBytes[:32])
	ecdsaPubKey.Y.FillBytes(pubKeyBytes[32:])

	// Keccak256 hash of the public key
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubKeyBytes)
	addressBytes := hash.Sum(nil)

	// Take the last 20 bytes as the address
	address := "0x" + hex.EncodeToString(addressBytes[12:])

	return &chain.Address{
		Value:     address,
		Chain:     chain.ChainTypeEthereum,
		Network:   network,
		PublicKey: publicKey,
	}, nil
}

// GetPublicKeyFromAddress retrieves the public key from an address
// Note: This is not possible for Ethereum without additional transaction data
func (p *Provider) GetPublicKeyFromAddress(ctx context.Context, address string, network chain.Network) (crypto.PublicKey, error) {
	// Ethereum addresses are derived from public keys via one-way hash
	// Cannot recover public key from address alone
	return nil, chain.ErrOperationNotSupported
}

// ValidateAddress checks if an address is valid
func (p *Provider) ValidateAddress(address string, network chain.Network) error {
	// Remove 0x prefix if present
	address = strings.TrimPrefix(address, "0x")
	
	// Check length (20 bytes = 40 hex chars)
	if len(address) != 40 {
		return fmt.Errorf("%w: invalid length", chain.ErrInvalidAddress)
	}

	// Check if valid hex
	_, err := hex.DecodeString(address)
	if err != nil {
		return fmt.Errorf("%w: invalid hex encoding", chain.ErrInvalidAddress)
	}

	// Validate network
	if !p.isNetworkSupported(network) {
		return chain.ErrNetworkNotSupported
	}

	return nil
}

// SignTransaction signs a transaction using a key pair
func (p *Provider) SignTransaction(keyPair sagecrypto.KeyPair, transaction interface{}) ([]byte, error) {
	// Check key type
	if keyPair.Type() != sagecrypto.KeyTypeSecp256k1 {
		return nil, fmt.Errorf("%w: Ethereum requires secp256k1 keys", chain.ErrInvalidPublicKey)
	}

	// Transaction signing would require full Ethereum transaction implementation
	// This is a placeholder for the actual implementation
	return nil, fmt.Errorf("transaction signing not yet implemented")
}

// VerifySignature verifies a signature
func (p *Provider) VerifySignature(publicKey crypto.PublicKey, message []byte, signature []byte) error {
	// Ethereum uses secp256k1 ECDSA signatures
	ecdsaPubKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return chain.ErrInvalidPublicKey
	}

	// Hash the message with Keccak256
	hash := sha3.NewLegacyKeccak256()
	hash.Write(message)
	messageHash := hash.Sum(nil)

	// Verify signature (would need proper ECDSA verification)
	// This is a placeholder
	_ = ecdsaPubKey
	_ = messageHash
	
	return fmt.Errorf("signature verification not yet implemented")
}

func (p *Provider) isNetworkSupported(network chain.Network) bool {
	for _, n := range p.SupportedNetworks() {
		if n == network {
			return true
		}
	}
	return false
}

// init registers the provider
func init() {
	chain.RegisterProvider(NewProvider())
}