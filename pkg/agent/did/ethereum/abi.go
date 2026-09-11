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
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed SageRegistryV2.abi.json
var sageRegistryV2ABI []byte

//go:embed AgentCardRegistry.abi.json
var agentCardRegistryABI []byte

// SageRegistryABI returns the ABI of the SageRegistry contract (V2)
func GetSageRegistryABI() (string, error) {
	// Validate the ABI is valid JSON
	var abiData []interface{}
	if err := json.Unmarshal(sageRegistryV2ABI, &abiData); err != nil {
		return "", fmt.Errorf("invalid ABI JSON: %w", err)
	}
	return string(sageRegistryV2ABI), nil
}

// GetAgentCardRegistryABI returns the ABI of the AgentCardRegistry contract
func GetAgentCardRegistryABI() (string, error) {
	// Validate the ABI is valid JSON
	var abiData []interface{}
	if err := json.Unmarshal(agentCardRegistryABI, &abiData); err != nil {
		return "", fmt.Errorf("invalid ABI JSON: %w", err)
	}
	return string(agentCardRegistryABI), nil
}

// SageRegistryABI is the ABI string (for backward compatibility)
// SageRegistryABI is the SageRegistry V2 ABI as a JSON string.
var SageRegistryABI = mustABI(GetSageRegistryABI)

// AgentCardRegistryABI is the AgentCardRegistry ABI as a JSON string.
var AgentCardRegistryABI = mustABI(GetAgentCardRegistryABI)

// mustABI validates an embedded ABI at package initialisation; the embedded
// files are part of the build, so a failure is a build defect, not a runtime condition.
func mustABI(load func() (string, error)) string {
	s, err := load()
	if err != nil {
		panic(fmt.Sprintf("embedded ABI is invalid: %v", err))
	}
	return s
}
