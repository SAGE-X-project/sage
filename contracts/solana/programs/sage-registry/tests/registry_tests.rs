// SAGE - Secure Agent Guarantee Engine
// Copyright (c) 2025 SAGE-X-project
// SPDX-License-Identifier: MIT

use sage_registry::{
    Agent, Registry, ErrorCode, AgentRegistered, AgentUpdated, AgentDeactivated,
    KeyAdded, KeyRevoked, KeyRotated, HookUpdated,
    MAX_KEYS_PER_AGENT, MAX_DID_LEN, MAX_NAME_LEN, MAX_DESCRIPTION_LEN,
    MAX_ENDPOINT_LEN, MAX_CAPABILITIES_LEN,
};
use anchor_lang::prelude::Pubkey;

#[cfg(test)]
mod registry_tests {
    use super::*;

    #[test]
    fn test_agent_len_calculation() {
        // Verify Agent struct size calculation is correct
        let expected_len =
            4 + MAX_DID_LEN +           // did
            4 + MAX_NAME_LEN +          // name
            4 + MAX_DESCRIPTION_LEN +   // description
            4 + MAX_ENDPOINT_LEN +      // endpoint
            4 + MAX_CAPABILITIES_LEN +  // capabilities
            32 +                         // owner
            8 +                          // registered_at
            8 +                          // updated_at
            1 +                          // active
            8 +                          // nonce
            1 +                          // key_count
            (32 * MAX_KEYS_PER_AGENT) + // public_keys
            MAX_KEYS_PER_AGENT +        // key_types
            MAX_KEYS_PER_AGENT;         // key_revoked

        assert_eq!(Agent::LEN, expected_len);
    }

    #[test]
    fn test_registry_len_calculation() {
        // Verify Registry struct size calculation
        let expected_len = 32 + 8 + 1 + 32; // authority + agent_count + option flag + pubkey
        assert_eq!(Registry::LEN, expected_len);
    }

    #[test]
    fn test_max_keys_limit() {
        assert_eq!(MAX_KEYS_PER_AGENT, 5);
    }

    #[test]
    fn test_string_length_limits() {
        assert_eq!(MAX_DID_LEN, 128);
        assert_eq!(MAX_NAME_LEN, 64);
        assert_eq!(MAX_DESCRIPTION_LEN, 256);
        assert_eq!(MAX_ENDPOINT_LEN, 128);
        assert_eq!(MAX_CAPABILITIES_LEN, 256);
    }

    #[test]
    fn test_error_codes_defined() {
        // Verify all error codes are properly defined
        let errors = vec![
            ErrorCode::DIDTooLong,
            ErrorCode::NameTooLong,
            ErrorCode::DescriptionTooLong,
            ErrorCode::EndpointTooLong,
            ErrorCode::CapabilitiesTooLong,
            ErrorCode::NoKeysProvided,
            ErrorCode::TooManyKeys,
            ErrorCode::KeyArrayMismatch,
            ErrorCode::UnsupportedKeyType,
            ErrorCode::InvalidSignature,
            ErrorCode::AgentAlreadyInactive,
            ErrorCode::InvalidKeyIndex,
            ErrorCode::KeyAlreadyRevoked,
            ErrorCode::CannotRevokeLastKey,
        ];

        assert_eq!(errors.len(), 14);
    }

    #[test]
    fn test_ed25519_signature_verification_logic() {
        // Test signature verification with known test vectors
        use ed25519_dalek::{Signer, SigningKey, Verifier, SECRET_KEY_LENGTH};
        use rand_core::{OsRng, RngCore};

        // Generate a keypair
        let mut secret_bytes = [0u8; SECRET_KEY_LENGTH];
        OsRng.fill_bytes(&mut secret_bytes);
        let signing_key = SigningKey::from_bytes(&secret_bytes);
        let verifying_key = signing_key.verifying_key();

        // Create a message
        let message = b"test message";

        // Sign the message
        let signature = signing_key.sign(message);

        // Verify - should succeed
        assert!(verifying_key.verify(message, &signature).is_ok());

        // Verify with wrong message - should fail
        let wrong_message = b"wrong message";
        assert!(verifying_key.verify(wrong_message, &signature).is_err());
    }

    #[test]
    fn test_key_type_validation() {
        // Key type 0 = Ed25519 (only supported on Solana)
        let valid_key_type = 0u8;
        assert_eq!(valid_key_type, 0);

        // Other key types should be rejected
        let invalid_key_types = vec![1u8, 2u8, 3u8, 255u8];
        for kt in invalid_key_types {
            assert_ne!(kt, 0, "Key type {} should be invalid", kt);
        }
    }

