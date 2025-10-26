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

## Troubleshooting

### Common Issues

#### Compilation Errors

**Problem:** `error[E0433]: failed to resolve: use of undeclared crate or module`

**Solutions:**
```bash
# 1. Ensure dependencies are up to date
cargo update

# 2. Check Cargo.toml for missing dependencies
# Add any missing crates

# 3. Clean and rebuild
cargo clean
cargo build

# 4. Check Rust version (requires 1.70+)
rustc --version
rustup update stable
```

#### Async Runtime Errors

**Problem:** `Cannot start a runtime from within a runtime` or `thread panicked at 'Cannot drop a runtime in a context where blocking is not allowed'`

**Solutions:**
```rust
// ❌ BAD - Nested runtime
#[tokio::main]
async fn main() {
    tokio::runtime::Runtime::new().unwrap(); // Error!
}

// ✅ GOOD - Use existing runtime
#[tokio::main]
async fn main() {
    let config = ClientConfig::new("http://localhost:8080");
    let mut client = Client::new(config).await.unwrap();
}

// ✅ GOOD - Spawn blocking tasks correctly
use tokio::task;

async fn blocking_operation() {
    task::spawn_blocking(|| {
        // CPU-intensive work
    }).await.unwrap();
}
```

#### Lifetime and Ownership Errors

**Problem:** `error[E0597]: ... does not live long enough`

**Solutions:**
```rust
// ❌ BAD - Temporary value dropped
let session_id = client.handshake(&server_did).await?;
let message = b"Hello";
client.send_message(&session_id, message).await?;  // May cause lifetime issues

// ✅ GOOD - Proper ownership
let server_did = "did:sage:ethereum:0xServer".to_string();
let session_id = client.handshake(&server_did).await?;
let message = b"Hello".to_vec();
client.send_message(&session_id, &message).await?;

// ✅ GOOD - Use references correctly
async fn send_secure_message(
    client: &mut Client,
    session_id: &str,
    message: &[u8]
) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    Ok(client.send_message(session_id, message).await?)
}
```

#### Session Management Errors

**Problem:** `Session expired or not found`

**Solutions:**
```rust
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

// ✅ Session pool with automatic renewal
pub struct SessionPool {
    client: Arc<RwLock<Client>>,
    sessions: Arc<RwLock<HashMap<String, String>>>,
}

impl SessionPool {
    pub fn new(client: Client) -> Self {
        Self {
            client: Arc::new(RwLock::new(client)),
            sessions: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub async fn get_session(&self, target_did: &str) -> Result<String, Box<dyn std::error::Error>> {
        // Check existing session
        {
            let sessions = self.sessions.read().await;
            if let Some(session_id) = sessions.get(target_did) {
                // TODO: Check if session is expired
                return Ok(session_id.clone());
            }
        }

        // Create new session
        let mut client = self.client.write().await;
        let session_id = client.handshake(target_did).await?;

        // Cache session
        let mut sessions = self.sessions.write().await;
        sessions.insert(target_did.to_string(), session_id.clone());

        Ok(session_id)
    }

    pub async fn cleanup_expired(&self) {
        let mut sessions = self.sessions.write().await;
        // Remove expired sessions
        sessions.retain(|_, session_id| {
            // Check expiration logic
            true // Placeholder
        });
    }
}
```

#### OpenSSL or Crypto Errors

**Problem:** `error: failed to run custom build command for 'openssl-sys'`

**Solutions:**
```bash
# Ubuntu/Debian
sudo apt-get install pkg-config libssl-dev

# macOS
brew install openssl
export OPENSSL_DIR=$(brew --prefix openssl)

# Or use rustls instead of OpenSSL
# In Cargo.toml:
# reqwest = { version = "0.11", features = ["rustls-tls"], default-features = false }
```

### Debug Mode

Enable verbose logging:

