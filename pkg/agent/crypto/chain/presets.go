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

package chain

import (
	"math/big"
	"strings"
)

// Additional networks referenced by the presets.
const (
	NetworkEthereumLocal Network = "ethereum-local" // Hardhat / Anvil, chain id 31337
	NetworkKaiaKairos    Network = "kaia-kairos"    // Kaia testnet, chain id 1001
	NetworkKaiaCypress   Network = "kaia-cypress"   // Kaia mainnet, chain id 8217
)

// Preset is the single source of truth for a known network: its chain id,
// a public RPC endpoint and the SAGE registry contracts deployed on it.
// Deployment configuration, the CLIs and the examples read from this table
// instead of carrying their own copies.
type Preset struct {
	Name    string
	Chain   ChainType
	Network Network
	ChainID *big.Int
	RPCURL  string
	// AgentCardRegistry is the multi-key registry (current). Empty when SAGE
	// has not been deployed on the network.
	AgentCardRegistry string
	// SageRegistryV2 is the legacy single-key registry, kept for networks
	// where only it exists.
	SageRegistryV2 string
	// Local marks development networks whose registry address comes from a
	// deployment file or SAGE_REGISTRY_ADDRESS.
	Local bool
}

// presets lists every network SAGE knows about. Keep the deployed addresses
// in step with the README of github.com/SAGE-X-project/sage-contracts.
var presets = []Preset{
	{Name: "local", Chain: ChainTypeEthereum, Network: NetworkEthereumLocal, ChainID: big.NewInt(31337), RPCURL: "http://localhost:8545", Local: true},
	{Name: "sepolia", Chain: ChainTypeEthereum, Network: NetworkEthereumSepolia, ChainID: big.NewInt(11155111), RPCURL: "https://ethereum-sepolia-rpc.publicnode.com",
		AgentCardRegistry: "0xC7eCF7Ad6ee71CB0d94f0eb00F46f1DDf432a808"},
	{Name: "ethereum-mainnet", Chain: ChainTypeEthereum, Network: NetworkEthereumMainnet, ChainID: big.NewInt(1), RPCURL: "https://ethereum-rpc.publicnode.com"},
	{Name: "kairos", Chain: ChainTypeEthereum, Network: NetworkKaiaKairos, ChainID: big.NewInt(1001), RPCURL: "https://public-en-kairos.node.kaia.io",
		SageRegistryV2: "0x4Ba6Fc825775eD9756104901b3d16DF1A1076545"},
	{Name: "cypress", Chain: ChainTypeEthereum, Network: NetworkKaiaCypress, ChainID: big.NewInt(8217), RPCURL: "https://public-en-cypress.klaytn.net"},
	{Name: "solana-devnet", Chain: ChainTypeSolana, Network: NetworkSolanaDevnet, RPCURL: "https://api.devnet.solana.com"},
	{Name: "solana-mainnet", Chain: ChainTypeSolana, Network: NetworkSolanaMainnet, RPCURL: "https://api.mainnet-beta.solana.com"},
}

// presetAliases maps alternative spellings to preset names.
var presetAliases = map[string]string{
	"localhost": "local",
	"hardhat":   "local",
	"anvil":     "local",
	"mainnet":   "cypress", // historical name of the Kaia mainnet preset in deployment configs
	"kaia":      "cypress",
	"klaytn":    "cypress",
	"baobab":    "kairos",
	"ethereum":  "ethereum-mainnet",
	"devnet":    "solana-devnet",
}

// Presets returns a copy of the preset table.
func Presets() []Preset {
	out := make([]Preset, len(presets))
	copy(out, presets)
	return out
}

// PresetFor returns the preset for a network name or alias (case-insensitive).
func PresetFor(name string) (Preset, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if alias, ok := presetAliases[key]; ok {
		key = alias
	}
	for _, p := range presets {
		if p.Name == key {
			return p, true
		}
	}
	return Preset{}, false
}

// RegistryAddress returns the registry contract address to use on this
// network: the AgentCardRegistry when deployed, else the legacy registry.
func (p Preset) RegistryAddress() string {
	if p.AgentCardRegistry != "" {
		return p.AgentCardRegistry
	}
	return p.SageRegistryV2
}
