// SAGE - Secure Agent Guarantee Engine
// Copyright (c) 2025 SAGE-X-project
// SPDX-License-Identifier: MIT

use anchor_lang::prelude::*;
use ed25519_dalek::{Signature, Verifier, VerifyingKey};

declare_id!("11111111111111111111111111111111");

pub const MAX_KEYS_PER_AGENT: usize = 5;
pub const MAX_DID_LEN: usize = 128;
pub const MAX_NAME_LEN: usize = 64;
pub const MAX_DESCRIPTION_LEN: usize = 256;
pub const MAX_ENDPOINT_LEN: usize = 128;
pub const MAX_CAPABILITIES_LEN: usize = 256;

#[program]
pub mod sage_registry {
    use super::*;

    /// Initialize the registry
    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        let registry = &mut ctx.accounts.registry;
        registry.authority = ctx.accounts.authority.key();
        registry.agent_count = 0;
        registry.verification_hook = None;
        Ok(())
    }

    /// Register a new agent with multiple keys
    pub fn register_agent(
        ctx: Context<RegisterAgent>,
        did: String,
        name: String,
        description: String,
        endpoint: String,
        capabilities: String,
        public_keys: Vec<[u8; 32]>,
        key_types: Vec<u8>,
        signatures: Vec<[u8; 64]>,
    ) -> Result<()> {
        // Validate inputs
        require!(did.len() <= MAX_DID_LEN, ErrorCode::DIDTooLong);
        require!(name.len() <= MAX_NAME_LEN, ErrorCode::NameTooLong);
        require!(description.len() <= MAX_DESCRIPTION_LEN, ErrorCode::DescriptionTooLong);
        require!(endpoint.len() <= MAX_ENDPOINT_LEN, ErrorCode::EndpointTooLong);
        require!(capabilities.len() <= MAX_CAPABILITIES_LEN, ErrorCode::CapabilitiesTooLong);
        require!(!public_keys.is_empty(), ErrorCode::NoKeysProvided);
        require!(public_keys.len() <= MAX_KEYS_PER_AGENT, ErrorCode::TooManyKeys);
        require!(public_keys.len() == key_types.len(), ErrorCode::KeyArrayMismatch);
        require!(public_keys.len() == signatures.len(), ErrorCode::KeyArrayMismatch);

        // Verify all key ownership proofs
        // For initial registration, use owner pubkey + DID as message (nonce is not yet initialized)
        let message = [
            ctx.accounts.owner.key().as_ref(),
            did.as_bytes(),
        ]
        .concat();

        for i in 0..public_keys.len() {
            let key_type = key_types[i];
            require!(key_type == 0, ErrorCode::UnsupportedKeyType); // Only Ed25519 supported on Solana

            verify_ed25519_signature(&public_keys[i], &message, &signatures[i])?;
        }

        let agent = &mut ctx.accounts.agent;
        let registry = &mut ctx.accounts.registry;
        let clock = Clock::get()?;

        // Initialize agent
        agent.did = did;
        agent.name = name;
        agent.description = description;
        agent.endpoint = endpoint;
        agent.capabilities = capabilities;
        agent.owner = ctx.accounts.owner.key();
        agent.registered_at = clock.unix_timestamp;
        agent.updated_at = clock.unix_timestamp;
        agent.active = true;
        agent.nonce = 0;
        agent.key_count = public_keys.len() as u8;

        // Store keys
        for i in 0..public_keys.len() {
            agent.public_keys[i] = public_keys[i];
            agent.key_types[i] = key_types[i];
            agent.key_revoked[i] = false;
        }

        registry.agent_count += 1;

        emit!(AgentRegistered {
            did: agent.did.clone(),
            owner: agent.owner,
            timestamp: clock.unix_timestamp,
        });

        Ok(())
    }

    /// Add a new key to an existing agent
    pub fn add_key(
        ctx: Context<UpdateAgent>,
        public_key: [u8; 32],
        key_type: u8,
        signature: [u8; 64],
    ) -> Result<()> {
        let agent = &mut ctx.accounts.agent;

        require!(agent.key_count < MAX_KEYS_PER_AGENT as u8, ErrorCode::TooManyKeys);
        require!(key_type == 0, ErrorCode::UnsupportedKeyType); // Only Ed25519 on Solana

        // Verify key ownership proof
        let message = [
            ctx.accounts.owner.key().as_ref(),
            &agent.nonce.to_le_bytes(),
        ]
        .concat();

        verify_ed25519_signature(&public_key, &message, &signature)?;

        // Add key
        let idx = agent.key_count as usize;
        agent.public_keys[idx] = public_key;
        agent.key_types[idx] = key_type;
        agent.key_revoked[idx] = false;
        agent.key_count += 1;
        agent.nonce += 1;
        agent.updated_at = Clock::get()?.unix_timestamp;

        emit!(KeyAdded {
            did: agent.did.clone(),
            key_index: idx as u8,
            timestamp: agent.updated_at,
        });

        Ok(())
    }

    /// Revoke a key
    pub fn revoke_key(
        ctx: Context<UpdateAgent>,
        key_index: u8,
    ) -> Result<()> {
        let agent = &mut ctx.accounts.agent;

        require!(key_index < agent.key_count, ErrorCode::InvalidKeyIndex);
        require!(!agent.key_revoked[key_index as usize], ErrorCode::KeyAlreadyRevoked);

        // Count active keys
        let active_keys = (0..agent.key_count)
            .filter(|&i| !agent.key_revoked[i as usize])
            .count();

        require!(active_keys > 1, ErrorCode::CannotRevokeLastKey);

        // Revoke the key
        agent.key_revoked[key_index as usize] = true;
        agent.nonce += 1;
        agent.updated_at = Clock::get()?.unix_timestamp;

        emit!(KeyRevoked {
            did: agent.did.clone(),
            key_index,
            timestamp: agent.updated_at,
        });

        Ok(())
    }

    /// Rotate a key atomically
    pub fn rotate_key(
        ctx: Context<UpdateAgent>,
        old_key_index: u8,
        new_public_key: [u8; 32],
        new_key_type: u8,
        signature: [u8; 64],
    ) -> Result<()> {
        let agent = &mut ctx.accounts.agent;

        require!(old_key_index < agent.key_count, ErrorCode::InvalidKeyIndex);
        require!(!agent.key_revoked[old_key_index as usize], ErrorCode::KeyAlreadyRevoked);
        require!(new_key_type == 0, ErrorCode::UnsupportedKeyType);

        // Verify new key ownership
        let message = [
            ctx.accounts.owner.key().as_ref(),
            &agent.nonce.to_le_bytes(),
        ]
        .concat();

        verify_ed25519_signature(&new_public_key, &message, &signature)?;

        // Atomically replace the key
        agent.public_keys[old_key_index as usize] = new_public_key;
        agent.key_types[old_key_index as usize] = new_key_type;
        agent.nonce += 1;
        agent.updated_at = Clock::get()?.unix_timestamp;

        emit!(KeyRotated {
            did: agent.did.clone(),
            key_index: old_key_index,
            timestamp: agent.updated_at,
        });

        Ok(())
    }

    /// Update agent metadata
    pub fn update_agent(
        ctx: Context<UpdateAgent>,
        name: Option<String>,
        description: Option<String>,
        endpoint: Option<String>,
        capabilities: Option<String>,
    ) -> Result<()> {
        let agent = &mut ctx.accounts.agent;

        if let Some(n) = name {
            require!(n.len() <= MAX_NAME_LEN, ErrorCode::NameTooLong);
            agent.name = n;
        }
        if let Some(d) = description {
            require!(d.len() <= MAX_DESCRIPTION_LEN, ErrorCode::DescriptionTooLong);
            agent.description = d;
        }
        if let Some(e) = endpoint {
            require!(e.len() <= MAX_ENDPOINT_LEN, ErrorCode::EndpointTooLong);
            agent.endpoint = e;
        }
        if let Some(c) = capabilities {
            require!(c.len() <= MAX_CAPABILITIES_LEN, ErrorCode::CapabilitiesTooLong);
            agent.capabilities = c;
        }

        agent.updated_at = Clock::get()?.unix_timestamp;

        emit!(AgentUpdated {
            did: agent.did.clone(),
            timestamp: agent.updated_at,
        });

        Ok(())
    }

    /// Deactivate an agent
    pub fn deactivate_agent(ctx: Context<UpdateAgent>) -> Result<()> {
        let agent = &mut ctx.accounts.agent;
        require!(agent.active, ErrorCode::AgentAlreadyInactive);

        agent.active = false;
        agent.updated_at = Clock::get()?.unix_timestamp;

        emit!(AgentDeactivated {
            did: agent.did.clone(),
            timestamp: agent.updated_at,
        });

        Ok(())
    }

    /// Set verification hook
    pub fn set_verification_hook(
        ctx: Context<SetHook>,
        hook_program: Option<Pubkey>,
    ) -> Result<()> {
        let registry = &mut ctx.accounts.registry;
        registry.verification_hook = hook_program;

        emit!(HookUpdated {
            hook_program,
            timestamp: Clock::get()?.unix_timestamp,
        });

        Ok(())
    }
}

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(
        init,
        payer = authority,
        space = 8 + Registry::LEN,
        seeds = [b"registry"],
        bump
    )]
    pub registry: Account<'info, Registry>,
    #[account(mut)]
    pub authority: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(did: String)]
