package config

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

// BlockchainConfig holds blockchain connection and contract configuration
type BlockchainConfig struct {
	NetworkRPC      string        `json:"network_rpc"`
	ContractAddr    string        `json:"contract_address"`
	ChainID         *big.Int      `json:"chain_id"`
	GasLimit        uint64        `json:"gas_limit"`
	MaxGasPrice     *big.Int      `json:"max_gas_price"`
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	RequestTimeout  time.Duration `json:"request_timeout"`
}

// NetworkPresets defines preset configurations for different networks
var NetworkPresets = map[string]*BlockchainConfig{
	"local": {
		NetworkRPC:     "http://localhost:8545",
		ChainID:        big.NewInt(31337),
		GasLimit:       3000000,
		MaxGasPrice:    big.NewInt(20000000000), // 20 Gwei
		MaxRetries:     3,
		RetryDelay:     time.Second,
		RequestTimeout: 30 * time.Second,
	},
	"kairos": {
		NetworkRPC:     "https://public-en-kairos.node.kaia.io",
		ChainID:        big.NewInt(1001),
		GasLimit:       5000000,
		MaxGasPrice:    big.NewInt(50000000000), // 50 Gwei
		MaxRetries:     5,
		RetryDelay:     2 * time.Second,
		RequestTimeout: 60 * time.Second,
	},
	"mainnet": {
		NetworkRPC:     "https://public-en-cypress.klaytn.net",
		ChainID:        big.NewInt(8217),
		GasLimit:       8000000,
		MaxGasPrice:    big.NewInt(100000000000), // 100 Gwei
		MaxRetries:     5,
		RetryDelay:     3 * time.Second,
		RequestTimeout: 90 * time.Second,
	},
}

// LoadConfig loads blockchain configuration from environment variables or uses defaults
func LoadConfig(env string) (*BlockchainConfig, error) {
	// Start with preset if available
	config, exists := NetworkPresets[strings.ToLower(env)]
	if !exists {
		config = NetworkPresets["local"] // Default to local
	}

	// Create a copy to avoid modifying the preset
	cfg := &BlockchainConfig{
		NetworkRPC:     config.NetworkRPC,
		ChainID:        new(big.Int).Set(config.ChainID),
		GasLimit:       config.GasLimit,
		MaxGasPrice:    new(big.Int).Set(config.MaxGasPrice),
		MaxRetries:     config.MaxRetries,
		RetryDelay:     config.RetryDelay,
		RequestTimeout: config.RequestTimeout,
	}

	// Override with environment variables if set
	if rpc := os.Getenv("SAGE_RPC_URL"); rpc != "" {
		cfg.NetworkRPC = rpc
	}

	// First try SAGE_REGISTRY_ADDRESS (new env var)
	if registryAddr := os.Getenv("SAGE_REGISTRY_ADDRESS"); registryAddr != "" {
		cfg.ContractAddr = registryAddr
	} else if contractAddr := os.Getenv("SAGE_CONTRACT_ADDRESS"); contractAddr != "" {
		// Fallback to old env var for compatibility
		cfg.ContractAddr = contractAddr
	}

	if chainID := os.Getenv("SAGE_CHAIN_ID"); chainID != "" {
		id, err := strconv.ParseInt(chainID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chain ID: %w", err)
		}
		cfg.ChainID = big.NewInt(id)
	}

	if gasLimit := os.Getenv("SAGE_GAS_LIMIT"); gasLimit != "" {
		limit, err := strconv.ParseUint(gasLimit, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid gas limit: %w", err)
		}
		cfg.GasLimit = limit
	}

	if maxGas := os.Getenv("SAGE_MAX_GAS_PRICE"); maxGas != "" {
		price, success := new(big.Int).SetString(maxGas, 10)
		if !success {
			return nil, fmt.Errorf("invalid max gas price: %s", maxGas)
		}
		cfg.MaxGasPrice = price
	}

	if retries := os.Getenv("SAGE_MAX_RETRIES"); retries != "" {
		r, err := strconv.Atoi(retries)
		if err != nil {
			return nil, fmt.Errorf("invalid max retries: %w", err)
		}
		cfg.MaxRetries = r
	}

	// Load contract address from deployment file if not set
	if cfg.ContractAddr == "" {
		// Try to load from deployment info
		if addr, err := GetContractAddress(env); err == nil {
			cfg.ContractAddr = addr
		} else {
			// Fallback to legacy method
			cfg.ContractAddr = loadContractFromDeployment(env)
		}
	}

	// Update config with deployment info if available
	_ = UpdateBlockchainConfig(cfg, env)

	return cfg, nil
}

// loadContractFromDeployment attempts to load contract address from deployment files
func loadContractFromDeployment(env string) string {
	deploymentPaths := []string{
		fmt.Sprintf("contracts/ethereum/deployments/%s.json", env),
		"contracts/ethereum/deployments/local.json",
		"contracts/ethereum/deployments/example.json",
	}

	for _, path := range deploymentPaths {
		if addr := readContractAddress(path); addr != "" {
			return addr
		}
	}

	return ""
}

// readContractAddress reads contract address from deployment JSON file
func readContractAddress(path string) string {
	// This is a placeholder - actual implementation would parse JSON
	// For now, return known addresses
	switch {
	case strings.Contains(path, "kairos"):
		return "0x4Ba6Fc825775eD9756104901b3d16DF1A1076545"
	case strings.Contains(path, "local"):
		// Will be set dynamically after deployment
		return os.Getenv("DEPLOYED_CONTRACT_ADDRESS")
	default:
		return ""
	}
}

// Validate checks if the configuration is valid
func (c *BlockchainConfig) Validate() error {
	if c.NetworkRPC == "" {
		return fmt.Errorf("network RPC URL is required")
	}

	if c.ChainID == nil || c.ChainID.Sign() <= 0 {
		return fmt.Errorf("valid chain ID is required")
	}

	if c.GasLimit == 0 {
		return fmt.Errorf("gas limit must be greater than 0")
	}

	if c.MaxGasPrice == nil || c.MaxGasPrice.Sign() <= 0 {
		return fmt.Errorf("valid max gas price is required")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	return nil
}

// GetRetryConfig returns retry configuration for the blockchain operations
func (c *BlockchainConfig) GetRetryConfig() (maxRetries int, delay time.Duration) {
	return c.MaxRetries, c.RetryDelay
}

// IsLocal returns true if the configuration is for a local network
func (c *BlockchainConfig) IsLocal() bool {
	return c.ChainID.Cmp(big.NewInt(31337)) == 0 ||
		strings.Contains(c.NetworkRPC, "localhost") ||
		strings.Contains(c.NetworkRPC, "127.0.0.1")
}