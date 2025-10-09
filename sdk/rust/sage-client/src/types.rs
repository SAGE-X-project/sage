//! Type definitions for SAGE client

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Cryptographic key pair
#[derive(Debug, Clone)]
pub struct KeyPair {
    pub private_key: Vec<u8>,
    pub public_key: Vec<u8>,
    pub key_type: KeyType,
}

/// Key type enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum KeyType {
    Ed25519,
    X25519,
}

impl KeyType {
    pub fn as_str(&self) -> &'static str {
        match self {
            KeyType::Ed25519 => "Ed25519",
            KeyType::X25519 => "X25519",
        }
    }
}

/// Generic message structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Message {
    pub sender_did: String,
    pub receiver_did: String,
    pub content: Vec<u8>,
    pub timestamp: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub signature: Option<Vec<u8>>,
}

/// HPKE handshake request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HandshakeRequest {
    pub sender_did: String,
    pub receiver_did: String,
    pub message: String, // Base64-encoded encrypted payload
    pub timestamp: i64,
    pub signature: String, // Base64-encoded signature
}

/// HPKE handshake response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HandshakeResponse {
    pub session_id: String,
    pub response: String, // Base64-encoded encrypted response
}

/// Message send request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageRequest {
    pub sender_did: String,
    pub receiver_did: String,
    pub message: String, // Base64-encoded encrypted message
    pub timestamp: i64,
    pub signature: String, // Base64-encoded signature
}

/// Message send response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageResponse {
    pub response: String, // Base64-encoded encrypted response
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_id: Option<String>,
}

/// Agent metadata for registration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentMetadata {
    pub did: String,
    pub name: String,
    pub is_active: bool,
    pub public_key: String,     // Base64-encoded Ed25519 public key
    pub public_kem_key: String, // Base64-encoded X25519 public key
}

/// Session information
#[derive(Debug, Clone)]
pub struct SessionInfo {
    pub session_id: String,
    pub client_did: String,
    pub server_did: String,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub last_activity: DateTime<Utc>,
    pub metadata: HashMap<String, String>,
}

/// Server health status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthStatus {
    pub status: String,
    pub timestamp: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sessions: Option<SessionStats>,
}

/// Session statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionStats {
    pub active: i64,
    pub total: i64,
}

/// Error response from server
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorResponse {
    pub error: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub code: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<serde_json::Value>,
}

/// Server KEM public key response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KemPublicKeyResponse {
    pub kem_public_key: String, // Base64-encoded
}

/// Server DID response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerDidResponse {
    pub did: String,
}

/// Agent registration response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterResponse {
    pub message: String,
}
