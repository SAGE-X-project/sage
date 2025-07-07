# Multi-Agent Secure Communication Example

This example demonstrates how SAGE enables secure communication between multiple AI agents, each with different capabilities and trust levels.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Research Agent  │────▶│   Coordinator   │◀────│ Trading Agent   │
│  (Read-only)    │     │  (Orchestrator) │     │ (Execute trades)│
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                       │                        │
         └───────────────────────┴────────────────────────┘
                               SAGE
                    (Identity & Capability Verification)
```

## Agents

### 1. Research Agent
- **DID**: `did:sage:ethereum:0x123...`
- **Capabilities**: `read_market_data`, `analyze_trends`
- **Cannot**: Execute trades or modify data

### 2. Trading Agent  
- **DID**: `did:sage:ethereum:0x456...`
- **Capabilities**: `execute_trades`, `manage_portfolio`
- **Cannot**: Access raw research data

### 3. Coordinator Agent
- **DID**: `did:sage:ethereum:0x789...`
- **Capabilities**: `orchestrate`, `delegate_tasks`
- **Role**: Coordinates between research and trading agents

## Security Features

1. **Identity Verification**: Each agent's identity is cryptographically verified
2. **Capability-Based Access**: Agents can only perform allowed operations
3. **Secure Communication**: All inter-agent messages are signed
4. **Audit Trail**: Every action is logged with verified agent identity

## Running the Example

### Start all agents:
```bash
# Terminal 1: Research Agent
cd research-agent
go run .

# Terminal 2: Trading Agent  
cd trading-agent
go run .

# Terminal 3: Coordinator
cd coordinator
go run .

# Terminal 4: Run the demo scenario
cd demo
go run .
```

## What Happens

1. Coordinator receives a trading request
2. Coordinator asks Research Agent for market analysis
3. Research Agent provides analysis (signed with its DID)
4. Coordinator validates the research and decides on action
5. Coordinator sends trade request to Trading Agent
6. Trading Agent executes the trade (after verifying Coordinator's authority)
7. All communications are cryptographically secured by SAGE

## Key Takeaways

- **Zero Trust**: Agents don't blindly trust each other
- **Capability Enforcement**: SAGE ensures agents stay within their permissions
- **Cryptographic Proof**: Every action has verifiable attribution
- **Decentralized Trust**: No central authority needed