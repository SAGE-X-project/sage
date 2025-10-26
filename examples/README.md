# SAGE Examples

This directory contains examples demonstrating how to use SAGE (Secure Agent Guarantee Engine) in various scenarios.

## Available Examples

### 1. [mcp-integration/](./mcp-integration/) - MCP Tool Integration
Shows how to add SAGE security to MCP (Model Context Protocol) tools:
- **basic-tool/** - Complete calculator tool with SAGE
- **simple/** - Minimal integration guide (just 3 lines!)
- **client/** - AI agent client library
- **vulnerable-vs-secure/** - Security demonstration
- **multi-agent/** - Agent-to-agent communication

### 2. [a2a-integration/](./a2a-integration/) - A2A Protocol Integration
Working examples of secure multi-key agent communication using A2A protocol:
- **01-register-agent/** - Register multi-key agents (ECDSA, Ed25519, X25519)
- **02-generate-card/** - Generate and export A2A Agent Cards
- **03-exchange-cards/** - Exchange and verify cards between agents
- **04-secure-message/** - Establish secure channels with HPKE encryption

### 3. policy-enforcement/ (Coming Soon)
Capability-based access control examples

### 4. blockchain-integration/ (Coming Soon)
DID registration and resolution examples

## Quick Start

### For MCP Tool Developers
```bash
cd mcp-integration/simple
go run example_tool.go
# Your tool is now SAGE-secured!
```

### For AI Agent Developers
```bash
cd mcp-integration/client
go run .
# Make secure calls to SAGE-protected tools
```

### For A2A Protocol Integration
```bash
# Start local blockchain
cd contracts/ethereum && npx hardhat node

# Deploy registry (in another terminal)
npx hardhat run scripts/deploy-v4-local.js --network localhost

# Run A2A examples
cd examples/a2a-integration/01-register-agent
go run main.go
```

### Security Demo
```bash
cd mcp-integration/vulnerable-vs-secure
# Follow the README to see attacks and defenses
```

## Key Concepts

1. **DIDs (Decentralized Identifiers)**: Every agent has a blockchain-verified identity
2. **RFC-9421 Signatures**: All messages are cryptographically signed
3. **Capability Checking**: Agents can only perform allowed operations
4. **Zero Trust**: No implicit trust between components

## Learn More

- [SAGE Documentation](../docs/)
- [DID Module](../docs/did/)
- [RFC-9421 Implementation](../docs/core/rfc9421-en.md)
- [Architecture Overview](../docs/architecture/)