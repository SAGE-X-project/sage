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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsValidEthereumAddress tests Ethereum address validation
func TestIsValidEthereumAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{
			name:    "Valid address with 0x prefix",
			address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			valid:   true,
		},
		{
			name:    "Valid address without 0x prefix",
			address: "742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			valid:   true,
		},
		{
			name:    "Valid address with 0X prefix (uppercase)",
			address: "0X742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			valid:   true,
		},
		{
			name:    "Invalid address - too short",
			address: "0x742d35Cc6634C0532925",
			valid:   false,
		},
		{
			name:    "Invalid address - too long",
			address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb123",
			valid:   false,
		},
		{
			name:    "Invalid address - contains non-hex characters",
			address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbG",
			valid:   false,
		},
		{
			name:    "Invalid address - empty string",
			address: "",
			valid:   false,
		},
		{
			name:    "Invalid address - only 0x prefix",
			address: "0x",
			valid:   false,
		},
		{
			name:    "Valid address - all lowercase",
			address: "0x742d35cc6634c0532925a3b844bc9e7595f0aebb",
			valid:   true,
		},
		{
			name:    "Valid address - all uppercase",
			address: "0x742D35CC6634C0532925A3B844BC9E7595F0AEBB",
			valid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEthereumAddress(tt.address)
			assert.Equal(t, tt.valid, result, "Address validation result mismatch")
		})
	}
}

// TestParseDID tests DID parsing functionality
func TestParseDID(t *testing.T) {
	resolver := NewResolver()

	tests := []struct {
		name      string
		did       string
		wantError bool
		expected  *ParsedDID
	}{
		{
			name:      "Valid DID with ethereum",
			did:       "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			wantError: false,
			expected: &ParsedDID{
				Scheme:  "did",
				Method:  "sage",
				Network: "ethereum",
				Address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			},
		},
		{
			name:      "Valid DID with eth shorthand",
			did:       "did:sage:eth:0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			wantError: false,
			expected: &ParsedDID{
				Scheme:  "did",
				Method:  "sage",
				Network: "eth",
				Address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			},
		},
		{
			name:      "Invalid DID - wrong scheme",
			did:       "urn:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			wantError: true,
		},
		{
			name:      "Invalid DID - wrong method",
			did:       "did:ethr:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			wantError: true,
		},
		{
			name:      "Invalid DID - wrong network",
			did:       "did:sage:solana:0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			wantError: true,
		},
		{
			name:      "Invalid DID - invalid address",
			did:       "did:sage:ethereum:not-a-valid-address",
			wantError: true,
		},
		{
			name:      "Invalid DID - too few parts",
			did:       "did:sage:ethereum",
			wantError: true,
		},
		{
			name:      "Invalid DID - too many parts",
			did:       "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb:extra",
			wantError: true,
		},
		{
			name:      "Invalid DID - empty string",
			did:       "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ParseDID(tt.did)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Scheme, result.Scheme)
				assert.Equal(t, tt.expected.Method, result.Method)
				assert.Equal(t, tt.expected.Network, result.Network)
				assert.Equal(t, tt.expected.Address, result.Address)
			}
		})
	}
}

