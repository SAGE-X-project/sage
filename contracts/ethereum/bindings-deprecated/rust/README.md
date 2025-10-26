# Rust Bindings for SAGE Contracts

## Setup

Add to your `Cargo.toml`:

```toml
[dependencies]
sage-contracts = { path = "./bindings/rust" }
ethers = "2.0"
tokio = { version = "1", features = ["full"] }
```

## Usage

```rust
use sage_contracts::client::SageClient;
use ethers::prelude::*;
use std::sync::Arc;

#[tokio::main]
async fn main() -> Result<()> {
    // Connect to network
    let provider = Provider::<Http>::try_from("https://public-en-kairos.node.kaia.io")?;
    let client = Arc::new(provider);

    // Create client
    let sage = SageClient::new(registry_address, client);

    // Read data
    let agent = sage.get_agent_by_did("did:sage:example".to_string()).await?;
    println!("Agent: {:?}", agent);

    Ok(())
}
```

## Build

```bash
cd bindings/rust
cargo build
cargo test
cargo run --example client
```

## Features

- Type-safe contract interactions
- Async/await support with Tokio
- Automatic ABI parsing with ethers-rs
- Event streaming support
- Type serialization with Serde

## Regenerate Bindings

```bash
npm run generate:rust
```
