package chain

import (
	"context"
	"crypto"
	"errors"

	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// ChainType represents the type of blockchain
type ChainType string

const (
	ChainTypeEthereum ChainType = "ethereum"
	ChainTypeSolana   ChainType = "solana"
	ChainTypeBitcoin  ChainType = "bitcoin"
	ChainTypeCosmos   ChainType = "cosmos"
)

// Network represents a specific blockchain network
type Network string

const (
	// Ethereum networks
	NetworkEthereumMainnet Network = "ethereum-mainnet"
	NetworkEthereumGoerli  Network = "ethereum-goerli"
	NetworkEthereumSepolia Network = "ethereum-sepolia"
	
	// Solana networks
	NetworkSolanaMainnet Network = "solana-mainnet"
	NetworkSolanaDevnet  Network = "solana-devnet"
	NetworkSolanaTestnet Network = "solana-testnet"
	
	// Bitcoin networks
	NetworkBitcoinMainnet Network = "bitcoin-mainnet"
	NetworkBitcoinTestnet Network = "bitcoin-testnet"
)

// Address represents a blockchain address
type Address struct {
	// Value is the string representation of the address
	Value string
	
	// Chain is the blockchain type
	Chain ChainType
	
	// Network is the specific network
	Network Network
	
	// PublicKey is the associated public key (if known)
	PublicKey crypto.PublicKey
}

// ChainProvider defines the interface for blockchain-specific operations
type ChainProvider interface {
	// ChainType returns the blockchain type this provider supports
	ChainType() ChainType
	
	// SupportedNetworks returns the list of supported networks
	SupportedNetworks() []Network
	
	// GenerateAddress generates an address from a public key
	GenerateAddress(publicKey crypto.PublicKey, network Network) (*Address, error)
	
	// GetPublicKeyFromAddress retrieves the public key from an address (if possible)
	// Note: Not all blockchains support this operation
	GetPublicKeyFromAddress(ctx context.Context, address string, network Network) (crypto.PublicKey, error)
	
	// ValidateAddress checks if an address is valid for the chain
	ValidateAddress(address string, network Network) error
	
	// SignTransaction signs a transaction using a key pair
	// The transaction format is chain-specific
	SignTransaction(keyPair sagecrypto.KeyPair, transaction interface{}) ([]byte, error)
	
	// VerifySignature verifies a signature for the chain
	VerifySignature(publicKey crypto.PublicKey, message []byte, signature []byte) error
}

// ChainRegistry manages multiple chain providers
type ChainRegistry interface {
	// RegisterProvider registers a new chain provider
	RegisterProvider(provider ChainProvider) error
	
	// GetProvider returns a provider for the specified chain
	GetProvider(chain ChainType) (ChainProvider, error)
	
	// ListProviders returns all registered chain types
	ListProviders() []ChainType
	
	// GenerateAddresses generates addresses for all registered chains
	GenerateAddresses(publicKey crypto.PublicKey) (map[ChainType]*Address, error)
}

// PublicKeyResolver provides methods to resolve public keys from various sources
type PublicKeyResolver interface {
	// ResolveFromAddress attempts to resolve a public key from a blockchain address
	ResolveFromAddress(ctx context.Context, address string, chain ChainType, network Network) (crypto.PublicKey, error)
	
	// ResolveFromDID resolves a public key from a DID
	ResolveFromDID(ctx context.Context, did string) (crypto.PublicKey, error)
}

// Common errors
var (
	ErrChainNotSupported    = errors.New("blockchain not supported")
	ErrNetworkNotSupported  = errors.New("network not supported")
	ErrInvalidAddress       = errors.New("invalid address")
	ErrPublicKeyNotFound    = errors.New("public key not found")
	ErrProviderNotFound     = errors.New("chain provider not found")
	ErrProviderExists       = errors.New("chain provider already registered")
	ErrInvalidPublicKey     = errors.New("invalid public key for this chain")
	ErrOperationNotSupported = errors.New("operation not supported for this chain")
)