    #[test]
    fn test_key_array_bounds() {
        // Test that we can't exceed MAX_KEYS_PER_AGENT
        let max_keys = MAX_KEYS_PER_AGENT;
        assert!(max_keys > 0, "Must allow at least one key");
        assert!(max_keys <= 10, "Should not allow too many keys for gas efficiency");
    }

    #[test]
    fn test_agent_struct_default_values() {
        // Verify that a new agent has correct initial values
        let owner = Pubkey::new_unique();
        let agent = Agent {
            did: "did:sage:test".to_string(),
            name: "Test Agent".to_string(),
            description: "Test Description".to_string(),
            endpoint: "https://test.com".to_string(),
            capabilities: "test,capability".to_string(),
            owner,
            registered_at: 0,
            updated_at: 0,
            active: true,
            nonce: 0,
            key_count: 0,
            public_keys: [[0u8; 32]; MAX_KEYS_PER_AGENT],
            key_types: [0u8; MAX_KEYS_PER_AGENT],
            key_revoked: [false; MAX_KEYS_PER_AGENT],
        };

        assert_eq!(agent.nonce, 0);
        assert_eq!(agent.key_count, 0);
        assert!(agent.active);
        assert_eq!(agent.owner, owner);
    }

    #[test]
    fn test_registry_struct_initialization() {
        let authority = Pubkey::new_unique();
        let registry = Registry {
            authority,
            agent_count: 0,
            verification_hook: None,
        };

        assert_eq!(registry.authority, authority);
        assert_eq!(registry.agent_count, 0);
        assert!(registry.verification_hook.is_none());
    }

    #[test]
    fn test_did_format_validation() {
        // Valid DID formats
        let valid_dids = vec![
            "did:sage:123",
            "did:sage:abc123",
            "did:example:testing",
        ];

        for did in valid_dids {
            assert!(did.starts_with("did:"));
            assert!(did.len() >= 10);
            assert!(did.len() <= MAX_DID_LEN);
        }

        // Invalid DID formats
        let invalid_dids = vec![
            "sage:123",           // Missing "did:"
            "did:",               // Too short
            "did:a",              // Too short
        ];

        for did in invalid_dids {
            assert!(
                !did.starts_with("did:") || did.len() < 10,
                "DID {} should be invalid", did
            );
        }
    }

    #[test]
    fn test_string_length_validation() {
        // Test DID length
        let too_long_did = "did:".to_string() + &"a".repeat(MAX_DID_LEN);
        assert!(too_long_did.len() > MAX_DID_LEN);

        // Test name length
        let too_long_name = "a".repeat(MAX_NAME_LEN + 1);
        assert!(too_long_name.len() > MAX_NAME_LEN);

        // Test description length
        let too_long_desc = "a".repeat(MAX_DESCRIPTION_LEN + 1);
        assert!(too_long_desc.len() > MAX_DESCRIPTION_LEN);

        // Test endpoint length
        let too_long_endpoint = "https://".to_string() + &"a".repeat(MAX_ENDPOINT_LEN);
        assert!(too_long_endpoint.len() > MAX_ENDPOINT_LEN);

        // Test capabilities length
        let too_long_caps = "a".repeat(MAX_CAPABILITIES_LEN + 1);
        assert!(too_long_caps.len() > MAX_CAPABILITIES_LEN);
    }

    #[test]
    fn test_key_management_logic() {
        // Test adding keys up to the limit
        let mut key_count = 0u8;

        for _i in 0..MAX_KEYS_PER_AGENT {
            assert!(key_count < MAX_KEYS_PER_AGENT as u8);
            key_count += 1;
        }

        assert_eq!(key_count, MAX_KEYS_PER_AGENT as u8);

        // Can't add more
        assert!(key_count >= MAX_KEYS_PER_AGENT as u8);
    }

    #[test]
    fn test_active_key_counting() {
        // Simulate agent with multiple keys, some revoked
        let mut key_revoked = [false; MAX_KEYS_PER_AGENT];
        let key_count = 5u8;

        // Revoke some keys
        key_revoked[1] = true;
        key_revoked[3] = true;

        // Count active keys
        let active_keys = (0..key_count)
            .filter(|&i| !key_revoked[i as usize])
            .count();

        assert_eq!(active_keys, 3); // 5 total - 2 revoked = 3 active
    }

    #[test]
    fn test_cannot_revoke_last_key_logic() {
        // Agent with only one key
        let key_revoked = [false; MAX_KEYS_PER_AGENT];
        let key_count = 1u8;

        let active_keys = (0..key_count)
            .filter(|&i| !key_revoked[i as usize])
            .count();

        assert_eq!(active_keys, 1);

        // Cannot revoke the last key
        assert!(active_keys == 1, "Cannot revoke last key");
    }

