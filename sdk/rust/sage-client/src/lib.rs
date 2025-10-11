//! SAGE Rust Client
//!
//! A Rust client library for the Secure Agent Guarantee Engine (SAGE).
//! Provides secure, decentralized identity and communication for AI agents.
//!
//! # Features
//!
//! - Ed25519 signatures for authentication
//! - X25519 key exchange
//! - HPKE encryption for secure sessions
//! - DID (Decentralized Identifier) support
//! - Async/await with tokio
//!
//! # Example
//!
//! ```no_run
//! use sage_client::{Client, ClientConfig};
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     let config = ClientConfig::new("http://localhost:8080");
//!     let mut client = Client::new(config).await?;
//!
//!     // Register agent
//!     client.register_agent("did:sage:ethereum:0xAlice", "Alice").await?;
//!
//!     // Initiate handshake
//!     let session_id = client.handshake("did:sage:ethereum:0xServer").await?;
//!
//!     // Send message
//!     let response = client.send_message(&session_id, b"Hello!").await?;
//!
//!     Ok(())
//! }
//! ```

pub mod client;
pub mod crypto;
pub mod did;
pub mod error;
pub mod session;
pub mod types;

pub use client::{Client, ClientConfig};
pub use crypto::Crypto;
pub use did::{Did, DidDocument};
pub use error::{Error, Result};
pub use session::{Session, SessionManager};
pub use types::{
    AgentMetadata, HandshakeRequest, HandshakeResponse, HealthStatus, KeyPair, Message,
    MessageRequest, MessageResponse,
};

/// Library version
pub const VERSION: &str = env!("CARGO_PKG_VERSION");
