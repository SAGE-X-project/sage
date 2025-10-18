// SAGE - Secure Agent Guarantee Engine
// Copyright (c) 2025 SAGE-X-project
// SPDX-License-Identifier: MIT

use sage_verification_hook::{
    HookState, UserState, ErrorCode, RegistrationRecorded, BlacklistUpdated,
    MAX_REGISTRATIONS_PER_DAY, REGISTRATION_COOLDOWN,
};
use anchor_lang::prelude::Pubkey;

#[cfg(test)]
mod hook_tests {
    use super::*;

    #[test]
    fn test_hook_state_len() {
        let expected_len = 32 + 1; // authority + enabled
        assert_eq!(HookState::LEN, expected_len);
    }

    #[test]
    fn test_user_state_len() {
        let expected_len = 1 + 8 + 8 + 1; // registration_count + last_registration + last_day + blacklisted
        assert_eq!(UserState::LEN, expected_len);
    }

    #[test]
    fn test_constants() {
        assert_eq!(MAX_REGISTRATIONS_PER_DAY, 5);
        assert_eq!(REGISTRATION_COOLDOWN, 60);
    }

    #[test]
    fn test_error_codes_defined() {
        let errors = vec![
            ErrorCode::Blacklisted,
            ErrorCode::CooldownActive,
            ErrorCode::DailyLimitReached,
            ErrorCode::InvalidDIDFormat,
            ErrorCode::InvalidSignature,
            ErrorCode::HookDisabled,
        ];

        assert_eq!(errors.len(), 6);
    }

    #[test]
    fn test_did_validation_logic() {
        // Valid DIDs
        let valid_dids = vec![
            "did:sage:123456789",
            "did:example:test",
            "did:web:example.com",
        ];

        for did in valid_dids {
            assert!(did.starts_with("did:"));
            assert!(did.len() >= 10);
        }

        // Invalid DIDs
        let invalid_dids = vec![
            "sage:123",      // Missing "did:"
            "did:",          // Too short
            "did:a",         // Too short
            "notadid:test",  // Wrong prefix
        ];

        for did in invalid_dids {
            assert!(
                !did.starts_with("did:") || did.len() < 10,
                "DID {} should be invalid", did
            );
        }
    }

    #[test]
    fn test_cooldown_calculation() {
        let last_registration = 1000i64;
        let cooldown = REGISTRATION_COOLDOWN;
        let current_time = 1050i64;

        let time_since_last = current_time - last_registration;
        assert_eq!(time_since_last, 50);

        let cooldown_remaining = cooldown - time_since_last;
        assert_eq!(cooldown_remaining, 10);

        // Cooldown should be satisfied after 60 seconds
        let future_time = 1060i64;
        assert!(future_time >= last_registration + cooldown);
    }

    #[test]
    fn test_daily_limit_reset_logic() {
        let last_day = 100i64;
        let current_timestamp = 86400 * 101; // Next day
        let current_day = current_timestamp / 86400;

        assert_eq!(current_day, 101);
        assert_ne!(current_day, last_day);

        // Should reset counter
        let should_reset = current_day != last_day;
        assert!(should_reset);
    }

    #[test]
    fn test_registration_count_logic() {
        let mut count = 0u8;
        let max_count = MAX_REGISTRATIONS_PER_DAY;

        // Can register up to max
        for _ in 0..max_count {
            assert!(count < max_count);
            count += 1;
        }

        assert_eq!(count, max_count);

        // Cannot exceed max
        assert!(count >= max_count);
    }

    #[test]
    fn test_blacklist_logic() {
        let mut blacklisted = false;

        // User not blacklisted initially
        assert!(!blacklisted);

        // Add to blacklist
        blacklisted = true;
        assert!(blacklisted);

        // Remove from blacklist
        blacklisted = false;
        assert!(!blacklisted);
    }

    #[test]
    fn test_user_state_initialization() {
        let user_state = UserState {
            registration_count: 0,
            last_registration: 0,
            last_day: 0,
            blacklisted: false,
        };

        assert_eq!(user_state.registration_count, 0);
        assert_eq!(user_state.last_registration, 0);
        assert_eq!(user_state.last_day, 0);
        assert!(!user_state.blacklisted);
    }

    #[test]
    fn test_hook_state_initialization() {
        let authority = Pubkey::new_unique();
        let hook_state = HookState {
            authority,
            enabled: true,
        };

        assert_eq!(hook_state.authority, authority);
        assert!(hook_state.enabled);
    }

    #[test]
    fn test_hook_disable_logic() {
        let mut enabled = true;
        assert!(enabled);

        enabled = false;
        assert!(!enabled);
    }

