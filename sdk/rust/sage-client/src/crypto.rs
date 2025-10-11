//! Cryptography module for SAGE client

use crate::error::{Error, Result};
use crate::types::{KeyPair, KeyType};
use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use base64::{engine::general_purpose, Engine as _};
use ed25519_dalek::{Keypair, PublicKey as Ed25519PublicKey, Signature, Verifier};
use hkdf::Hkdf;
use rand::rngs::OsRng;
use sha2::{Digest, Sha256};
use x25519_dalek::{EphemeralSecret, PublicKey, StaticSecret};

/// Cryptographic operations
pub struct Crypto;

impl Crypto {
    /// Generate Ed25519 keypair for signing
    pub fn generate_ed25519_keypair() -> Result<KeyPair> {
        let keypair = Keypair::generate(&mut OsRng);

        Ok(KeyPair {
            private_key: keypair.secret.to_bytes().to_vec(),
            public_key: keypair.public.to_bytes().to_vec(),
            key_type: KeyType::Ed25519,
        })
    }

    /// Generate X25519 keypair for key exchange
    pub fn generate_x25519_keypair() -> Result<KeyPair> {
        let secret = StaticSecret::new(OsRng);
        let public = PublicKey::from(&secret);

        Ok(KeyPair {
            private_key: secret.to_bytes().to_vec(),
            public_key: public.to_bytes().to_vec(),
            key_type: KeyType::X25519,
        })
    }

    /// Sign message with Ed25519 private key
    pub fn sign(message: &[u8], private_key: &[u8]) -> Result<Vec<u8>> {
        let key_bytes: [u8; 32] = private_key
            .try_into()
            .map_err(|_| Error::Crypto("Invalid private key length".to_string()))?;

        use ed25519_dalek::{SecretKey, ExpandedSecretKey};
        let secret = SecretKey::from_bytes(&key_bytes)
            .map_err(|e| Error::Crypto(format!("Invalid secret key: {}", e)))?;
        let expanded = ExpandedSecretKey::from(&secret);
        let public = Ed25519PublicKey::from(&secret);
        let signature = expanded.sign(message, &public);

        Ok(signature.to_bytes().to_vec())
    }

    /// Verify Ed25519 signature
    pub fn verify(message: &[u8], signature: &[u8], public_key: &[u8]) -> Result<bool> {
        let key_bytes: [u8; 32] = public_key
            .try_into()
            .map_err(|_| Error::Crypto("Invalid public key length".to_string()))?;

        let public = Ed25519PublicKey::from_bytes(&key_bytes)
            .map_err(|e| Error::Crypto(format!("Invalid public key: {}", e)))?;

        let sig_bytes: [u8; 64] = signature
            .try_into()
            .map_err(|_| Error::Crypto("Invalid signature length".to_string()))?;

        let signature = Signature::from(sig_bytes);

        public
            .verify(message, &signature)
            .map(|_| true)
            .map_err(|_| Error::SignatureVerification)
    }

    /// Compute X25519 Diffie-Hellman shared secret
    pub fn compute_dh(private_key: &[u8], public_key: &[u8]) -> Result<Vec<u8>> {
        let secret_bytes: [u8; 32] = private_key
            .try_into()
            .map_err(|_| Error::Crypto("Invalid private key length".to_string()))?;

        let public_bytes: [u8; 32] = public_key
            .try_into()
            .map_err(|_| Error::Crypto("Invalid public key length".to_string()))?;

        let secret = StaticSecret::from(secret_bytes);
        let public = PublicKey::from(public_bytes);

        let shared_secret = secret.diffie_hellman(&public);

        Ok(shared_secret.as_bytes().to_vec())
    }

    /// Derive key from shared secret using HKDF
    pub fn derive_key(shared_secret: &[u8], info: &[u8], length: usize) -> Result<Vec<u8>> {
        let hkdf = Hkdf::<Sha256>::new(None, shared_secret);
        let mut key = vec![0u8; length];
        hkdf.expand(info, &mut key)
            .map_err(|e| Error::Crypto(format!("HKDF failed: {}", e)))?;

        Ok(key)
    }

    /// Encrypt with AES-256-GCM
    pub fn encrypt_aes_gcm(plaintext: &[u8], key: &[u8], nonce_bytes: &[u8]) -> Result<Vec<u8>> {
        let key_array: [u8; 32] = key
            .try_into()
            .map_err(|_| Error::Crypto("Invalid key length".to_string()))?;

        let cipher = Aes256Gcm::new(&key_array.into());

        let nonce_array: [u8; 12] = nonce_bytes
            .try_into()
            .map_err(|_| Error::Crypto("Invalid nonce length".to_string()))?;

        let nonce = Nonce::from_slice(&nonce_array);

        let ciphertext = cipher
            .encrypt(nonce, plaintext)
            .map_err(|e| Error::Encryption(e.to_string()))?;

        Ok(ciphertext)
    }

    /// Decrypt with AES-256-GCM
    pub fn decrypt_aes_gcm(ciphertext: &[u8], key: &[u8], nonce_bytes: &[u8]) -> Result<Vec<u8>> {
        let key_array: [u8; 32] = key
            .try_into()
            .map_err(|_| Error::Crypto("Invalid key length".to_string()))?;

        let cipher = Aes256Gcm::new(&key_array.into());

        let nonce_array: [u8; 12] = nonce_bytes
            .try_into()
            .map_err(|_| Error::Crypto("Invalid nonce length".to_string()))?;

        let nonce = Nonce::from_slice(&nonce_array);

        let plaintext = cipher
            .decrypt(nonce, ciphertext)
            .map_err(|e| Error::Decryption(e.to_string()))?;

        Ok(plaintext)
    }

