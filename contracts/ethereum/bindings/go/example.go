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

//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	registry "./registry"
)

func main() {
	// Connect to Kaia node
	client, err := ethclient.Dial("https://public-en-kairos.node.kaia.io")
	if err != nil {
		log.Fatal(err)
	}

	// Contract address (replace with actual deployed address)
	contractAddress := common.HexToAddress("0x...")

	// Create contract instance
	instance, err := registry.NewSageRegistryV2(contractAddress, client)
	if err != nil {
		log.Fatal(err)
	}

	// Call contract method (example: get owner)
	owner, err := instance.Owner(&bind.CallOpts{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Contract owner: %s\n", owner.Hex())

	// Get agent by DID
	did := "did:sage:example"
	agent, err := instance.GetAgentByDID(&bind.CallOpts{}, did)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Agent name: %s\n", agent.Name)
	fmt.Printf("Agent active: %v\n", agent.Active)
}
