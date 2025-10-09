//! SAGE client API

use crate::crypto::{setup_hpke_sender, Crypto};
use crate::did::Did;
use crate::error::{Error, Result};
use crate::session::{Session, SessionManager};
use crate::types::*;
use reqwest::Client as HttpClient;
use std::time::{SystemTime, UNIX_EPOCH};

/// Client configuration
#[derive(Debug, Clone)]
pub struct ClientConfig {
    pub base_url: String,
    pub timeout_seconds: u64,
    pub max_sessions: usize,
}

impl ClientConfig {
    /// Create new client configuration
    pub fn new(base_url: &str) -> Self {
        Self {
            base_url: base_url.trim_end_matches('/').to_string(),
            timeout_seconds: 30,
            max_sessions: 100,
        }
    }
}

/// SAGE client
pub struct Client {
    config: ClientConfig,
    http_client: HttpClient,
    identity_keypair: Option<KeyPair>,
    kem_keypair: Option<KeyPair>,
    client_did: Option<String>,
    session_manager: SessionManager,
}

impl Client {
    /// Create new SAGE client
    pub async fn new(config: ClientConfig) -> Result<Self> {
        let http_client = HttpClient::builder()
            .timeout(std::time::Duration::from_secs(config.timeout_seconds))
            .build()
            .map_err(|e| Error::Network(e))?;

        let mut client = Self {
            config: config.clone(),
            http_client,
            identity_keypair: None,
            kem_keypair: None,
            client_did: None,
            session_manager: SessionManager::new(config.max_sessions),
        };

        client.initialize().await?;
        Ok(client)
    }

    /// Initialize client with keypairs
    pub async fn initialize(&mut self) -> Result<()> {
        self.identity_keypair = Some(Crypto::generate_ed25519_keypair()?);
        self.kem_keypair = Some(Crypto::generate_x25519_keypair()?);
        Ok(())
    }

    /// Get server's KEM public key
    pub async fn get_server_kem_key(&self) -> Result<Vec<u8>> {
        let url = format!("{}/debug/kem-pub", self.config.base_url);
        let response = self
            .http_client
            .get(&url)
            .send()
            .await?
            .json::<KemPublicKeyResponse>()
            .await?;

        Crypto::base64_decode(&response.kem_public_key)
    }

    /// Get server's DID
    pub async fn get_server_did(&self) -> Result<String> {
        let url = format!("{}/debug/server-did", self.config.base_url);
        let response = self
            .http_client
            .get(&url)
            .send()
            .await?
            .json::<ServerDidResponse>()
            .await?;

        Ok(response.did)
    }

    /// Health check
    pub async fn health_check(&self) -> Result<HealthStatus> {
        let url = format!("{}/debug/health", self.config.base_url);
        let response = self
            .http_client
            .get(&url)
            .send()
            .await?
            .json::<HealthStatus>()
            .await?;

        Ok(response)
    }

    /// Register agent (development only)
    pub async fn register_agent(&mut self, did: &str, name: &str) -> Result<()> {
        let identity = self
            .identity_keypair
            .as_ref()
            .ok_or(Error::NotInitialized)?;
        let kem = self.kem_keypair.as_ref().ok_or(Error::NotInitialized)?;

        let metadata = AgentMetadata {
            did: did.to_string(),
            name: name.to_string(),
            is_active: true,
            public_key: Crypto::base64_encode(&identity.public_key),
            public_kem_key: Crypto::base64_encode(&kem.public_key),
        };

        let url = format!("{}/debug/register-agent", self.config.base_url);
        self.http_client
            .post(&url)
            .json(&metadata)
            .send()
            .await?
            .json::<RegisterResponse>()
            .await?;

        self.client_did = Some(did.to_string());
        Ok(())
    }

    /// Initiate HPKE handshake
    pub async fn handshake(&mut self, server_did: &str) -> Result<String> {
        let client_did = self
            .client_did
            .as_ref()
            .ok_or(Error::NotInitialized)?
            .clone();
        let identity = self
            .identity_keypair
            .as_ref()
            .ok_or(Error::NotInitialized)?;

        let server_kem_key = self.get_server_kem_key().await?;
        let (mut hpke_ctx, encapsulated_key) = setup_hpke_sender(&server_kem_key)?;

        let handshake_data = serde_json::json!({
            "type": "handshake",
            "client_did": client_did,
            "timestamp": current_timestamp(),
        });
        let plaintext = serde_json::to_vec(&handshake_data)?;
        let ciphertext = hpke_ctx.seal(&plaintext)?;

        let mut message = encapsulated_key.clone();
        message.extend_from_slice(&ciphertext);
        let message_b64 = Crypto::base64_encode(&message);

        let timestamp = current_timestamp();
        let to_sign = format!("{}|{}|{}|{}", client_did, server_did, message_b64, timestamp);
        let signature = Crypto::sign(to_sign.as_bytes(), &identity.private_key)?;
        let signature_b64 = Crypto::base64_encode(&signature);

        let request = HandshakeRequest {
            sender_did: client_did.clone(),
            receiver_did: server_did.to_string(),
            message: message_b64,
            timestamp,
            signature: signature_b64,
        };

        let url = format!("{}/v1/a2a:sendMessage", self.config.base_url);
        let response = self
            .http_client
            .post(&url)
            .json(&request)
            .send()
            .await?
            .json::<HandshakeResponse>()
            .await?;

        let session_id = response.session_id.clone();
        let session = Session::new(
            session_id.clone(),
            client_did,
            server_did.to_string(),
            hpke_ctx,
            3600,
        );
        self.session_manager.add_session(session)?;

        Ok(session_id)
    }

    /// Send encrypted message
    pub async fn send_message(&mut self, session_id: &str, message: &[u8]) -> Result<Vec<u8>> {
        let session = self
            .session_manager
            .get_session(session_id)
            .ok_or_else(|| Error::Session("Session not found".to_string()))?;

        let ciphertext = session.encrypt(message)?;
        let message_b64 = Crypto::base64_encode(&ciphertext);

        let client_did = self.client_did.as_ref().ok_or(Error::NotInitialized)?;
        let identity = self.identity_keypair.as_ref().ok_or(Error::NotInitialized)?;

        let timestamp = current_timestamp();
        let to_sign = format!(
            "{}|{}|{}|{}",
            client_did, session.server_did, message_b64, timestamp
        );
        let signature = Crypto::sign(to_sign.as_bytes(), &identity.private_key)?;
        let signature_b64 = Crypto::base64_encode(&signature);

        let request = MessageRequest {
            sender_did: client_did.clone(),
            receiver_did: session.server_did.clone(),
            message: message_b64,
            timestamp,
            signature: signature_b64,
        };

        let url = format!("{}/v1/a2a:sendMessage", self.config.base_url);
        let response = self
            .http_client
            .post(&url)
            .json(&request)
            .header("X-Session-ID", session_id)
            .send()
            .await?
            .json::<MessageResponse>()
            .await?;

        let response_bytes = Crypto::base64_decode(&response.response)?;
        let session = self
            .session_manager
            .get_session(session_id)
            .ok_or_else(|| Error::Session("Session not found".to_string()))?;
        session.decrypt(&response_bytes)
    }

    /// Get active session count
    pub fn active_sessions(&mut self) -> usize {
        self.session_manager.count()
    }
}

fn current_timestamp() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64
}
