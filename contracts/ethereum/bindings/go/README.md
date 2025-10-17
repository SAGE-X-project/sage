# Go Bindings for SAGE Contracts

## Installation

```bash
go get github.com/ethereum/go-ethereum
```

## Usage

```go
import (
    "github.com/ethereum/go-ethereum/ethclient"
    registry "./bindings/go/registry"
)

// Connect to network
client, err := ethclient.Dial("https://public-en-kairos.node.kaia.io")

// Load contract
contract, err := registry.NewSageRegistryV2(address, client)

// Read data
agent, err := contract.GetAgentByDID(&bind.CallOpts{}, "did:sage:example")

// Write data (requires private key)
auth := bind.NewKeyedTransactor(privateKey)
tx, err := contract.RegisterAgent(auth, did, name, desc, endpoint, pubKey, capabilities, signature)
```

## Regenerate Bindings

```bash
npm run generate:go
```
