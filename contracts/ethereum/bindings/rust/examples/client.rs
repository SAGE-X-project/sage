//! Example usage of SAGE contract bindings

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
