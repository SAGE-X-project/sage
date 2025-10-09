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


package ethereum

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sage-x-project/sage/did"
)

// ResolvePublicKey retrieves only the public key for an agent
func (c *EthereumClient) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (crypto.PublicKey, error) {
	metadata, err := c.Resolve(ctx, agentDID)
	if err != nil {
		return nil, err
	}
	
	if !metadata.IsActive {
		return nil, did.ErrInactiveAgent
	}
	
	return metadata.PublicKey, nil
}

// VerifyMetadata checks if the provided metadata matches the on-chain data
func (c *EthereumClient) VerifyMetadata(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
	// Fetch on-chain data
	onChainData, err := c.Resolve(ctx, agentDID)
	if err != nil {
		if errors.Is(err, did.ErrDIDNotFound) {
			return &did.VerificationResult{
				Valid:      false,
				Error:      "DID not found on chain",
				VerifiedAt: time.Now(),
			}, nil
		}
		return nil, err
	}
	
	// Compare metadata
	valid := true
	var errorMsg string
	
	if metadata.Name != onChainData.Name {
		valid = false
		errorMsg = fmt.Sprintf("name mismatch: expected %s, got %s", onChainData.Name, metadata.Name)
	}
	
	if metadata.Description != onChainData.Description {
		valid = false
		if errorMsg != "" {
			errorMsg += "; "
		}
		errorMsg += fmt.Sprintf("description mismatch")
	}
	
	if metadata.Endpoint != onChainData.Endpoint {
		valid = false
		if errorMsg != "" {
			errorMsg += "; "
		}
		errorMsg += fmt.Sprintf("endpoint mismatch: expected %s, got %s", onChainData.Endpoint, metadata.Endpoint)
	}
	
	// Compare capabilities (deep comparison)
	if !compareCapabilities(metadata.Capabilities, onChainData.Capabilities) {
		valid = false
		if errorMsg != "" {
			errorMsg += "; "
		}
		errorMsg += "capabilities mismatch"
	}
	
	result := &did.VerificationResult{
		Valid:      valid,
		Agent:      onChainData,
		VerifiedAt: time.Now(),
	}
	
	if !valid {
		result.Error = errorMsg
	}
	
	return result, nil
}

// ListAgentsByOwner retrieves all agents owned by a specific address
func (c *EthereumClient) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	// Validate address
	if !common.IsHexAddress(ownerAddress) {
		return nil, fmt.Errorf("invalid Ethereum address: %s", ownerAddress)
	}
	
	owner := common.HexToAddress(ownerAddress)
	
	// Prepare call data
	callData, err := c.contractABI.Pack("getAgentsByOwner", owner)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}
	
	// Make the call
	output, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contractAddress,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}
	
	// Unpack the result
	var dids []string
	err = c.contractABI.UnpackIntoInterface(&dids, "getAgentsByOwner", output)
	if err != nil {
		return nil, fmt.Errorf("failed to get agents by owner: %w", err)
	}
	
	// Fetch metadata for each DID
	agents := make([]*did.AgentMetadata, 0, len(dids))
	for _, agentDID := range dids {
		metadata, err := c.Resolve(ctx, did.AgentDID(agentDID))
		if err != nil {
			// Skip failed resolutions
			continue
		}
		agents = append(agents, metadata)
	}
	
	return agents, nil
}

// Search finds agents matching the given criteria
func (c *EthereumClient) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	// Note: This is a simplified implementation. In production, you would:
	// 1. Use events to build an index
	// 2. Query a graph protocol subgraph
	// 3. Use a separate indexing service
	
	// For now, we'll return an error indicating this needs off-chain indexing
	return nil, fmt.Errorf("search functionality requires off-chain indexing")
}

// GetRegistrationStatus checks the status of a registration transaction
func (c *EthereumClient) GetRegistrationStatus(ctx context.Context, txHash string) (*did.RegistrationResult, error) {
	hash := common.HexToHash(txHash)
	
	// Get transaction receipt
	receipt, err := c.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	
	// Check if transaction failed
	if receipt.Status == 0 {
		return nil, fmt.Errorf("transaction failed")
	}
	
	// Get block for timestamp
	block, err := c.client.BlockByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}
	
	return &did.RegistrationResult{
		TransactionHash: txHash,
		BlockNumber:     receipt.BlockNumber.Uint64(),
		Timestamp:       time.Unix(int64(block.Time()), 0),
		GasUsed:         receipt.GasUsed,
	}, nil
}

// Helper function to compare capabilities
func compareCapabilities(cap1, cap2 map[string]interface{}) bool {
	if len(cap1) != len(cap2) {
		return false
	}

	// Marshal both to JSON for deep comparison
	json1, err1 := json.Marshal(cap1)
	json2, err2 := json.Marshal(cap2)

	if err1 != nil || err2 != nil {
		return false
	}

	return strings.EqualFold(string(json1), string(json2))
}

