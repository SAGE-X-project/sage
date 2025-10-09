# SAGE Rust Client

High-performance Rust client library for SAGE (Secure Agent Guarantee Engine) - providing secure, decentralized identity and communication for AI agents.

## Features

- ✅ **Ed25519 Signatures**: Fast cryptographic signing and verification
- ✅ **X25519 Key Exchange**: Elliptic curve Diffie-Hellman
- ✅ **HPKE Encryption**: Hybrid Public Key Encryption for secure sessions
- ✅ **DID Support**: Decentralized identifiers for agent identity
- ✅ **Session Management**: Efficient stateful communication
- ✅ **Async/Await**: Full async support with `tokio`
- ✅ **Type Safety**: Strong typing with zero-cost abstractions
- ✅ **Performance**: Optimized for high-throughput applications

## Installation

Add to your `Cargo.toml`:

```toml
[dependencies]
sage-client = "0.1.0"
tokio = { version = "1", features = ["full"] }
```

## Quick Start

```rust
use sage_client::{Client, ClientConfig};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize client
    let config = ClientConfig::new("http://localhost:8080");
    let mut client = Client::new(config).await?;

    // Register agent
    client.register_agent("did:sage:ethereum:0xAlice", "Alice").await?;

    // Initiate handshake
    let session_id = client.handshake("did:sage:ethereum:0xServer").await?;

    // Send message
    let response = client.send_message(&session_id, b"Hello!").await?;
    println!("Response: {}", String::from_utf8_lossy(&response));

    Ok(())
}
```

## Requirements

- Rust 1.70+
- Tokio async runtime

## Documentation

### Client Configuration

```rust
use sage_client::{Client, ClientConfig};

// Basic configuration
let config = ClientConfig::new("http://localhost:8080");

// Custom configuration
let config = ClientConfig {
    base_url: "http://localhost:8080".to_string(),
    timeout_seconds: 30,
    max_sessions: 100,
};

let mut client = Client::new(config).await?;
```

### Agent Registration

```rust
// Register agent (development only)
client.register_agent(
    "did:sage:ethereum:0xAlice",
    "Alice Agent"
).await?;
```

### Secure Communication

```rust
// Get server DID
let server_did = client.get_server_did().await?;

// Initiate HPKE handshake
let session_id = client.handshake(&server_did).await?;

// Send encrypted message
let message = b"Hello, Server!";
let response = client.send_message(&session_id, message).await?;

// Send multiple messages
for i in 0..5 {
    let msg = format!("Message {}", i);
    let resp = client.send_message(&session_id, msg.as_bytes()).await?;
    println!("Response: {}", String::from_utf8_lossy(&resp));
}
```

### Health Check

```rust
let health = client.health_check().await?;
println!("Status: {}", health.status);
if let Some(sessions) = health.sessions {
    println!("Active sessions: {}", sessions.active);
}
```

### Cryptography

```rust
use sage_client::Crypto;

// Generate keypairs
let ed25519_keypair = Crypto::generate_ed25519_keypair()?;
let x25519_keypair = Crypto::generate_x25519_keypair()?;

// Sign and verify
let message = b"Important message";
let signature = Crypto::sign(message, &ed25519_keypair.private_key)?;
let is_valid = Crypto::verify(message, &signature, &ed25519_keypair.public_key)?;

// Base64 encoding
let encoded = Crypto::base64_encode(b"data");
let decoded = Crypto::base64_decode(&encoded)?;
```

### DID Management

```rust
use sage_client::Did;

// Parse DID
let did = Did::new("did:sage:ethereum:0x742d35Cc...")?;
println!("Network: {}", did.network);
println!("Address: {}", did.address);

// Create DID from parts
let did = Did::from_parts("ethereum", "0x742d35Cc...");
println!("DID: {}", did);
```

## Examples

See [`examples/`](examples/) directory:

```bash
# Run basic usage example
cargo run --example basic_usage
```

## Testing

```bash
# Run all tests
cargo test

# Run tests with output
cargo test -- --nocapture

# Run specific test
cargo test test_ed25519_keypair_generation

# Run with coverage
cargo tarpaulin --out Html
```

## Benchmarks

```bash
# Run benchmarks (requires nightly)
cargo +nightly bench
```

## Performance

Rust implementation provides significant performance advantages:

- **Ed25519 signing**: ~70,000 signatures/sec
- **Ed25519 verification**: ~25,000 verifications/sec
- **HPKE encryption**: ~50,000 operations/sec
- **Zero-copy operations**: Minimal allocations
- **Async I/O**: Non-blocking network operations

## Architecture

```
src/
├── lib.rs          # Library exports
├── client.rs       # Client API
├── crypto.rs       # Cryptography (Ed25519, X25519, HPKE)
├── did.rs          # DID parsing and types
├── session.rs      # Session management
├── types.rs        # Data types
└── error.rs        # Error types
```

## Security

- All communication is encrypted with HPKE
- Messages are signed with Ed25519
- Replay attack prevention via timestamps
- Session expiration (default: 1 hour)
- Memory-safe Rust implementation

## Error Handling

```rust
use sage_client::Error;

match client.handshake(&server_did).await {
    Ok(session_id) => println!("Session: {}", session_id),
    Err(Error::Network(e)) => eprintln!("Network error: {}", e),
    Err(Error::Session(e)) => eprintln!("Session error: {}", e),
    Err(Error::Crypto(e)) => eprintln!("Crypto error: {}", e),
    Err(e) => eprintln!("Error: {}", e),
}
```

## Building for Production

```bash
# Release build with optimizations
cargo build --release

# Strip binary
strip target/release/libsage_client.so

# Check binary size
ls -lh target/release/libsage_client.so
```

## Cross-Compilation

```bash
# Linux to Windows
cargo build --target x86_64-pc-windows-gnu --release

# Linux to macOS
cargo build --target x86_64-apple-darwin --release

# ARM64
cargo build --target aarch64-unknown-linux-gnu --release
```

## License

LGPL-3.0 - See [LICENSE](../../../LICENSE) for details

## Contributing

Contributions welcome! Please see [CONTRIBUTING.md](../../../CONTRIBUTING.md)

## Support

- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://docs.rs/sage-client
- Label: `rust-sdk`

## Changelog

### 0.1.0 (2025-10-10)

- Initial release
- Ed25519 signing and verification
- X25519 key exchange
- HPKE encryption support
- DID parsing and validation
- Session management
- Async HTTP client
- Comprehensive test coverage

## Roadmap

- [ ] WebSocket support for real-time communication
- [ ] Connection pooling
- [ ] Automatic retry logic
- [ ] Circuit breaker pattern
- [ ] Blockchain DID resolution
- [ ] Advanced performance optimizations
- [ ] WASM support

---

**Built with ⚡ by the SAGE Team**
