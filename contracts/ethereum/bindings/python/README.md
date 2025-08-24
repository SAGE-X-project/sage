# Python Bindings for SAGE Contracts

## Installation

```bash
pip install -r requirements.txt
```

## Usage

```python
from web3 import Web3
from sage_contracts import SageRegistryV2

# Connect to network
w3 = Web3(Web3.HTTPProvider('https://public-en-kairos.node.kaia.io'))

# Load contract (read-only)
registry = SageRegistryV2(w3, contract_address)

# Read data
agent = registry.get_agent_by_did("did:sage:example")
print(f"Agent: {agent}")

# Write data (requires private key)
registry_with_key = SageRegistryV2(w3, contract_address, private_key=private_key)
receipt = registry_with_key.register_agent(
    did="did:sage:test",
    name="Test Agent",
    description="Test",
    endpoint="https://test.com",
    public_key="0x04...",
    capabilities='["test"]',
    signature="0x..."
)
```

## Development

```bash
# Install in development mode
pip install -e .

# Run example
python example.py
```

## Regenerate Bindings

```bash
npm run generate:python
```
