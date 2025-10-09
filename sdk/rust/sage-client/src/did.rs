//! DID (Decentralized Identifier) module

use crate::error::{Error, Result};
use std::fmt;

/// DID (Decentralized Identifier)
///
/// Format: did:sage:<network>:<address>
/// Example: did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct Did {
    /// Full DID string
    pub did_string: String,
    /// Network (e.g., "ethereum", "kaia")
    pub network: String,
    /// Address on the network
    pub address: String,
}

impl Did {
    /// Create DID from string
    pub fn new(did_string: &str) -> Result<Self> {
        let parts: Vec<&str> = did_string.split(':').collect();

        if parts.len() != 4 || parts[0] != "did" || parts[1] != "sage" {
            return Err(Error::Validation(format!(
                "Invalid DID format: {}",
                did_string
            )));
        }

        Ok(Self {
            did_string: did_string.to_string(),
            network: parts[2].to_string(),
            address: parts[3].to_string(),
        })
    }

    /// Create DID from network and address
    pub fn from_parts(network: &str, address: &str) -> Self {
        let did_string = format!("did:sage:{}:{}", network, address);
        Self {
            did_string,
            network: network.to_string(),
            address: address.to_string(),
        }
    }

    /// Get DID as string
    pub fn as_str(&self) -> &str {
        &self.did_string
    }
}

impl fmt::Display for Did {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.did_string)
    }
}

impl From<Did> for String {
    fn from(did: Did) -> Self {
        did.did_string
    }
}

impl TryFrom<&str> for Did {
    type Error = Error;

    fn try_from(s: &str) -> Result<Self> {
        Did::new(s)
    }
}

impl TryFrom<String> for Did {
    type Error = Error;

    fn try_from(s: String) -> Result<Self> {
        Did::new(&s)
    }
}

/// DID Document containing identity information
#[derive(Debug, Clone)]
pub struct DidDocument {
    pub did: Did,
    pub public_key: Vec<u8>,     // Ed25519 public key
    pub public_kem_key: Vec<u8>, // X25519 public key
    pub owner_address: String,
    pub is_active: bool,
    pub revoked: bool,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_did_parsing() {
        let did_str = "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb";
        let did = Did::new(did_str).unwrap();

        assert_eq!(did.network, "ethereum");
        assert_eq!(did.address, "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb");
        assert_eq!(did.as_str(), did_str);
    }

    #[test]
    fn test_did_from_parts() {
        let did = Did::from_parts("ethereum", "0xAlice");
        assert_eq!(did.as_str(), "did:sage:ethereum:0xAlice");
    }

    #[test]
    fn test_invalid_did() {
        let result = Did::new("invalid:did:format");
        assert!(result.is_err());
    }
}