    /// Compute SHA-256 hash
    pub fn hash_sha256(data: &[u8]) -> Vec<u8> {
        let mut hasher = Sha256::new();
        hasher.update(data);
        hasher.finalize().to_vec()
    }

    /// Encode bytes to base64 string
    pub fn base64_encode(data: &[u8]) -> String {
        general_purpose::STANDARD.encode(data)
    }

    /// Decode base64 string to bytes
    pub fn base64_decode(data: &str) -> Result<Vec<u8>> {
        general_purpose::STANDARD
            .decode(data)
            .map_err(|e| Error::Base64Decode(e))
    }
}

/// HPKE context for encryption/decryption
pub struct HpkeContext {
    key: Vec<u8>,
    sequence: u64,
}

impl HpkeContext {
    /// Create new HPKE context
    pub fn new(key: Vec<u8>) -> Self {
        Self { key, sequence: 0 }
    }

    /// Seal (encrypt) plaintext
    pub fn seal(&mut self, plaintext: &[u8]) -> Result<Vec<u8>> {
        let nonce = self.sequence.to_be_bytes();
        let mut nonce_12 = [0u8; 12];
        nonce_12[4..].copy_from_slice(&nonce);

        self.sequence += 1;

        let ciphertext = Crypto::encrypt_aes_gcm(plaintext, &self.key, &nonce_12)?;

        // Prepend nonce to ciphertext
        let mut result = nonce_12.to_vec();
        result.extend_from_slice(&ciphertext);

        Ok(result)
    }

    /// Open (decrypt) ciphertext
    pub fn open(&mut self, ciphertext: &[u8]) -> Result<Vec<u8>> {
        if ciphertext.len() < 12 {
            return Err(Error::Decryption("Ciphertext too short".to_string()));
        }

        let nonce = &ciphertext[..12];
        let actual_ciphertext = &ciphertext[12..];

        self.sequence += 1;

        Crypto::decrypt_aes_gcm(actual_ciphertext, &self.key, nonce)
    }
}

/// Setup HPKE as sender (encapsulation)
pub fn setup_hpke_sender(receiver_public_key: &[u8]) -> Result<(HpkeContext, Vec<u8>)> {
    // Generate ephemeral keypair
    let ephemeral = EphemeralSecret::new(OsRng);
    let ephemeral_public = PublicKey::from(&ephemeral);

    let receiver_pk_bytes: [u8; 32] = receiver_public_key
        .try_into()
        .map_err(|_| Error::Crypto("Invalid receiver public key length".to_string()))?;

    let receiver_public = PublicKey::from(receiver_pk_bytes);

    // Compute shared secret
    let shared_secret = ephemeral.diffie_hellman(&receiver_public);

    // Derive encryption key
    let info = b"SAGE HPKE v1";
    let key = Crypto::derive_key(shared_secret.as_bytes(), info, 32)?;

    let context = HpkeContext::new(key);
    let encapsulated_key = ephemeral_public.to_bytes().to_vec();

    Ok((context, encapsulated_key))
}

/// Setup HPKE as receiver (decapsulation)
pub fn setup_hpke_receiver(
    encapsulated_key: &[u8],
    receiver_private_key: &[u8],
) -> Result<HpkeContext> {
    let private_bytes: [u8; 32] = receiver_private_key
        .try_into()
        .map_err(|_| Error::Crypto("Invalid receiver private key length".to_string()))?;

    let public_bytes: [u8; 32] = encapsulated_key
        .try_into()
        .map_err(|_| Error::Crypto("Invalid encapsulated key length".to_string()))?;

    let secret = StaticSecret::from(private_bytes);
    let public = PublicKey::from(public_bytes);

    // Compute shared secret
    let shared_secret = secret.diffie_hellman(&public);

    // Derive encryption key
    let info = b"SAGE HPKE v1";
    let key = Crypto::derive_key(shared_secret.as_bytes(), info, 32)?;

    Ok(HpkeContext::new(key))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ed25519_keypair_generation() {
        let keypair = Crypto::generate_ed25519_keypair().unwrap();
        assert_eq!(keypair.private_key.len(), 32);
        assert_eq!(keypair.public_key.len(), 32);
        assert_eq!(keypair.key_type, KeyType::Ed25519);
    }

    #[test]
    fn test_x25519_keypair_generation() {
        let keypair = Crypto::generate_x25519_keypair().unwrap();
        assert_eq!(keypair.private_key.len(), 32);
        assert_eq!(keypair.public_key.len(), 32);
        assert_eq!(keypair.key_type, KeyType::X25519);
    }

    #[test]
    fn test_sign_and_verify() {
        let keypair = Crypto::generate_ed25519_keypair().unwrap();
        let message = b"Test message";

        let signature = Crypto::sign(message, &keypair.private_key).unwrap();
        assert_eq!(signature.len(), 64);

        let is_valid = Crypto::verify(message, &signature, &keypair.public_key).unwrap();
        assert!(is_valid);
    }

    #[test]
    fn test_hpke_encryption() {
        let receiver_keypair = Crypto::generate_x25519_keypair().unwrap();

        let (mut sender_ctx, encapsulated_key) =
            setup_hpke_sender(&receiver_keypair.public_key).unwrap();
        let plaintext = b"Secret message";
        let ciphertext = sender_ctx.seal(plaintext).unwrap();

        let mut receiver_ctx =
            setup_hpke_receiver(&encapsulated_key, &receiver_keypair.private_key).unwrap();
        let decrypted = receiver_ctx.open(&ciphertext).unwrap();

        assert_eq!(plaintext.to_vec(), decrypted);
    }
}
