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

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// DeploymentInfo represents the deployment information saved after contract deployment
type DeploymentInfo struct {
	Network   string    `json:"network"`
	ChainID   int64     `json:"chainId"`
	Deployer  string    `json:"deployer"`
	Timestamp string    `json:"timestamp"`
	Contracts Contracts `json:"contracts"`
	Agents    []Agent   `json:"agents"`
	GasUsed   string    `json:"gasUsed"`
}

// Contracts contains deployed contract addresses
type Contracts struct {
	SageRegistryV2       ContractInfo `json:"SageRegistryV2"`
	SageVerificationHook ContractInfo `json:"SageVerificationHook"`
}

// ContractInfo contains details about a deployed contract
type ContractInfo struct {
	Address         string `json:"address"`
	TransactionHash string `json:"transactionHash"`
	BlockNumber     int64  `json:"blockNumber"`
	GasUsed         string `json:"gasUsed"`
}

// Agent represents a registered agent
type Agent struct {
	ID              string `json:"id"`
	DID             string `json:"did"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Endpoint        string `json:"endpoint"`
	Owner           string `json:"owner"`
	TransactionHash string `json:"transactionHash"`
	GasUsed         string `json:"gasUsed"`
}

// LoadDeploymentInfo loads deployment information from JSON file
func LoadDeploymentInfo(network string) (*DeploymentInfo, error) {
	// Try multiple paths to find deployment file
	possiblePaths := []string{
		filepath.Join("contracts", "ethereum", "deployments", fmt.Sprintf("%s.json", network)),
		filepath.Join("sage", "contracts", "ethereum", "deployments", fmt.Sprintf("%s.json", network)),
		filepath.Join("..", "contracts", "ethereum", "deployments", fmt.Sprintf("%s.json", network)),
		filepath.Join("deployments", fmt.Sprintf("%s.json", network)),
		// Try latest.json if network-specific file not found
		filepath.Join("contracts", "ethereum", "deployments", "latest.json"),
		filepath.Join("sage", "contracts", "ethereum", "deployments", "latest.json"),
	}

	var deploymentInfo DeploymentInfo
	var lastErr error

	for _, path := range possiblePaths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}

		err = json.Unmarshal(data, &deploymentInfo)
		if err != nil {
			lastErr = fmt.Errorf("failed to parse deployment file %s: %w", path, err)
			continue
		}

		// Successfully loaded
		return &deploymentInfo, nil
	}

	// If no file found, return the last error
	if lastErr != nil {
		return nil, fmt.Errorf("failed to load deployment info for network %s: %w", network, lastErr)
	}

	return nil, fmt.Errorf("no deployment file found for network %s", network)
}

// GetContractAddress returns the contract address for the given network
func GetContractAddress(network string) (string, error) {
	// First try environment variable
	if addr := os.Getenv("SAGE_REGISTRY_ADDRESS"); addr != "" {
		return addr, nil
	}

	// Try to load from deployment file
	info, err := LoadDeploymentInfo(network)
	if err != nil {
		// Fallback to known addresses
		switch network {
		case "kairos":
			return "0x4Ba6Fc825775eD9756104901b3d16DF1A1076545", nil
		case "local", "localhost", "hardhat":
			// For local network, must be set via environment variable
			return "", fmt.Errorf("local network contract address must be set via SAGE_REGISTRY_ADDRESS environment variable")
		default:
			return "", fmt.Errorf("unknown network %s and no deployment info found", network)
		}
	}

	if info.Contracts.SageRegistryV2.Address == "" {
		return "", fmt.Errorf("no SageRegistryV2 address found in deployment info")
	}

	return info.Contracts.SageRegistryV2.Address, nil
}

// UpdateBlockchainConfig updates the blockchain config with deployment information
func UpdateBlockchainConfig(cfg *BlockchainConfig, network string) error {
	// Try to load deployment info
	info, err := LoadDeploymentInfo(network)
	if err != nil {
		// Not fatal - can still use environment variables
		return nil
	}

	// Update contract address if not already set
	if cfg.ContractAddr == "" && info.Contracts.SageRegistryV2.Address != "" {
		cfg.ContractAddr = info.Contracts.SageRegistryV2.Address
	}

	// Update chain ID if it matches
	if info.ChainID > 0 && cfg.ChainID.Int64() == info.ChainID {
		// Configuration is consistent
		return nil
	}

	return nil
}

// SaveDeploymentAddress saves the contract address after deployment
func SaveDeploymentAddress(network, address string) error {
	// Create simple deployment info for Go applications
	deploymentDir := filepath.Join("deployments")
	if err := os.MkdirAll(deploymentDir, 0755); err != nil {
		return fmt.Errorf("failed to create deployment directory: %w", err)
	}

	// Save as simple JSON
	data := map[string]string{
		"network": network,
		"address": address,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal deployment data: %w", err)
	}

	file := filepath.Join(deploymentDir, fmt.Sprintf("%s-address.json", network))
	if err := ioutil.WriteFile(file, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write deployment file: %w", err)
	}

	// Also update environment variable for current session
	os.Setenv("DEPLOYED_CONTRACT_ADDRESS", address)

	return nil
}

// GetDeploymentAgents returns the list of registered agents from deployment
func GetDeploymentAgents(network string) ([]Agent, error) {
	info, err := LoadDeploymentInfo(network)
	if err != nil {
		return nil, err
	}
	return info.Agents, nil
}
