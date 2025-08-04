# SAGE - Secure Agent Guarantee Engine

A comprehensive framework for securing AI agent interactions through cryptographic verification, decentralized identity management, and RFC-9421 compliant HTTP message signatures.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
- [Agent Implementation Example](#agent-implementation-example)
- [Contract Deployment](#contract-deployment)
- [Agent Registration](#agent-registration)
  - [Generate Key Pair](#generate-key-pair)
  - [Register Agent Metadata](#register-agent-metadata)
- [Querying Agent Metadata via RPC](#querying-agent-metadata-via-rpc)
- [Project Structure](#project-structure)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [Issues](#issues)
- [License](#license)

## Overview

SAGE (Secure Agent Guarantee Engine) provides a robust infrastructure for AI agents to establish verifiable identities, securely communicate, and interact with blockchain systems. The framework ensures that AI agents can be trusted through cryptographic signatures and decentralized identity verification.

### Key Components

- **Core Library**: Central orchestration of crypto operations, DID management, and verification
- **Cryptographic Module**: Support for Ed25519 and Secp256k1 key pairs with secure storage
- **DID Module**: Multi-chain decentralized identifier management (Ethereum, Solana)
- **RFC-9421 Implementation**: HTTP message signatures for secure agent-to-agent communication
- **Smart Contracts**: On-chain agent registry for Ethereum and Solana

## Features

- **Multi-Chain Support**: Deploy agents on Ethereum and Solana
- **Cryptographic Security**: Ed25519 and Secp256k1 signature support
- **RFC-9421 Compliance**: Standardized HTTP message signatures
- **Decentralized Identity**: W3C DID-compliant agent identifiers
- **Secure Key Management**: File-based and memory storage with rotation support
- **Verification Hooks**: Customizable pre/post registration validation
- **Agent Capabilities**: Structured metadata for agent discovery

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Node.js 16+ and npm (for smart contract deployment)
- Ethereum development environment (Hardhat)
- Solana CLI tools (for Solana deployment)

### Installation

Clone the repository:

```bash
git clone https://github.com/sage-x-project/sage.git
cd sage
```

Install Go dependencies:

```bash
go mod download
```

Build the project:

```bash
make build
```

## Agent Implementation Example

Here's a complete example of implementing a SAGE-secured AI agent:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/sage-x-project/sage/core"
    "github.com/sage-x-project/sage/core/rfc9421"
    "github.com/sage-x-project/sage/crypto"
    "github.com/sage-x-project/sage/did"
)

// AgentServer represents a SAGE-secured AI agent
type AgentServer struct {
    core     *core.Core
    keyPair  crypto.KeyPair
    agentDID string
}

func NewAgentServer() (*AgentServer, error) {
    // Initialize SAGE core
    sageCore := core.New()
    
    // Generate key pair
    keyPair, err := sageCore.GenerateKeyPair(crypto.KeyTypeEd25519)
    if err != nil {
        return nil, fmt.Errorf("failed to generate key pair: %w", err)
    }
    
    // Configure for Ethereum
    err = sageCore.ConfigureDID(did.ChainEthereum, &did.RegistryConfig{
        RPC:      "https://eth-mainnet.alchemyapi.io/v2/YOUR-API-KEY",
        Contract: "0x...", // Your deployed registry address
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure DID: %w", err)
    }
    
    return &AgentServer{
        core:     sageCore,
        keyPair:  keyPair,
        agentDID: "did:sage:eth:agent001",
    }, nil
}

// HandleRequest processes incoming requests with SAGE verification
func (s *AgentServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Verify the incoming message
    headers := make(map[string]string)
    for k, v := range r.Header {
        if len(v) > 0 {
            headers[k] = v[0]
        }
    }
    
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("Signature")
    
    result, err := s.core.VerifyMessageFromHeaders(
        context.Background(),
        headers,
        body,
        []byte(signature),
    )
    
    if err != nil || !result.Valid {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Process the request
    response := map[string]interface{}{
        "status": "success",
        "agent":  s.agentDID,
        "time":   time.Now().Unix(),
    }
    
    // Sign the response
    respBody, _ := json.Marshal(response)
    signature, _ = s.core.SignMessage(s.keyPair, respBody)
    
    w.Header().Set("X-Agent-DID", s.agentDID)
    w.Header().Set("Signature", string(signature))
    w.Header().Set("Content-Type", "application/json")
    w.Write(respBody)
}

// MakeSignedRequest creates a SAGE-signed outgoing request
func (s *AgentServer) MakeSignedRequest(url string, data interface{}) error {
    // Create message
    body, _ := json.Marshal(data)
    message := s.core.CreateRFC9421Message(s.agentDID, body).
        WithHeader("Content-Type", "application/json").
        WithHeader("Date", time.Now().UTC().Format(http.TimeFormat)).
        Build()
    
    // Sign message
    signature, err := s.core.SignMessage(s.keyPair, message.Body)
    if err != nil {
        return err
    }
    
    // Create HTTP request
    req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
    req.Header.Set("X-Agent-DID", s.agentDID)
    req.Header.Set("Signature", string(signature))
    req.Header.Set("Content-Type", "application/json")
    
    // Send request
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

func main() {
    server, err := NewAgentServer()
    if err != nil {
        log.Fatal(err)
    }
    
    http.HandleFunc("/api/agent", server.HandleRequest)
    
    fmt.Println("SAGE Agent Server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Contract Deployment

### Ethereum Deployment

1. Navigate to the contracts directory:

```bash
cd contracts/ethereum
```

2. Install dependencies:

```bash
npm install
```

3. Configure your deployment in `hardhat.config.js`:

```javascript
module.exports = {
  networks: {
    mainnet: {
      url: process.env.ETH_RPC_URL,
      accounts: [process.env.PRIVATE_KEY]
    }
  }
};
```

4. Deploy the contracts:

```bash
npx hardhat run scripts/deploy.js --network mainnet
```

### Solana Deployment

1. Navigate to the Solana contracts:

```bash
cd contracts/solana
```

2. Build the program:

```bash
cargo build-bpf
```

3. Deploy:

```bash
solana program deploy target/deploy/sage_registry.so
```

## Agent Registration

### Generate Key Pair

Use the SAGE CLI to generate a key pair:

```bash
# Generate Ed25519 key pair
sage-crypto generate --type ed25519 --output agent-key.json

# Generate Secp256k1 key pair (for Ethereum)
sage-crypto generate --type secp256k1 --output agent-key-eth.json
```

### Register Agent Metadata

Here's an example of registering an agent on Ethereum:

```javascript
const { ethers } = require('ethers');
const fs = require('fs');

async function registerAgent() {
    // Setup provider and signer
    const provider = new ethers.providers.JsonRpcProvider(process.env.RPC_URL);
    const signer = new ethers.Wallet(process.env.PRIVATE_KEY, provider);
    
    // Load contract
    const registryABI = require('./artifacts/contracts/SageRegistry.sol/SageRegistry.json').abi;
    const registry = new ethers.Contract(REGISTRY_ADDRESS, registryABI, signer);
    
    // Prepare agent metadata
    const agentData = {
        did: "did:sage:eth:myagent001",
        name: "My AI Assistant",
        description: "An AI agent for code generation and analysis",
        endpoint: "https://api.myagent.com/v1",
        publicKey: "0x...", // Your public key
        capabilities: JSON.stringify({
            models: ["gpt-4", "claude-3"],
            skills: ["code-generation", "analysis", "testing"],
            languages: ["en", "es", "ko"]
        })
    };
    
    // Create signature
    const messageHash = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
            ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
            [
                agentData.did,
                agentData.name,
                agentData.description,
                agentData.endpoint,
                agentData.publicKey,
                agentData.capabilities,
                signer.address,
                0 // nonce
            ]
        )
    );
    
    const signature = await signer.signMessage(ethers.utils.arrayify(messageHash));
    
    // Register agent
    const tx = await registry.registerAgent(
        agentData.did,
        agentData.name,
        agentData.description,
        agentData.endpoint,
        agentData.publicKey,
        agentData.capabilities,
        signature
    );
    
    const receipt = await tx.wait();
    console.log("Agent registered! Transaction:", receipt.transactionHash);
    
    // Get agent ID from event
    const event = receipt.events.find(e => e.event === 'AgentRegistered');
    console.log("Agent ID:", event.args.agentId);
}

registerAgent().catch(console.error);
```

## Querying Agent Metadata via RPC

### Using Go

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/sage-x-project/sage/core"
    "github.com/sage-x-project/sage/did"
)

func main() {
    // Initialize SAGE core
    sageCore := core.New()
    
    // Configure for Ethereum
    err := sageCore.ConfigureDID(did.ChainEthereum, &did.RegistryConfig{
        RPC:      "https://eth-mainnet.alchemyapi.io/v2/YOUR-API-KEY",
        Contract: "0x...", // Registry contract address
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Resolve agent by DID
    agentDID := "did:sage:eth:myagent001"
    metadata, err := sageCore.ResolveAgent(context.Background(), agentDID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Agent Name: %s\n", metadata.Name)
    fmt.Printf("Description: %s\n", metadata.Description)
    fmt.Printf("Endpoint: %s\n", metadata.Endpoint)
    fmt.Printf("Active: %v\n", metadata.Active)
    fmt.Printf("Capabilities: %s\n", metadata.Capabilities)
}
```

### Using JavaScript/Web3

```javascript
const { ethers } = require('ethers');

async function queryAgent() {
    const provider = new ethers.providers.JsonRpcProvider(RPC_URL);
    const registryABI = require('./artifacts/contracts/SageRegistry.sol/SageRegistry.json').abi;
    const registry = new ethers.Contract(REGISTRY_ADDRESS, registryABI, provider);
    
    // Query by DID
    const agentDID = "did:sage:eth:myagent001";
    const metadata = await registry.getAgentByDID(agentDID);
    
    console.log("Agent Metadata:");
    console.log("- Name:", metadata.name);
    console.log("- Description:", metadata.description);
    console.log("- Endpoint:", metadata.endpoint);
    console.log("- Active:", metadata.active);
    console.log("- Owner:", metadata.owner);
    console.log("- Capabilities:", metadata.capabilities);
    
    // Parse capabilities
    const capabilities = JSON.parse(metadata.capabilities);
    console.log("- Supported Models:", capabilities.models);
    console.log("- Skills:", capabilities.skills);
}

queryAgent().catch(console.error);
```

## Project Structure

```
sage/
├── cmd/                    # CLI applications
│   ├── sage-crypto/       # Cryptographic operations CLI
│   └── sage-did/          # DID management CLI
├── contracts/             # Smart contracts
│   ├── ethereum/          # Ethereum contracts
│   └── solana/            # Solana programs
├── core/                  # Core library
│   ├── rfc9421/          # RFC-9421 implementation
│   └── verification.go    # Verification service
├── crypto/                # Cryptographic operations
│   ├── chain/            # Blockchain-specific crypto
│   ├── keys/             # Key pair implementations
│   └── storage/          # Key storage
├── did/                   # DID management
│   ├── ethereum/         # Ethereum DID resolver
│   └── solana/           # Solana DID resolver
├── examples/              # Example implementations
│   └── mcp-integration/  # Model Context Protocol examples
└── docs/                  # Documentation
```

## Documentation

- [CLI Documentation](docs/cli/README.md)
- [Core Library Guide](docs/core/README.md)
- [RFC-9421 Implementation Details](docs/core/rfc9421-en.md)
- [Smart Contract Documentation](contracts/README.md)

## Contributing

We welcome contributions to the SAGE project! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure your code:
- Follows Go best practices and conventions
- Includes appropriate tests
- Updates documentation as needed
- Passes all CI checks

## Issues

If you encounter any problems or have suggestions, please file an issue on our GitHub repository:

[https://github.com/sage-x-project/sage/issues](https://github.com/sage-x-project/sage/issues)

When reporting issues, please include:
- A clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- System information (OS, Go version, etc.)

## License

This project is licensed under the GNU General Public License v2.0 - see the [LICENSE](LICENSE) file for details.

Copyright (C) 2025 SAGE-X Project Contributors