// TestCompareCapabilitiesEdgeCases tests additional edge cases for capability comparison
func TestCompareCapabilitiesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		cap1     map[string]interface{}
		cap2     map[string]interface{}
		expected bool
	}{
		{
			name: "Nested capabilities - complex structure",
			cap1: map[string]interface{}{
				"chat": true,
				"advanced": map[string]interface{}{
					"translation": true,
					"summarization": map[string]interface{}{
						"enabled":   true,
						"maxLength": 1000,
					},
				},
			},
			cap2: map[string]interface{}{
				"chat": true,
				"advanced": map[string]interface{}{
					"translation": true,
					"summarization": map[string]interface{}{
						"enabled":   true,
						"maxLength": 1000,
					},
				},
			},
			expected: true,
		},
		{
			name: "Deep nested capabilities - different",
			cap1: map[string]interface{}{
				"advanced": map[string]interface{}{
					"summarization": map[string]interface{}{
						"maxLength": 1000,
					},
				},
			},
			cap2: map[string]interface{}{
				"advanced": map[string]interface{}{
					"summarization": map[string]interface{}{
						"maxLength": 2000,
					},
				},
			},
			expected: false,
		},
		{
			name:     "Both nil capabilities",
			cap1:     nil,
			cap2:     nil,
			expected: true,
		},
		{
			name:     "Empty vs nil capabilities",
			cap1:     map[string]interface{}{},
			cap2:     nil,
			expected: false, // JSON marshaling treats {} and null differently
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareCapabilities(tt.cap1, tt.cap2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDIDCache tests cache functionality
func TestDIDCache(t *testing.T) {
	t.Run("Cache Set and Get", func(t *testing.T) {
		cache := &DIDCache{
			items:    make(map[string]*cacheItem),
			maxItems: 10,
			ttl:      5 * time.Minute,
		}

		doc := &DIDDocument{
			ID:         "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			Controller: "0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			PublicKey:  "0x04abcd...",
		}

		// Set and retrieve
		cache.Set("test-did", doc)
		retrieved := cache.Get("test-did")

		assert.NotNil(t, retrieved)
		assert.Equal(t, doc.ID, retrieved.ID)
		assert.Equal(t, doc.Controller, retrieved.Controller)
	})

	t.Run("Cache Get non-existent item", func(t *testing.T) {
		cache := &DIDCache{
			items:    make(map[string]*cacheItem),
			maxItems: 10,
			ttl:      5 * time.Minute,
		}

		retrieved := cache.Get("non-existent")
		assert.Nil(t, retrieved)
	})

	t.Run("Cache expiration", func(t *testing.T) {
		cache := &DIDCache{
			items:    make(map[string]*cacheItem),
			maxItems: 10,
			ttl:      100 * time.Millisecond,
		}

		doc := &DIDDocument{
			ID: "test-did",
		}

		cache.Set("test-did", doc)

		// Should be available immediately
		retrieved := cache.Get("test-did")
		assert.NotNil(t, retrieved)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Should be expired now
		retrieved = cache.Get("test-did")
		assert.Nil(t, retrieved)
	})

	t.Run("Cache eviction when full", func(t *testing.T) {
		cache := &DIDCache{
			items:    make(map[string]*cacheItem),
			maxItems: 2,
			ttl:      5 * time.Minute,
		}

		doc1 := &DIDDocument{ID: "did-1"}
		doc2 := &DIDDocument{ID: "did-2"}
		doc3 := &DIDDocument{ID: "did-3"}

		cache.Set("did-1", doc1)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		cache.Set("did-2", doc2)
		time.Sleep(10 * time.Millisecond)
		cache.Set("did-3", doc3) // Should evict oldest (did-1)

		// did-1 should be evicted
		assert.Nil(t, cache.Get("did-1"))
		// did-2 and did-3 should still be there
		assert.NotNil(t, cache.Get("did-2"))
		assert.NotNil(t, cache.Get("did-3"))
	})

	t.Run("Cache Clear", func(t *testing.T) {
		cache := &DIDCache{
			items:    make(map[string]*cacheItem),
			maxItems: 10,
			ttl:      5 * time.Minute,
		}

		doc1 := &DIDDocument{ID: "did-1"}
		doc2 := &DIDDocument{ID: "did-2"}

		cache.Set("did-1", doc1)
		cache.Set("did-2", doc2)

		// Verify items exist
		assert.NotNil(t, cache.Get("did-1"))
		assert.NotNil(t, cache.Get("did-2"))

		// Clear cache
		cache.Clear()

		// All items should be gone
		assert.Nil(t, cache.Get("did-1"))
		assert.Nil(t, cache.Get("did-2"))
	})

	t.Run("Cache Cleanup", func(t *testing.T) {
		cache := &DIDCache{
			items:    make(map[string]*cacheItem),
			maxItems: 10,
			ttl:      100 * time.Millisecond,
		}

		doc1 := &DIDDocument{ID: "did-1"}
		doc2 := &DIDDocument{ID: "did-2"}

		cache.Set("did-1", doc1)
		cache.Set("did-2", doc2)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Items should be expired but still in cache (lazy deletion)
		assert.Nil(t, cache.Get("did-1"))
		assert.Nil(t, cache.Get("did-2"))

		// Cleanup should remove them
		cache.Cleanup()

		// Verify items are physically removed
		cache.mu.RLock()
		itemCount := len(cache.items)
		cache.mu.RUnlock()
		assert.Equal(t, 0, itemCount)
	})
}

// TestNewResolver tests resolver creation
func TestNewResolver(t *testing.T) {
	t.Run("Create resolver without cache", func(t *testing.T) {
		resolver := NewResolver()

		assert.NotNil(t, resolver)
		assert.False(t, resolver.useCache)
		assert.Nil(t, resolver.cache)
	})

	t.Run("Create resolver with cache", func(t *testing.T) {
		maxItems := 100
		ttl := 10 * time.Minute

		resolver := NewResolverWithCache(maxItems, ttl)

		assert.NotNil(t, resolver)
		assert.True(t, resolver.useCache)
		assert.NotNil(t, resolver.cache)
		assert.Equal(t, maxItems, resolver.cache.maxItems)
		assert.Equal(t, ttl, resolver.cache.ttl)
		assert.NotNil(t, resolver.cache.items)
	})
}

// TestCacheConcurrency tests concurrent cache access
func TestCacheConcurrency(t *testing.T) {
	cache := &DIDCache{
		items:    make(map[string]*cacheItem),
		maxItems: 100,
		ttl:      5 * time.Minute,
	}

	// Run concurrent Set operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				did := &DIDDocument{ID: time.Now().String()}
				cache.Set(time.Now().String(), did)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cache should not panic and should have items
	cache.mu.RLock()
	itemCount := len(cache.items)
	cache.mu.RUnlock()
	assert.True(t, itemCount > 0)
	assert.True(t, itemCount <= 100) // Should not exceed maxItems
}
