//! Error types for SAGE client

use thiserror::Error;

/// Result type alias for SAGE operations
pub type Result<T> = std::result::Result<T, Error>;

/// SAGE client errors
#[derive(Error, Debug)]
pub enum Error {
    /// Cryptographic operation failed
    #[error("Crypto error: {0}")]
    Crypto(String),

    /// Session-related error
    #[error("Session error: {0}")]
    Session(String),

    /// Network/HTTP error
    #[error("Network error: {0}")]
    Network(#[from] reqwest::Error),

    /// DID-related error
    #[error("DID error: {0}")]
    Did(String),

    /// Validation error
    #[error("Validation error: {0}")]
    Validation(String),

    /// Signature verification failed
    #[error("Signature verification failed")]
    SignatureVerification,

    /// Session expired
    #[error("Session expired: {0}")]
    SessionExpired(String),

    /// Encryption/decryption failed
    #[error("Encryption error: {0}")]
    Encryption(String),

    /// Decryption failed
    #[error("Decryption error: {0}")]
    Decryption(String),

    /// Serialization error
    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    /// Base64 decode error
    #[error("Base64 decode error: {0}")]
    Base64Decode(#[from] base64::DecodeError),

    /// Client not initialized
    #[error("Client not initialized")]
    NotInitialized,

    /// Generic error
    #[error("{0}")]
    Other(String),
}

impl From<String> for Error {
    fn from(s: String) -> Self {
        Error::Other(s)
    }
}

impl From<&str> for Error {
    fn from(s: &str) -> Self {
        Error::Other(s.to_string())
    }
}
