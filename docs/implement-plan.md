# SAGE Implement Plan

## 1. Core Library Components
### 1.1 sage-core
Purpose: RFC-9421 implementation + cryptographic primitives
Deliverables:
- HTTP message canonicalization
- Signature generation/verification
- Header parsing and validation
- Supported algorithms: Ed25519, ECDSA

### 1.2 sage-crypto
Purpose: Key generation and secure storage

Deliverables:
- Key pair generation (Ed25519, Secp256k1)
- Key export/import (JWK, PEM formats)
- Secure Key storage interface
- Key rotation support

### 1.3 sage-did
Purpose: agent's did metadata management

Deliverables:
- DID metadata creation/parsing
- Public key extraction
- DID method support (did:eth, did:key)
- Verification method handling

## 2. Blockchain integration
### 2.1 sage-provider
Purpose: Multi-chain DID resolution

Deliverables:
- Provider interface definition
- Ethereum provider (Web3)
- Solana provider
- Caching layer (5-min TTL)

### 2.2 sage-contracts
Purpose: On-chain agent registry

Deliverables:
- AgentRegistry.sol (Ethereum)
- DID registration functions
- Public Key update mechanism
- Event emission for monitoring

## 3. How to use sage example
Purpose: Minimal working example

Deliverables:
- MCP tool exposure
- Request authentication
- Response signing
- Access control

## 4. Attack Demonstraction
### 4.1 demo/vulnerable-chat
Purpose: Show Mitm vulnerability

Scenario:
- Two agent exchanging messages
- Attacker intercepts and modifies
- No detection mechanism

### 4.2 demo/secure-chat
Purpose: Show SAGE protection

Scenario:
- Same setup with SAGE
- Attacker attempts modification
- Attack detected and rejected

## 5. Success Criteria
### 5.1 Security:
- Pass all attack scenarios in demo
### 5.2 Performance:
- < 50ms signature verification
### 5.3 Compatibility:
- Work with existing MCP tools
### 5.4 Usability:
- < 10 lines to add SAGE to agent