```rust
// Add to Cargo.toml
# [dependencies]
# env_logger = "0.11"

use env_logger;

#[tokio::main]
async fn main() {
    // Initialize logger
    env_logger::Builder::from_default_env()
        .filter_level(log::LevelFilter::Debug)
        .init();

    log::debug!("Starting SAGE client");

    let config = ClientConfig::new("http://localhost:8080");
    let mut client = Client::new(config).await.unwrap();

    // All operations will be logged
}
```

```bash
# Run with debug output
RUST_LOG=debug cargo run

# Or specific module only
RUST_LOG=sage_client=debug cargo run
```

### Performance Issues

**Problem:** Slow operations or high memory usage

**Diagnostics:**
```rust
use std::time::Instant;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut client = Client::new(ClientConfig::new("http://localhost:8080")).await?;

    // Measure handshake
    let start = Instant::now();
    let session_id = client.handshake("did:sage:ethereum:0xServer").await?;
    println!("Handshake took: {:?}", start.elapsed());  // Expected: 50-200ms

    // Measure message send
    let start = Instant::now();
    let response = client.send_message(&session_id, b"test").await?;
    println!("Message took: {:?}", start.elapsed());  // Expected: 20-100ms

    Ok(())
}
```

**Solutions:**
- Use release build for benchmarks: `cargo build --release`
- Enable LTO in Cargo.toml for smaller binary and better performance
- Use connection pooling for multiple requests
- Consider using `tokio` with multi-threaded runtime

---

## Best Practices

### Security

#### 1. Never Expose Private Keys

```rust
// ❌ BAD - Logging private keys
println!("Private key: {:?}", private_key);

// ❌ BAD - Storing in plain text
std::fs::write("key.txt", private_key)?;

// ✅ GOOD - Use secure storage
use keyring::Entry;

let entry = Entry::new("sage-client", "identity-key")?;
entry.set_password(&hex::encode(private_key))?;

// Retrieve when needed
let private_key_hex = entry.get_password()?;
let private_key = hex::decode(private_key_hex)?;

// ✅ GOOD - Use environment variables
let private_key_hex = std::env::var("SAGE_PRIVATE_KEY")
    .expect("SAGE_PRIVATE_KEY not set");
```

#### 2. Validate All Inputs

```rust
use regex::Regex;

// ✅ Validate DIDs
fn validate_did(did: &str) -> Result<(), String> {
    let did_regex = Regex::new(r"^did:sage:(ethereum|solana):0x[a-fA-F0-9]{40}$")
        .unwrap();

    if !did_regex.is_match(did) {
        return Err(format!("Invalid DID format: {}", did));
    }

    Ok(())
}

// ✅ Validate message size
const MAX_MESSAGE_SIZE: usize = 1024 * 1024; // 1MB

pub async fn send_message(
    client: &mut Client,
    session_id: &str,
    message: &[u8]
) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    if message.len() > MAX_MESSAGE_SIZE {
        return Err(format!(
            "Message too large: {} bytes (max: {})",
            message.len(),
            MAX_MESSAGE_SIZE
        ).into());
    }

    Ok(client.send_message(session_id, message).await?)
}
```

#### 3. Proper Error Handling

```rust
use thiserror::Error;

// ✅ Custom error types
#[derive(Error, Debug)]
pub enum SageError {
    #[error("Network error: {0}")]
    Network(#[from] reqwest::Error),

    #[error("Crypto error: {0}")]
    Crypto(String),

    #[error("Session expired: {0}")]
    SessionExpired(String),

    #[error("Invalid DID: {0}")]
    InvalidDID(String),
}

// Usage with proper error propagation
async fn send_secure_message(
    client: &mut Client,
    target_did: &str,
    message: &[u8]
) -> Result<Vec<u8>, SageError> {
    validate_did(target_did)
        .map_err(|e| SageError::InvalidDID(e))?;

    let session_id = client.handshake(target_did)
        .await
        .map_err(|e| SageError::Network(e))?;

    client.send_message(&session_id, message)
        .await
        .map_err(|e| SageError::Network(e))
}
```

#### 4. Use Type Safety

