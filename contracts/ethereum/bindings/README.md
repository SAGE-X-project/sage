#  Multi-Language Contract Bindings for SAGE

This directory contains auto-generated contract bindings for multiple programming languages, enabling seamless integration with SAGE smart contracts across different platforms.

##  Available Bindings

###  Go
- **Location**: `./go/`
- **Requirements**: Go 1.21+, abigen tool
- **Install abigen**: `go install github.com/ethereum/go-ethereum/cmd/abigen@latest`
- **Usage**: Native Go types with go-ethereum

### ☕ Java
- **Location**: `./java/`
- **Requirements**: Java 11+, web3j CLI
- **Install web3j**: `curl -L https://get.web3j.io | sh`
- **Build**: Maven or Gradle
- **Usage**: Type-safe Java classes with Web3j

### 🐍 Python
- **Location**: `./python/`
- **Requirements**: Python 3.8+, web3.py
- **Install**: `pip install -r requirements.txt`
- **Usage**: Dynamic Python classes with web3.py

### 🦀 Rust
- **Location**: `./rust/`
- **Requirements**: Rust 1.70+, Cargo
- **Build**: `cargo build`
- **Usage**: Type-safe Rust structs with ethers-rs

##  Quick Start

### Generate All Bindings
```bash
# Extract ABIs and generate all language bindings
npm run generate:all

# Or generate individually
npm run generate:go      # Go bindings
npm run generate:java    # Java bindings
npm run generate:python  # Python bindings
npm run generate:rust    # Rust bindings
```

### Prerequisites Installation

#### Go (abigen)
```bash
# Install abigen
go install github.com/ethereum/go-ethereum/cmd/abigen@latest

# Or with Homebrew (macOS)
brew install ethereum
```

#### Java (web3j)
```bash
# macOS
brew tap web3j/web3j && brew install web3j

# Linux/Windows
curl -L https://get.web3j.io | sh
```

#### Python
```bash
# No special tools needed, just Python packages
pip install web3 eth-account
```

#### Rust
```bash
# Install Rust if not already installed
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

##  Usage Examples

### Go
```go
import registry "./bindings/go/registry"

client, _ := ethclient.Dial("https://public-en-kairos.node.kaia.io")
contract, _ := registry.NewSageRegistryV2(address, client)
agent, _ := contract.GetAgentByDID(&bind.CallOpts{}, "did:sage:example")
```

### Java
```java
import io.sage.contracts.SageRegistryV2;

Web3j web3j = Web3j.build(new HttpService("https://public-en-kairos.node.kaia.io"));
SageRegistryV2 contract = SageRegistryV2.load(address, web3j, credentials, gasProvider);
var agent = contract.getAgentByDID("did:sage:example").send();
```

### Python
```python
from sage_contracts import SageRegistryV2

w3 = Web3(Web3.HTTPProvider('https://public-en-kairos.node.kaia.io'))
registry = SageRegistryV2(w3, contract_address)
agent = registry.get_agent_by_did("did:sage:example")
```

### Rust
```rust
use sage_contracts::client::SageClient;

let provider = Provider::<Http>::try_from("https://public-en-kairos.node.kaia.io")?;
let client = Arc::new(provider);
let sage = SageClient::new(registry_address, client);
let agent = sage.get_agent_by_did("did:sage:example".to_string()).await?;
```

## 📁 Directory Structure

```
bindings/
├── go/
│   ├── registry.go           # SageRegistryV2 bindings
│   ├── hook.go              # SageVerificationHook bindings
│   ├── go.mod               # Go module file
│   └── example.go           # Usage example
│
├── java/
│   ├── src/main/java/io/sage/contracts/
│   │   ├── SageRegistryV2.java
│   │   └── SageVerificationHook.java
│   ├── pom.xml              # Maven config
│   └── build.gradle         # Gradle config
│
├── python/
│   ├── sage_contracts/
│   │   ├── registry.py      # SageRegistryV2 class
│   │   ├── hook.py         # SageVerificationHook class
│   │   └── base.py         # Base contract class
│   ├── requirements.txt     # Python dependencies
│   └── setup.py            # Package setup
│
└── rust/
    ├── src/
    │   └── lib.rs          # Rust library with bindings
    ├── Cargo.toml          # Rust package config
    └── examples/
        └── client.rs       # Usage example
```

## 🔄 Regeneration

When contracts are updated:

1. **Recompile contracts**:
   ```bash
   npm run compile
   ```

2. **Extract new ABIs**:
   ```bash
   npm run extract-abi
   ```

3. **Regenerate bindings**:
   ```bash
   npm run generate:all
   ```

## 🛠 Supported Contract Methods

### Read Methods
- `owner()` - Get contract owner
- `getAgent(bytes32)` - Get agent by ID
- `getAgentByDID(string)` - Get agent by DID
- `getAgentsByOwner(address)` - List agents by owner
- `isKeyValid(bytes)` - Check key validity

### Write Methods
- `registerAgent(...)` - Register new agent
- `updateAgent(...)` - Update agent metadata
- `deactivateAgent(bytes32)` - Deactivate agent
- `revokeKey(bytes)` - Revoke public key

### Events
- `AgentRegistered` - New agent registered
- `AgentUpdated` - Agent metadata updated
- `AgentDeactivated` - Agent deactivated
- `KeyValidated` - Key validated
- `KeyRevoked` - Key revoked

## 📚 Documentation

Each language binding has its own README with specific setup and usage instructions:
- [Go Documentation](./go/README.md)
- [Java Documentation](./java/README.md)
- [Python Documentation](./python/README.md)
- [Rust Documentation](./rust/README.md)

## 🔗 Network Configuration

### Kaia Testnet (Kairos)
- **RPC**: https://public-en-kairos.node.kaia.io
- **Chain ID**: 1001
- **Explorer**: https://kairos.kaiascan.io

### Kaia Mainnet
- **RPC**: https://public-en.node.kaia.io
- **Chain ID**: 8217
- **Explorer**: https://kaiascan.io

## 📄 License

MIT - See [LICENSE](../LICENSE) for details.