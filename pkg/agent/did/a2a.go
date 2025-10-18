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

package did

import (
	"encoding/hex"
	"fmt"

	"github.com/mr-tron/base58"
)

// GenerateA2ACard creates a Google A2A protocol Agent Card from AgentMetadataV4
//
// The A2A Agent Card is a standardized format for representing AI agent metadata
// that enables interoperability between different AI agent platforms.
//
// Spec: https://github.com/a2aproject/a2a
func GenerateA2ACard(metadata *AgentMetadataV4) (*A2AAgentCard, error) {
	if metadata == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}

	// Convert keys to A2A format
	publicKeys := make([]A2APublicKey, 0, len(metadata.Keys))
	for i, key := range metadata.Keys {
		if !key.Verified {
			// Only include verified keys in the Agent Card
			continue
		}

		keyID := fmt.Sprintf("%s#key-%d", metadata.DID, i+1)
		keyType := mapKeyTypeToA2A(key.Type)

		publicKeys = append(publicKeys, A2APublicKey{
			ID:              keyID,
			Type:            keyType,
			Controller:      string(metadata.DID),
			PublicKeyBase58: base58.Encode(key.KeyData),
			PublicKeyHex:    hex.EncodeToString(key.KeyData),
		})
	}

	// Extract capabilities
	capabilities := extractCapabilities(metadata.Capabilities)

	// Create service endpoints
	endpoints := []A2AEndpoint{
		{
			Type: "MessageService",
			URI:  metadata.Endpoint,
		},
	}

	// Add additional endpoints from capabilities if present
	if serviceEndpoints, ok := metadata.Capabilities["endpoints"].([]interface{}); ok {
		for _, ep := range serviceEndpoints {
			if epMap, ok := ep.(map[string]interface{}); ok {
				epType, _ := epMap["type"].(string)
				epURI, _ := epMap["uri"].(string)
				if epType != "" && epURI != "" {
					endpoints = append(endpoints, A2AEndpoint{
						Type: epType,
						URI:  epURI,
					})
				}
			}
		}
	}

	card := &A2AAgentCard{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/ed25519-2020/v1",
			"https://w3id.org/security/suites/secp256k1-2019/v1",
		},
		ID:           string(metadata.DID),
		Type:         []string{"Agent", "AIAgent"},
		Name:         metadata.Name,
		Description:  metadata.Description,
		PublicKeys:   publicKeys,
		Endpoints:    endpoints,
		Capabilities: capabilities,
		Created:      metadata.CreatedAt,
		Updated:      metadata.UpdatedAt,
	}

	return card, nil
}

// mapKeyTypeToA2A converts SAGE KeyType to A2A key type string
func mapKeyTypeToA2A(keyType KeyType) string {
	switch keyType {
	case KeyTypeEd25519:
		return "Ed25519VerificationKey2020"
	case KeyTypeECDSA:
		return "EcdsaSecp256k1VerificationKey2019"
	case KeyTypeX25519:
		return "X25519KeyAgreementKey2019"
	default:
		return "UnknownKeyType"
	}
}

// extractCapabilities extracts capability strings from capabilities map
func extractCapabilities(capMap map[string]interface{}) []string {
	var capabilities []string

	// Try to extract from "capabilities" key
	if caps, ok := capMap["capabilities"].([]interface{}); ok {
		for _, cap := range caps {
			if capStr, ok := cap.(string); ok {
				capabilities = append(capabilities, capStr)
			}
		}
		return capabilities
	}

	// Try to extract from "functions" key
	if funcs, ok := capMap["functions"].([]interface{}); ok {
		for _, fn := range funcs {
			if fnStr, ok := fn.(string); ok {
				capabilities = append(capabilities, fnStr)
			}
		}
	}

	// Default capabilities if none specified
	if len(capabilities) == 0 {
		capabilities = []string{"message-signing", "message-verification"}
	}

	return capabilities
}

// ValidateA2ACard validates an A2A Agent Card structure
func ValidateA2ACard(card *A2AAgentCard) error {
	if card == nil {
		return fmt.Errorf("card cannot be nil")
	}

	if card.ID == "" {
		return fmt.Errorf("card ID is required")
	}

	if card.Name == "" {
		return fmt.Errorf("card name is required")
	}

	if len(card.PublicKeys) == 0 {
		return fmt.Errorf("at least one public key is required")
	}

	// Validate each public key
	for i, key := range card.PublicKeys {
		if key.ID == "" {
			return fmt.Errorf("public key %d: ID is required", i)
		}
		if key.Type == "" {
			return fmt.Errorf("public key %d: type is required", i)
		}
		if key.Controller == "" {
			return fmt.Errorf("public key %d: controller is required", i)
		}
		if key.PublicKeyBase58 == "" && key.PublicKeyHex == "" {
			return fmt.Errorf("public key %d: either publicKeyBase58 or publicKeyHex is required", i)
		}
	}

	// Validate endpoints
	if len(card.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint is required")
	}

	for i, endpoint := range card.Endpoints {
		if endpoint.Type == "" {
			return fmt.Errorf("endpoint %d: type is required", i)
		}
		if endpoint.URI == "" {
			return fmt.Errorf("endpoint %d: URI is required", i)
		}
	}

	return nil
}

// MergeA2ACard merges capabilities from an A2A Agent Card into AgentMetadataV4
func MergeA2ACard(metadata *AgentMetadataV4, card *A2AAgentCard) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}
	if card == nil {
		return fmt.Errorf("card cannot be nil")
	}

	// Validate the card first
	if err := ValidateA2ACard(card); err != nil {
		return fmt.Errorf("invalid A2A card: %w", err)
	}

	// Update basic metadata
	metadata.Name = card.Name
	metadata.Description = card.Description

	// Update capabilities
	if metadata.Capabilities == nil {
		metadata.Capabilities = make(map[string]interface{})
	}

	if len(card.Capabilities) > 0 {
		metadata.Capabilities["capabilities"] = card.Capabilities
	}

	// Extract endpoints
	if len(card.Endpoints) > 0 {
		endpoints := make([]map[string]interface{}, 0, len(card.Endpoints))
		for _, ep := range card.Endpoints {
			endpoints = append(endpoints, map[string]interface{}{
				"type": ep.Type,
				"uri":  ep.URI,
			})
		}
		metadata.Capabilities["endpoints"] = endpoints

		// Use first endpoint as primary
		metadata.Endpoint = card.Endpoints[0].URI
	}

	// Update timestamps
	if !card.Created.IsZero() {
		metadata.CreatedAt = card.Created
	}
	if !card.Updated.IsZero() {
		metadata.UpdatedAt = card.Updated
	}

	return nil
}