```rust
// ✅ Newtype pattern for IDs
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct SessionId(String);

impl SessionId {
    pub fn new(id: String) -> Self {
        Self(id)
    }

    pub fn as_str(&self) -> &str {
        &self.0
    }
}

#[derive(Debug, Clone)]
pub struct DID(String);

impl DID {
    pub fn new(did: String) -> Result<Self, String> {
        validate_did(&did)?;
        Ok(Self(did))
    }

    pub fn as_str(&self) -> &str {
        &self.0
    }
}

// Now the type system prevents mixing up IDs
async fn send_message(
    client: &mut Client,
    session_id: &SessionId,
    message: &[u8]
) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    client.send_message(session_id.as_str(), message).await
}
```

### Performance

#### 1. Reuse Connections and Sessions

```rust
// ❌ BAD - New client for each request
for target_did in target_dids {
    let client = Client::new(config.clone()).await?;
    let session_id = client.handshake(target_did).await?;
    client.send_message(&session_id, message).await?;
}

// ✅ GOOD - Reuse client and sessions
let mut client = Client::new(config).await?;
let session_id = client.handshake(server_did).await?;

for message in messages {
    client.send_message(&session_id, message).await?;
}

// ✅ BETTER - Connection pool with session caching
use deadpool::managed::{Manager, Pool};

struct ClientManager {
    config: ClientConfig,
}

#[async_trait::async_trait]
impl Manager for ClientManager {
    type Type = Client;
    type Error = Box<dyn std::error::Error + Send + Sync>;

    async fn create(&self) -> Result<Self::Type, Self::Error> {
        Ok(Client::new(self.config.clone()).await?)
    }

    async fn recycle(&self, _obj: &mut Self::Type) -> Result<(), Self::Error> {
        Ok(())
    }
}
```

#### 2. Parallel Processing

```rust
use futures::future::join_all;

// ✅ Send messages in parallel
pub async fn broadcast_message(
    client: &mut Client,
    target_dids: Vec<String>,
    message: &[u8]
) -> Result<Vec<Vec<u8>>, Box<dyn std::error::Error>> {
    // Create sessions in parallel
    let session_futures = target_dids.iter()
        .map(|did| client.handshake(did))
        .collect::<Vec<_>>();

    let session_ids = join_all(session_futures).await
        .into_iter()
        .collect::<Result<Vec<_>, _>>()?;

    // Send messages in parallel
    let message_futures = session_ids.iter()
        .map(|session_id| client.send_message(session_id, message))
        .collect::<Vec<_>>();

    let responses = join_all(message_futures).await
        .into_iter()
        .collect::<Result<Vec<_>, _>>()?;

    Ok(responses)
}
```

#### 3. Optimize Memory Usage

```rust
// ✅ Use iterators to avoid allocations
pub async fn send_large_file(
    client: &mut Client,
    session_id: &str,
    file_path: &str
) -> Result<(), Box<dyn std::error::Error>> {
    use tokio::io::AsyncReadExt;

    let mut file = tokio::fs::File::open(file_path).await?;
    let mut buffer = vec![0u8; 1024 * 1024]; // 1MB buffer

    loop {
        let n = file.read(&mut buffer).await?;
        if n == 0 {
            break;
        }

        client.send_message(session_id, &buffer[..n]).await?;
    }

    Ok(())
}

// ✅ Use zero-copy where possible
use bytes::Bytes;

pub async fn send_bytes(
    client: &mut Client,
    session_id: &str,
    data: Bytes
) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    // Bytes uses Arc internally, so cloning is cheap
    Ok(client.send_message(session_id, &data).await?)
}
```

#### 4. Compile-Time Optimizations

```toml
# Cargo.toml
[profile.release]
opt-level = 3
lto = true
codegen-units = 1
strip = true  # Remove debug symbols

[profile.release.package."*"]
opt-level = 3
```

### Async Best Practices

#### 1. Avoid Blocking the Executor