pub struct RegisterAgent<'info> {
    #[account(
        init,
        payer = owner,
        space = 8 + Agent::LEN,
        seeds = [b"agent", did.as_bytes()],
        bump
    )]
    pub agent: Account<'info, Agent>,
    #[account(
        mut,
        seeds = [b"registry"],
        bump
    )]
    pub registry: Account<'info, Registry>,
    #[account(mut)]
    pub owner: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct UpdateAgent<'info> {
    #[account(
        mut,
        seeds = [b"agent", agent.did.as_bytes()],
        bump,
        has_one = owner
    )]
    pub agent: Account<'info, Agent>,
    pub owner: Signer<'info>,
}

#[derive(Accounts)]
pub struct SetHook<'info> {
    #[account(
        mut,
        seeds = [b"registry"],
        bump,
        has_one = authority
    )]
    pub registry: Account<'info, Registry>,
    pub authority: Signer<'info>,
}

#[account]
pub struct Registry {
    pub authority: Pubkey,
    pub agent_count: u64,
    pub verification_hook: Option<Pubkey>,
}

impl Registry {
    pub const LEN: usize = 32 + 8 + 1 + 32;
}

#[account]
pub struct Agent {
    pub did: String,
    pub name: String,
    pub description: String,
    pub endpoint: String,
    pub capabilities: String,
    pub owner: Pubkey,
    pub registered_at: i64,
    pub updated_at: i64,
    pub active: bool,
    pub nonce: u64,
    pub key_count: u8,
    pub public_keys: [[u8; 32]; MAX_KEYS_PER_AGENT],
    pub key_types: [u8; MAX_KEYS_PER_AGENT],
    pub key_revoked: [bool; MAX_KEYS_PER_AGENT],
}