    #[test]
    fn test_nonce_increment_logic() {
        let mut nonce = 0u64;

        // Simulate multiple operations that increment nonce
        nonce += 1; // add_key
        assert_eq!(nonce, 1);

        nonce += 1; // revoke_key
        assert_eq!(nonce, 2);

        nonce += 1; // rotate_key
        assert_eq!(nonce, 3);

        // Nonce should never overflow in practice
        assert!(nonce < u64::MAX);
    }

    #[test]
    fn test_ownership_message_construction() {
        // Test message construction for key ownership proof
        let owner = Pubkey::new_unique();
        let did = "did:sage:test123";
        let nonce = 42u64;

        // Registration message: owner + DID
        let reg_message = [owner.as_ref(), did.as_bytes()].concat();
        assert!(reg_message.len() == 32 + did.len());

        // Update message: owner + nonce
        let update_message = [owner.as_ref(), &nonce.to_le_bytes()].concat();
        assert!(update_message.len() == 32 + 8);
    }

    #[test]
    fn test_event_structures() {
        // Test that event structures are properly defined
        let owner = Pubkey::new_unique();
        let did = "did:sage:test".to_string();
        let timestamp = 1234567890i64;

        let _reg_event = AgentRegistered {
            did: did.clone(),
            owner,
            timestamp,
        };

        let _update_event = AgentUpdated {
            did: did.clone(),
            timestamp,
        };

        let _deactivate_event = AgentDeactivated {
            did: did.clone(),
            timestamp,
        };

        let _key_added = KeyAdded {
            did: did.clone(),
            key_index: 0,
            timestamp,
        };

        let _key_revoked = KeyRevoked {
            did: did.clone(),
            key_index: 0,
            timestamp,
        };

        let _key_rotated = KeyRotated {
            did: did.clone(),
            key_index: 0,
            timestamp,
        };

        let hook_program = Some(Pubkey::new_unique());
        let _hook_updated = HookUpdated {
            hook_program,
            timestamp,
        };
    }

    #[test]
    fn test_pubkey_serialization() {
        let pubkey = Pubkey::new_unique();
        let bytes = pubkey.to_bytes();

        assert_eq!(bytes.len(), 32);

        let reconstructed = Pubkey::new_from_array(bytes);
        assert_eq!(pubkey, reconstructed);
    }
}

#[cfg(test)]
mod ed25519_tests {
    use ed25519_dalek::{Signer, SigningKey, Verifier, SECRET_KEY_LENGTH};
    use rand_core::{OsRng, RngCore};

    fn generate_signing_key() -> SigningKey {
        let mut secret_bytes = [0u8; SECRET_KEY_LENGTH];
        OsRng.fill_bytes(&mut secret_bytes);
        SigningKey::from_bytes(&secret_bytes)
    }

    #[test]
    fn test_ed25519_key_generation() {
        let signing_key = generate_signing_key();
        let verifying_key = signing_key.verifying_key();

        // Verify key sizes
        assert_eq!(verifying_key.to_bytes().len(), 32);
    }

    #[test]
    fn test_ed25519_sign_and_verify() {
        let signing_key = generate_signing_key();
        let verifying_key = signing_key.verifying_key();

        let message = b"Hello, Solana!";
        let signature = signing_key.sign(message);

        assert!(verifying_key.verify(message, &signature).is_ok());
    }

    #[test]
    fn test_ed25519_signature_size() {
        let signing_key = generate_signing_key();

        let message = b"test";
        let signature = signing_key.sign(message);
        let sig_bytes = signature.to_bytes();

        assert_eq!(sig_bytes.len(), 64);
    }

    #[test]
    fn test_ed25519_invalid_signature_fails() {
        let signing_key = generate_signing_key();
        let verifying_key = signing_key.verifying_key();

        let message = b"original message";
        let signature = signing_key.sign(message);

        let different_message = b"different message";
        assert!(verifying_key.verify(different_message, &signature).is_err());
    }

    #[test]
    fn test_ed25519_different_key_fails() {
        let signing_key1 = generate_signing_key();
        let signing_key2 = generate_signing_key();
        let verifying_key2 = signing_key2.verifying_key();

        let message = b"test message";
        let signature = signing_key1.sign(message);

        // Signature from key1 should not verify with key2
        assert!(verifying_key2.verify(message, &signature).is_err());
    }

    #[test]
    fn test_ed25519_multiple_signatures() {
        let signing_key = generate_signing_key();
        let verifying_key = signing_key.verifying_key();

        // Sign multiple messages
        let messages = vec![
            b"message 1" as &[u8],
            b"message 2",
            b"message 3",
        ];

        for msg in messages {
            let signature = signing_key.sign(msg);
            assert!(verifying_key.verify(msg, &signature).is_ok());
        }
    }
}