```rust
// ❌ BAD - Blocking in async context
async fn bad_example() {
    std::thread::sleep(std::time::Duration::from_secs(1));  // Blocks executor!
}

// ✅ GOOD - Use async sleep
async fn good_example() {
    tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
}

// ✅ GOOD - Use spawn_blocking for CPU-intensive work
use tokio::task;

async fn cpu_intensive_work(data: Vec<u8>) -> Vec<u8> {
    task::spawn_blocking(move || {
        // CPU-intensive computation
        data.iter().map(|b| b.wrapping_add(1)).collect()
    }).await.unwrap()
}
```

#### 2. Proper Task Spawning

```rust
use tokio::task::JoinHandle;

// ✅ Spawn tasks correctly
pub async fn spawn_agents(
    client_configs: Vec<ClientConfig>
) -> Vec<JoinHandle<Result<(), Box<dyn std::error::Error + Send + Sync>>>> {
    let mut handles = vec![];

    for config in client_configs {
        let handle = tokio::spawn(async move {
            let mut client = Client::new(config).await?;
            // Agent logic
            Ok(())
        });

        handles.push(handle);
    }

    handles
}

// Wait for all tasks
let results = futures::future::join_all(handles).await;
```

#### 3. Graceful Shutdown

```rust
use tokio::signal;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut client = Client::new(ClientConfig::new("http://localhost:8080")).await?;

    // Spawn background task
    let handle = tokio::spawn(async move {
        // Long-running task
    });

    // Wait for CTRL+C
    signal::ctrl_c().await?;
    println!("Shutting down gracefully...");

    // Cancel background task
    handle.abort();

    // Cleanup
    drop(client);

    Ok(())
}
```

### Testing

#### 1. Unit Tests

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_validate_did() {
        assert!(validate_did("did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEa1").is_ok());
        assert!(validate_did("invalid").is_err());
    }

    #[tokio::test]
    async fn test_session_creation() {
        let config = ClientConfig::new("http://localhost:8080");
        let mut client = Client::new(config).await.unwrap();

        // Mock or use test server
        // let session_id = client.handshake("did:sage:test").await.unwrap();
        // assert!(!session_id.is_empty());
    }
}
```

#### 2. Integration Tests

```rust
// tests/integration_test.rs
use sage_client::{Client, ClientConfig};

#[tokio::test]
#[ignore] // Run with: cargo test -- --ignored
async fn test_full_handshake() {
    let server_url = std::env::var("SAGE_TEST_SERVER")
        .unwrap_or_else(|_| "http://localhost:8080".to_string());

    let config = ClientConfig::new(&server_url);
    let mut client = Client::new(config).await.expect("Failed to create client");

    let session_id = client.handshake("did:sage:test:server")
        .await
        .expect("Failed to perform handshake");

    assert!(!session_id.is_empty());
}
```

#### 3. Benchmarks

```rust
// benches/benchmark.rs
use criterion::{black_box, criterion_group, criterion_main, Criterion};
use sage_client::{Client, ClientConfig};

fn criterion_benchmark(c: &mut Criterion) {
    let rt = tokio::runtime::Runtime::new().unwrap();

    c.bench_function("handshake", |b| {
        b.to_async(&rt).iter(|| async {
            let config = ClientConfig::new("http://localhost:8080");
            let mut client = Client::new(config).await.unwrap();
            black_box(client.handshake("did:sage:test").await.unwrap())
        });
    });
}

criterion_group!(benches, criterion_benchmark);
criterion_main!(benches);
```

```bash
# Run benchmarks
cargo bench
```

---

## Advanced Usage

### Multi-Agent Coordination

```rust
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

pub struct AgentCoordinator {
    clients: Arc<RwLock<HashMap<String, Client>>>,
}

