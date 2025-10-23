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
