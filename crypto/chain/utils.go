package chain

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"fmt"
	"strings"

	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// AddressFromKeyPair generates blockchain addresses for a key pair
func AddressFromKeyPair(keyPair sagecrypto.KeyPair, chains ...ChainType) (map[ChainType]*Address, error) {
	publicKey := keyPair.PublicKey()
	
	// If no chains specified, generate for all supported chains
	if len(chains) == 0 {
		return GenerateAddresses(publicKey)
	}

	// Generate for specified chains
	addresses := make(map[ChainType]*Address)
	for _, chainType := range chains {
		provider, err := GetProvider(chainType)
		if err != nil {
			continue // Skip unsupported chains
		}

		// Get the first supported network as default
		networks := provider.SupportedNetworks()
		if len(networks) == 0 {
			continue
		}

		address, err := provider.GenerateAddress(publicKey, networks[0])
		if err != nil {
			// Skip if key type is not supported by this chain
			if err == ErrInvalidPublicKey {
				continue
			}
			return nil, fmt.Errorf("failed to generate %s address: %w", chainType, err)
		}

		addresses[chainType] = address
	}

	return addresses, nil
}

// GetSupportedChainsForKey returns which blockchains support a given key type
func GetSupportedChainsForKey(keyPair sagecrypto.KeyPair) []ChainType {
	var supportedChains []ChainType
	publicKey := keyPair.PublicKey()
	
	// Query all registered providers to see which ones support this key type
	for _, chainType := range ListProviders() {
		provider, err := GetProvider(chainType)
		if err != nil {
			continue
		}
		
		// Try to generate an address with the first supported network
		networks := provider.SupportedNetworks()
		if len(networks) == 0 {
			continue
		}
		
		// If the provider can generate an address for this key, it's supported
		_, err = provider.GenerateAddress(publicKey, networks[0])
		if err == nil {
			supportedChains = append(supportedChains, chainType)
		}
	}
	
	return supportedChains
}

// GetKeyTypeForChain returns the required key type for a blockchain
func GetKeyTypeForChain(chain ChainType) (sagecrypto.KeyType, error) {
	switch chain {
	case ChainTypeEthereum:
		return sagecrypto.KeyTypeSecp256k1, nil
	case ChainTypeSolana:
		return sagecrypto.KeyTypeEd25519, nil
	case ChainTypeBitcoin:
		// Bitcoin uses secp256k1, but we don't have a provider implemented yet
		return sagecrypto.KeyTypeSecp256k1, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrChainNotSupported, chain)
	}
}

// ValidateKeyForChain checks if a key is compatible with a blockchain
func ValidateKeyForChain(publicKey crypto.PublicKey, chain ChainType) error {
	requiredKeyType, err := GetKeyTypeForChain(chain)
	if err != nil {
		return err
	}

	switch requiredKeyType {
	case sagecrypto.KeyTypeEd25519:
		if _, ok := publicKey.(ed25519.PublicKey); !ok {
			return fmt.Errorf("%w: %s requires Ed25519 keys", ErrInvalidPublicKey, chain)
		}
	case sagecrypto.KeyTypeSecp256k1:
		if _, ok := publicKey.(*ecdsa.PublicKey); !ok {
			return fmt.Errorf("%w: %s requires secp256k1 keys", ErrInvalidPublicKey, chain)
		}
	}

	return nil
}

// FormatAddress formats an address according to chain conventions
func FormatAddress(address *Address) string {
	switch address.Chain {
	case ChainTypeEthereum:
		// Ethereum addresses should have 0x prefix
		if !strings.HasPrefix(address.Value, "0x") {
			return "0x" + address.Value
		}
		return address.Value
		
	case ChainTypeSolana:
		// Solana addresses are base58 encoded, no prefix
		return address.Value
		
	default:
		return address.Value
	}
}

// ParseAddress parses an address string and attempts to determine its chain type
func ParseAddress(addressStr string) (*Address, error) {
	// Try Ethereum format
	if len(addressStr) == 42 && strings.HasPrefix(addressStr, "0x") {
		provider, err := GetProvider(ChainTypeEthereum)
		if err == nil {
			if err := provider.ValidateAddress(addressStr, NetworkEthereumMainnet); err == nil {
				return &Address{
					Value:   addressStr,
					Chain:   ChainTypeEthereum,
					Network: NetworkEthereumMainnet,
				}, nil
			}
		}
	}

	// Try Solana format (base58, typically 32-44 chars)
	if len(addressStr) >= 32 && len(addressStr) <= 44 {
		provider, err := GetProvider(ChainTypeSolana)
		if err == nil {
			if err := provider.ValidateAddress(addressStr, NetworkSolanaMainnet); err == nil {
				return &Address{
					Value:   addressStr,
					Chain:   ChainTypeSolana,
					Network: NetworkSolanaMainnet,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: unable to determine chain type", ErrInvalidAddress)
}