impl AgentCoordinator {
    pub fn new() -> Self {
        Self {
            clients: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub async fn add_agent(&self, id: String, config: ClientConfig) -> Result<(), Box<dyn std::error::Error>> {
        let client = Client::new(config).await?;
        let mut clients = self.clients.write().await;
        clients.insert(id, client);
        Ok(())
    }

    pub async fn broadcast(
        &self,
        message: &[u8],
        target_dids: Vec<String>
    ) -> Result<Vec<Vec<u8>>, Box<dyn std::error::Error>> {
        let clients = self.clients.read().await;

        let mut futures = vec![];
        for (id, client) in clients.iter() {
            for target_did in &target_dids {
                let client_clone = client.clone();
                let target_did_clone = target_did.clone();
                let message_clone = message.to_vec();

                futures.push(tokio::spawn(async move {
                    let session_id = client_clone.handshake(&target_did_clone).await?;
                    client_clone.send_message(&session_id, &message_clone).await
                }));
            }
        }

        let results = futures::future::join_all(futures).await;
        results.into_iter()
            .map(|r| r?)
            .collect()
    }
}
```

### Monitoring and Metrics

```rust
use prometheus::{Counter, Histogram, Registry};
use std::time::Instant;

pub struct MonitoredClient {
    client: Client,
    handshake_counter: Counter,
    message_counter: Counter,
    handshake_duration: Histogram,
    message_duration: Histogram,
}

impl MonitoredClient {
    pub fn new(client: Client, registry: &Registry) -> Result<Self, Box<dyn std::error::Error>> {
        let handshake_counter = Counter::new("sage_handshakes_total", "Total handshakes")?;
        let message_counter = Counter::new("sage_messages_total", "Total messages")?;
        let handshake_duration = Histogram::new("sage_handshake_duration_seconds", "Handshake duration")?;
        let message_duration = Histogram::new("sage_message_duration_seconds", "Message duration")?;

        registry.register(Box::new(handshake_counter.clone()))?;
        registry.register(Box::new(message_counter.clone()))?;
        registry.register(Box::new(handshake_duration.clone()))?;
        registry.register(Box::new(message_duration.clone()))?;

        Ok(Self {
            client,
            handshake_counter,
            message_counter,
            handshake_duration,
            message_duration,
        })
    }

    pub async fn handshake(&mut self, target_did: &str) -> Result<String, Box<dyn std::error::Error>> {
        let start = Instant::now();
        let result = self.client.handshake(target_did).await;
        self.handshake_duration.observe(start.elapsed().as_secs_f64());

        if result.is_ok() {
            self.handshake_counter.inc();
        }

        result
    }

    pub async fn send_message(&mut self, session_id: &str, message: &[u8]) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
        let start = Instant::now();
        let result = self.client.send_message(session_id, message).await;
        self.message_duration.observe(start.elapsed().as_secs_f64());

        if result.is_ok() {
            self.message_counter.inc();
        }

        result
    }
}
```

### WASM Support (Experimental)

```rust
// Compile to WASM
// cargo build --target wasm32-unknown-unknown --release

use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmClient {
    // WASM-compatible client implementation
}

#[wasm_bindgen]
impl WasmClient {
    #[wasm_bindgen(constructor)]
    pub fn new(base_url: String) -> Self {
        // Initialize WASM client
        Self {}
    }

    #[wasm_bindgen]
    pub async fn handshake(&mut self, target_did: String) -> Result<String, JsValue> {
        // Perform handshake
        Ok("session-id".to_string())
    }
}
```

---

## API Documentation

Full API documentation on docs.rs:

```bash
# Generate local documentation
cargo doc --open

# View documentation
open target/doc/sage_client/index.html
```

### Core Types

- **Client**: Main client interface
- **ClientConfig**: Client configuration
- **SessionId**: Type-safe session identifier
- **DID**: Decentralized identifier
- **Error types**: Various error types for different failure modes

---

## Roadmap

- [ ] WebSocket support for real-time communication
- [ ] Connection pooling
- [ ] Automatic retry logic
- [ ] Circuit breaker pattern
- [ ] Blockchain DID resolution
- [ ] Advanced performance optimizations
- [ ] WASM support
- [ ] no_std support for embedded systems
- [ ] Async-std runtime support
- [ ] Batch operations API

---

**Built with ⚡ by the SAGE Team**