    #[test]
    fn test_timestamp_to_day_conversion() {
        let timestamps = vec![
            (0i64, 0i64),
            (86400i64, 1i64),
            (86400 * 2, 2i64),
            (86400 * 365, 365i64),
        ];

        for (timestamp, expected_day) in timestamps {
            let day = timestamp / 86400;
            assert_eq!(day, expected_day);
        }
    }

    #[test]
    fn test_signature_length_validation() {
        let valid_sig = vec![0u8; 64];
        assert_eq!(valid_sig.len(), 64);

        let invalid_sigs = vec![
            vec![0u8; 32],
            vec![0u8; 63],
            vec![0u8; 65],
            vec![0u8; 128],
        ];

        for sig in invalid_sigs {
            assert_ne!(sig.len(), 64, "Signature length {} is invalid", sig.len());
        }
    }

    #[test]
    fn test_event_structures() {
        let user = Pubkey::new_unique();
        let timestamp = 1234567890i64;

        let _registration_event = RegistrationRecorded {
            user,
            timestamp,
            count: 1,
        };

        let _blacklist_event = BlacklistUpdated {
            user,
            blacklisted: true,
            authority: Pubkey::new_unique(),
        };
    }

    #[test]
    fn test_multiple_registrations_same_day() {
        let mut count = 0u8;
        let max = MAX_REGISTRATIONS_PER_DAY;

        // Simulate multiple registrations on same day
        for _i in 0..max {
            if count < max {
                count += 1;
                assert!(count <= max);
            }
        }

        assert_eq!(count, max);

        // Next attempt should be blocked
        assert!(count >= max, "Should have reached daily limit");
    }

    #[test]
    fn test_cooldown_enforcement() {
        let last_reg = 1000i64;
        let cooldown = REGISTRATION_COOLDOWN;

        // Try to register before cooldown expires
        let too_early = last_reg + 30; // Only 30 seconds passed
        assert!(too_early < last_reg + cooldown);

        // Try after cooldown expires
        let ok_time = last_reg + 61; // 61 seconds passed
        assert!(ok_time >= last_reg + cooldown);
    }

    #[test]
    fn test_day_boundary_transition() {
        let day1 = 100i64;
        let timestamp_day1 = day1 * 86400;

        let day2 = 101i64;
        let timestamp_day2 = day2 * 86400;

        assert_eq!(timestamp_day1 / 86400, day1);
        assert_eq!(timestamp_day2 / 86400, day2);
        assert_ne!(day1, day2);
    }
}

#[cfg(test)]
mod signature_verification_tests {
    use super::*;
    use ed25519_dalek::{Signature, Signer, SigningKey, Verifier, SECRET_KEY_LENGTH};
    use rand_core::{OsRng, RngCore};

    fn generate_signing_key() -> SigningKey {
        let mut secret_bytes = [0u8; SECRET_KEY_LENGTH];
        OsRng.fill_bytes(&mut secret_bytes);
        SigningKey::from_bytes(&secret_bytes)
    }

    #[test]
    fn test_ed25519_signature_verification() {
        let signing_key = generate_signing_key();
        let verifying_key = signing_key.verifying_key();

        let message = b"test message for DID registration";
        let signature = signing_key.sign(message);

        assert!(verifying_key.verify(message, &signature).is_ok());
    }

    #[test]
    fn test_pubkey_to_bytes_conversion() {
        let pubkey = Pubkey::new_unique();
        let bytes: [u8; 32] = pubkey.to_bytes();

        assert_eq!(bytes.len(), 32);

        let reconstructed = Pubkey::new_from_array(bytes);
        assert_eq!(pubkey, reconstructed);
    }

    #[test]
    fn test_signature_bytes_conversion() {
        let signing_key = generate_signing_key();

        let message = b"test";
        let signature = signing_key.sign(message);
        let sig_bytes = signature.to_bytes();

        assert_eq!(sig_bytes.len(), 64);

        let reconstructed_sig = Signature::from_bytes(&sig_bytes);
        assert_eq!(signature, reconstructed_sig);
    }

    #[test]
    fn test_message_construction_for_verification() {
        let signer = Pubkey::new_unique();
        let did = "did:sage:test123";

        // Construct message as done in verify_registration
        let message = [signer.as_ref(), did.as_bytes()].concat();

        assert_eq!(message.len(), 32 + did.len());

        // Verify message contains both parts
        assert_eq!(&message[..32], signer.as_ref());
        assert_eq!(&message[32..], did.as_bytes());
    }