// DIDDocument represents a DID document on Ethereum (for testing)
type DIDDocument struct {
	ID         string    `json:"id"`
	Controller string    `json:"controller"`
	PublicKey  string    `json:"publicKey"`
	Created    time.Time `json:"created"`
	Updated    time.Time `json:"updated"`
	Revoked    bool      `json:"revoked"`
}

// ParsedDID represents a parsed DID
type ParsedDID struct {
	Scheme  string // "did"
	Method  string // "sage"
	Network string // "ethereum"
	Address string // Ethereum address
}

// Resolver handles DID resolution for Ethereum
type Resolver struct {
	cache    *DIDCache
	useCache bool
}

// DIDCache provides caching for DID resolution
type DIDCache struct {
	mu       sync.RWMutex
	items    map[string]*cacheItem
	maxItems int
	ttl      time.Duration
}

type cacheItem struct {
	document  *DIDDocument
	expiresAt time.Time
}

// NewResolver creates a new DID resolver
func NewResolver() *Resolver {
	return &Resolver{
		useCache: false,
	}
}

// NewResolverWithCache creates a new DID resolver with caching
func NewResolverWithCache(maxItems int, ttl time.Duration) *Resolver {
	return &Resolver{
		cache: &DIDCache{
			items:    make(map[string]*cacheItem),
			maxItems: maxItems,
			ttl:      ttl,
		},
		useCache: true,
	}
}

// ParseDID parses a DID string into its components
func (r *Resolver) ParseDID(did string) (*ParsedDID, error) {
	// DID format: did:sage:ethereum:0x...
	parts := strings.Split(did, ":")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid DID format: expected 4 parts, got %d", len(parts))
	}

	if parts[0] != "did" {
		return nil, fmt.Errorf("invalid DID: scheme must be 'did'")
	}

	if parts[1] != "sage" {
		return nil, fmt.Errorf("invalid DID: method must be 'sage'")
	}

	if parts[2] != "ethereum" && parts[2] != "eth" {
		return nil, fmt.Errorf("invalid network: must be 'ethereum' or 'eth'")
	}

	// Validate Ethereum address
	address := parts[3]
	if !isValidEthereumAddress(address) {
		return nil, fmt.Errorf("invalid Ethereum address: %s", address)
	}

	return &ParsedDID{
		Scheme:  parts[0],
		Method:  parts[1],
		Network: parts[2],
		Address: address,
	}, nil
}

// Resolve resolves a DID to its document
func (r *Resolver) Resolve(ctx context.Context, did string) (*DIDDocument, error) {
	// Parse DID first
	parsedDID, err := r.ParseDID(did)
	if err != nil {
		return nil, err
	}

	// Check cache if enabled
	if r.useCache && r.cache != nil {
		if doc := r.cache.Get(did); doc != nil {
			return doc, nil
		}
	}

	// In a real implementation, this would query the blockchain
	// For now, return a mock document
	doc := &DIDDocument{
		ID:         did,
		Controller: parsedDID.Address,
		PublicKey:  "mock-public-key", // Would be fetched from blockchain
		Created:    time.Now(),
		Updated:    time.Now(),
		Revoked:    false,
	}

	// Store in cache if enabled
	if r.useCache && r.cache != nil {
		r.cache.Set(did, doc)
	}

	return doc, nil
}

// isValidEthereumAddress checks if a string is a valid Ethereum address
func isValidEthereumAddress(address string) bool {
	// Remove 0x prefix if present
	addr := strings.TrimPrefix(address, "0x")
	addr = strings.TrimPrefix(addr, "0X")

	// Check length (40 hex characters)
	if len(addr) != 40 {
		return false
	}

	// Check if all characters are hex
	for _, c := range addr {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// DIDCache methods

// Get retrieves a document from cache
func (c *DIDCache) Get(did string) *DIDDocument {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[did]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		// Don't delete here to avoid write lock, let cleanup handle it
		return nil
	}

	return item.document
}

// Set stores a document in cache
func (c *DIDCache) Set(did string, doc *DIDDocument) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: remove oldest if at capacity
	if len(c.items) >= c.maxItems {
		// Find and remove oldest item
		var oldestDID string
		var oldestTime time.Time
		for did, item := range c.items {
			if oldestTime.IsZero() || item.expiresAt.Before(oldestTime) {
				oldestDID = did
				oldestTime = item.expiresAt
			}
		}
		if oldestDID != "" {
			delete(c.items, oldestDID)
		}
	}

	c.items[did] = &cacheItem{
		document:  doc,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Clear removes all items from cache
func (c *DIDCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}

// Cleanup removes expired items from cache
func (c *DIDCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for did, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, did)
		}
	}
}