impl Agent {
    pub const LEN: usize =
        4 + MAX_DID_LEN +
        4 + MAX_NAME_LEN +
        4 + MAX_DESCRIPTION_LEN +
        4 + MAX_ENDPOINT_LEN +
        4 + MAX_CAPABILITIES_LEN +
        32 + // owner
        8 +  // registered_at
        8 +  // updated_at
        1 +  // active
        8 +  // nonce
        1 +  // key_count
        (32 * MAX_KEYS_PER_AGENT) + // public_keys
        MAX_KEYS_PER_AGENT +         // key_types
        MAX_KEYS_PER_AGENT;          // key_revoked
}

#[event]
pub struct AgentRegistered {
    pub did: String,
    pub owner: Pubkey,
    pub timestamp: i64,
}

#[event]
pub struct AgentUpdated {
    pub did: String,
    pub timestamp: i64,
}

#[event]
pub struct AgentDeactivated {
    pub did: String,
    pub timestamp: i64,
}

#[event]
pub struct KeyAdded {
    pub did: String,
    pub key_index: u8,
    pub timestamp: i64,
}

#[event]
pub struct KeyRevoked {
    pub did: String,
    pub key_index: u8,
    pub timestamp: i64,
}

#[event]
pub struct KeyRotated {
    pub did: String,
    pub key_index: u8,
    pub timestamp: i64,
}

#[event]
pub struct HookUpdated {
    pub hook_program: Option<Pubkey>,
    pub timestamp: i64,
}

#[error_code]
pub enum ErrorCode {
    #[msg("DID too long")]
    DIDTooLong,
    #[msg("Name too long")]
    NameTooLong,
    #[msg("Description too long")]
    DescriptionTooLong,
    #[msg("Endpoint too long")]
    EndpointTooLong,
    #[msg("Capabilities too long")]
    CapabilitiesTooLong,
    #[msg("No keys provided")]
    NoKeysProvided,
    #[msg("Too many keys")]
    TooManyKeys,
    #[msg("Key array length mismatch")]
    KeyArrayMismatch,
    #[msg("Unsupported key type")]
    UnsupportedKeyType,
    #[msg("Invalid signature")]
    InvalidSignature,
    #[msg("Agent already inactive")]
    AgentAlreadyInactive,
    #[msg("Invalid key index")]
    InvalidKeyIndex,
    #[msg("Key already revoked")]
    KeyAlreadyRevoked,
    #[msg("Cannot revoke last key")]
    CannotRevokeLastKey,
}

/// Verify Ed25519 signature using ed25519-dalek
fn verify_ed25519_signature(
    pubkey: &[u8; 32],
    message: &[u8],
    signature: &[u8; 64],
) -> Result<()> {
    // Convert public key bytes to VerifyingKey
    let verifying_key = VerifyingKey::from_bytes(pubkey)
        .map_err(|_| ErrorCode::InvalidSignature)?;

    // Convert signature bytes to Signature
    let sig = Signature::from_bytes(signature);

    // Verify the signature
    verifying_key
        .verify(message, &sig)
        .map_err(|_| ErrorCode::InvalidSignature)?;

    Ok(())
}
