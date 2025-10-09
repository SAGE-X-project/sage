//! Basic usage example for SAGE Rust client

use sage_client::{Client, ClientConfig};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize client
    let config = ClientConfig::new("http://localhost:8080");
    let mut client = Client::new(config).await?;

    println!("SAGE Rust Client - Basic Usage Example");
    println!("======================================\n");

    // Register agent
    let client_did = "did:sage:ethereum:0xAlice";
    client.register_agent(client_did, "Alice Agent").await?;
    println!("✓ Registered: {}", client_did);

    // Check server health
    let health = client.health_check().await?;
    println!("✓ Server status: {}", health.status);

    // Get server DID
    let server_did = client.get_server_did().await?;
    println!("✓ Server DID: {}", server_did);

    // Initiate handshake
    let session_id = client.handshake(&server_did).await?;
    println!("✓ Session established: {}", session_id);

    // Send message
    let message = b"Hello, Server!";
    let response = client.send_message(&session_id, message).await?;
    println!(
        "✓ Response: {}",
        String::from_utf8_lossy(&response)
    );

    // Send another message
    let message2 = b"How are you?";
    let response2 = client.send_message(&session_id, message2).await?;
    println!(
        "✓ Response 2: {}",
        String::from_utf8_lossy(&response2)
    );

    // Show active sessions
    println!("\n✓ Active sessions: {}", client.active_sessions());

    println!("\n✓ All operations completed successfully!");

    Ok(())
}
