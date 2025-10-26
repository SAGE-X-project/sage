#!/usr/bin/env node

/**
 * Generate Rust bindings from contract ABIs
 * Uses ethers-rs abigen macro
 */

const fs = require('fs');
const path = require('path');

// Color codes for console output
const colors = {
  reset: "\x1b[0m",
  bright: "\x1b[1m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  red: "\x1b[31m",
  cyan: "\x1b[36m"
};

function log(message, color = "reset") {
  console.log(colors[color] + message + colors.reset);
}

async function generateRustBindings() {
  try {
    log("\n🦀 Generating Rust Bindings for Smart Contracts", "cyan");
    log("=" .repeat(50), "bright");

    // Create Rust bindings directory
    const rustDir = path.join(__dirname, '..', 'bindings', 'rust');
    const srcDir = path.join(rustDir, 'src');
    
    if (!fs.existsSync(srcDir)) {
      fs.mkdirSync(srcDir, { recursive: true });
      log(" Created bindings/rust directory structure", "green");
    }

    // Contracts to generate bindings for
    const contracts = [
      {
        name: 'SageRegistryV2',
        abi: 'abi/SageRegistryV2.abi.json',
        bin: 'artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json',
        module: 'registry'
      },
      {
        name: 'SageVerificationHook',
        abi: 'abi/SageVerificationHook.abi.json',
        bin: 'artifacts/contracts/SageVerificationHook.sol/SageVerificationHook.json',
        module: 'hook'
      }
    ];

    // Copy ABI files to Rust project
    const abiDir = path.join(rustDir, 'abi');
    if (!fs.existsSync(abiDir)) {
      fs.mkdirSync(abiDir, { recursive: true });
    }

    for (const contract of contracts) {
      const abiPath = path.join(__dirname, '..', contract.abi);
      if (fs.existsSync(abiPath)) {
        const targetPath = path.join(abiDir, `${contract.name}.json`);
        fs.copyFileSync(abiPath, targetPath);
        log(` Copied ${contract.name} ABI`, "green");
      }
    }

    // Create lib.rs with abigen macros
    const libContent = `//! SAGE Smart Contract Rust Bindings
//! Generated: ${new Date().toISOString()}

pub mod contracts {
    use ethers::prelude::*;

    // Generate bindings for SageRegistryV2
    abigen!(
        SageRegistryV2,
        "./abi/SageRegistryV2.json",
        event_derives(serde::Deserialize, serde::Serialize)
    );

    // Generate bindings for SageVerificationHook
    abigen!(
        SageVerificationHook,
        "./abi/SageVerificationHook.json",
        event_derives(serde::Deserialize, serde::Serialize)
    );
}

pub use contracts::*;

/// Example client for interacting with SAGE contracts
pub mod client {
    use super::*;
    use ethers::prelude::*;
    use std::sync::Arc;

    pub struct SageClient<M: Middleware> {
        pub registry: SageRegistryV2<M>,
        pub hook: Option<SageVerificationHook<M>>,
        client: Arc<M>,
    }

    impl<M: Middleware> SageClient<M> {
        /// Create a new client with registry address
        pub fn new(registry_address: Address, client: Arc<M>) -> Self {
            let registry = SageRegistryV2::new(registry_address, client.clone());
            Self {
                registry,
                hook: None,
                client,
            }
        }

        /// Add verification hook contract
        pub fn with_hook(mut self, hook_address: Address) -> Self {
            self.hook = Some(SageVerificationHook::new(hook_address, self.client.clone()));
            self
        }

        /// Get agent by DID
        pub async fn get_agent_by_did(&self, did: String) -> Result<contracts::AgentMetadata, ContractError<M>> {
            self.registry.get_agent_by_did(did).call().await
        }

        /// Register a new agent (requires signer)
        pub async fn register_agent(
            &self,
            did: String,
            name: String,
            description: String,
            endpoint: String,
            public_key: Bytes,
            capabilities: String,
            signature: Bytes,
        ) -> Result<TransactionReceipt, ContractError<M>> {
            let tx = self.registry
                .register_agent(did, name, description, endpoint, public_key, capabilities, signature);
            
            let pending = tx.send().await?;
            let receipt = pending.await?.ok_or_else(|| {
                ContractError::ProviderError(ProviderError::CustomError("Transaction failed".into()))
            })?;
            
            Ok(receipt)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_contract_generation() {
        // This will fail to compile if the abigen! macro fails
        let _ = std::mem::size_of::<contracts::SageRegistryV2<ethers::providers::Provider<ethers::providers::Http>>>();
    }
}
`;

    fs.writeFileSync(path.join(srcDir, 'lib.rs'), libContent);
    log(" Created lib.rs", "green");

    // Create Cargo.toml
    const cargoContent = `[package]
name = "sage-contracts"
version = "1.0.0"
edition = "2021"
authors = ["SAGE Team"]
description = "Rust bindings for SAGE smart contracts"
license = "MIT"
repository = "https://github.com/sage-x-project/sage"

[dependencies]
ethers = { version = "2.0", features = ["abigen", "ws", "rustls"] }
tokio = { version = "1", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
hex = "0.4"
anyhow = "1.0"
dotenv = "0.15"

[dev-dependencies]
tokio-test = "0.4"

[features]
default = ["mainnet"]
mainnet = []
testnet = []

[[example]]
name = "client"
path = "examples/client.rs"
`;

    fs.writeFileSync(path.join(rustDir, 'Cargo.toml'), cargoContent);
    log(" Created Cargo.toml", "green");

    // Create example
    const exampleDir = path.join(rustDir, 'examples');
    if (!fs.existsSync(exampleDir)) {
      fs.mkdirSync(exampleDir, { recursive: true });
    }

    const exampleContent = `//! Example usage of SAGE contract bindings

use anyhow::Result;
use ethers::prelude::*;
use sage_contracts::client::SageClient;
use std::sync::Arc;

#[tokio::main]
async fn main() -> Result<()> {
    // Load environment variables
    dotenv::dotenv().ok();

    // Connect to Kaia testnet
    let provider = Provider::<Http>::try_from("https://public-en-kairos.node.kaia.io")?;
    let client = Arc::new(provider);

    // Contract address (replace with actual deployed address)
    let registry_address = "0x0000000000000000000000000000000000000000"
        .parse::<Address>()?;

    // Create client
    let sage_client = SageClient::new(registry_address, client.clone());

    // Get contract owner
    let owner = sage_client.registry.owner().call().await?;
    println!("Contract owner: {:?}", owner);

    // Get agent by DID
    let did = "did:sage:example".to_string();
    match sage_client.get_agent_by_did(did.clone()).await {
        Ok(agent) => {
            println!("Agent found:");
            println!("  Name: {}", agent.name);
            println!("  Active: {}", agent.active);
            println!("  Endpoint: {}", agent.endpoint);
        }
        Err(e) => {
            println!("Agent not found: {:?}", e);
        }
    }

    // For write operations, you need a signer
    // Example with private key (DO NOT hardcode in production!)
    /*
    let wallet = "YOUR_PRIVATE_KEY"
        .parse::<LocalWallet>()?
        .with_chain_id(1001u64); // Kairos chain ID

    let client = SignerMiddleware::new(provider, wallet);
    let client = Arc::new(client);
    
    let sage_client = SageClient::new(registry_address, client);
    
    // Register a new agent
    let receipt = sage_client.register_agent(
        "did:sage:test".to_string(),
        "Test Agent".to_string(),
        "Test Description".to_string(),
        "https://test.example.com".to_string(),
        Bytes::from(vec![0x04; 65]), // Example public key
        r#"["test", "example"]"#.to_string(),
        Bytes::from(vec![0x00; 65]), // Example signature
    ).await?;
    
    println!("Agent registered! Tx: {:?}", receipt.transaction_hash);
    */

    Ok(())
}
`;

    fs.writeFileSync(path.join(exampleDir, 'client.rs'), exampleContent);
    log(" Created client.rs example", "green");

    // Create build.rs for automatic binding generation
    const buildContent = `use std::env;
use std::path::PathBuf;

fn main() {
    // Re-run build script if ABI files change
    println!("cargo:rerun-if-changed=abi/");
    
    // Set ABI directory path for runtime
    let abi_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap())
        .join("abi");
    println!("cargo:rustc-env=ABI_DIR={}", abi_dir.display());
}
`;

    fs.writeFileSync(path.join(rustDir, 'build.rs'), buildContent);
    log(" Created build.rs", "green");

    // Create .gitignore
    const gitignoreContent = `/target
**/*.rs.bk
Cargo.lock
`;

    fs.writeFileSync(path.join(rustDir, '.gitignore'), gitignoreContent);
    log(" Created .gitignore", "green");

    // Create README
    const readmeContent = `# Rust Bindings for SAGE Contracts

## Setup

Add to your \`Cargo.toml\`:

\`\`\`toml
[dependencies]
sage-contracts = { path = "./bindings/rust" }
ethers = "2.0"
tokio = { version = "1", features = ["full"] }
\`\`\`

## Usage

\`\`\`rust
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
\`\`\`

## Build

\`\`\`bash
cd bindings/rust
cargo build
cargo test
cargo run --example client
\`\`\`

## Features

- Type-safe contract interactions
- Async/await support with Tokio
- Automatic ABI parsing with ethers-rs
- Event streaming support
- Type serialization with Serde

## Regenerate Bindings

\`\`\`bash
npm run generate:rust
\`\`\`
`;

    fs.writeFileSync(path.join(rustDir, 'README.md'), readmeContent);
    log(" Created README.md", "green");

    log("\n" + "=".repeat(50), "bright");
    log(" Rust binding generation complete!", "green");
    log("\nGenerated files:", "yellow");
    log(`  📁 ${rustDir}/`, "cyan");
    log("  📄 src/lib.rs - Main library with contract bindings", "cyan");
    log("  📄 Cargo.toml - Rust package configuration", "cyan");
    log("  📄 examples/client.rs - Usage example", "cyan");
    log("  📄 build.rs - Build script", "cyan");
    log("  📄 abi/ - Contract ABI files", "cyan");

    log("\n To use the Rust bindings:", "yellow");
    log("  cd bindings/rust", "cyan");
    log("  cargo build", "cyan");
    log("  cargo run --example client", "cyan");

  } catch (error) {
    log("\n Error generating Rust bindings:", "red");
    console.error(error);
    process.exit(1);
  }
}

// Run generation
generateRustBindings();