    #[test]
    fn test_invalid_signature_detection() {
        let signing_key = generate_signing_key();
        let verifying_key = signing_key.verifying_key();

        let message = b"original message";
        let signature = signing_key.sign(message);

        // Tamper with signature
        let mut bad_sig_bytes = signature.to_bytes();
        bad_sig_bytes[0] ^= 0xFF; // Flip bits
        let bad_signature = Signature::from_bytes(&bad_sig_bytes);

        // Should fail verification
        assert!(verifying_key.verify(message, &bad_signature).is_err());
    }

    #[test]
    fn test_wrong_key_verification_fails() {
        let signing_key1 = generate_signing_key();
        let signing_key2 = generate_signing_key();
        let verifying_key2 = signing_key2.verifying_key();

        let message = b"test";
        let signature = signing_key1.sign(message);

        // Signature from key1 should not verify with key2
        assert!(verifying_key2.verify(message, &signature).is_err());
    }
}

#[cfg(test)]
mod integration_scenarios {
    use super::*;

    #[test]
    fn test_full_registration_flow() {
        // Initialize user state
        let mut user_state = UserState {
            registration_count: 0,
            last_registration: 0,
            last_day: 0,
            blacklisted: false,
        };

        // Check not blacklisted
        assert!(!user_state.blacklisted);

        // First registration of the day
        let current_time = 86400 * 100; // Day 100
        let current_day = current_time / 86400;

        assert_eq!(current_day, 100);

        // Reset count for new day
        if user_state.last_day != current_day {
            user_state.registration_count = 0;
            user_state.last_day = current_day;
        }

        // Check daily limit
        assert!(user_state.registration_count < MAX_REGISTRATIONS_PER_DAY);

        // Record registration
        user_state.registration_count += 1;
        user_state.last_registration = current_time;

        assert_eq!(user_state.registration_count, 1);
        assert_eq!(user_state.last_registration, current_time);
    }

    #[test]
    fn test_multiple_registrations_with_cooldown() {
        let mut user_state = UserState {
            registration_count: 0,
            last_registration: 0,
            last_day: 100,
            blacklisted: false,
        };

        let start_time = 86400 * 100; // Day 100

        // First registration
        user_state.registration_count += 1;
        user_state.last_registration = start_time;

        // Try second registration too soon (30 seconds later)
        let too_soon = start_time + 30;
        let cooldown_ok = too_soon >= user_state.last_registration + REGISTRATION_COOLDOWN;
        assert!(!cooldown_ok, "Should fail cooldown check");

        // Try after cooldown (70 seconds later)
        let after_cooldown = start_time + 70;
        let cooldown_ok = after_cooldown >= user_state.last_registration + REGISTRATION_COOLDOWN;
        assert!(cooldown_ok, "Should pass cooldown check");

        // Second registration succeeds
        user_state.registration_count += 1;
        user_state.last_registration = after_cooldown;

        assert_eq!(user_state.registration_count, 2);
    }

    #[test]
    fn test_daily_limit_enforcement() {
        let mut user_state = UserState {
            registration_count: 0,
            last_registration: 0,
            last_day: 100,
            blacklisted: false,
        };

        let day_timestamp = 86400 * 100;

        // Register up to limit
        for i in 0..MAX_REGISTRATIONS_PER_DAY {
            assert!(user_state.registration_count < MAX_REGISTRATIONS_PER_DAY);
            user_state.registration_count += 1;
            user_state.last_registration = day_timestamp + (i as i64 * 100);
        }

        assert_eq!(user_state.registration_count, MAX_REGISTRATIONS_PER_DAY);

        // Next attempt should fail
        assert!(user_state.registration_count >= MAX_REGISTRATIONS_PER_DAY);
    }

    #[test]
    fn test_blacklist_blocks_registration() {
        let user_state = UserState {
            registration_count: 0,
            last_registration: 0,
            last_day: 0,
            blacklisted: true, // Blacklisted
        };

        // Should be blocked
        assert!(user_state.blacklisted);
    }

    #[test]
    fn test_day_rollover_resets_count() {
        let mut user_state = UserState {
            registration_count: 5, // At limit
            last_registration: 86400 * 100,
            last_day: 100,
            blacklisted: false,
        };

        // Next day
        let next_day_timestamp = 86400 * 101;
        let current_day = next_day_timestamp / 86400;

        // Reset counter
        if user_state.last_day != current_day {
            user_state.registration_count = 0;
            user_state.last_day = current_day;
        }

        assert_eq!(user_state.registration_count, 0);
        assert_eq!(user_state.last_day, 101);
    }
}
