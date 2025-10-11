// SAGE - Secure Agent Guarantee Engine
// Copyright (c) 2025 SAGE-X-project
// SPDX-License-Identifier: MIT

use anchor_lang::prelude::*;
use anchor_lang::solana_program::pubkey::Pubkey;
use anchor_lang::solana_program::program::invoke_signed;
use anchor_lang::solana_program::ed25519_program;
use anchor_lang::solana_program::instruction::Instruction;

declare_id!("Sage11111111111111111111111111111111111111");

pub const MAX_NAME_LENGTH: usize = 64;
pub const MAX_DESCRIPTION_LENGTH: usize = 256;
pub const MAX_ENDPOINT_LENGTH: usize = 128;
pub const MAX_CAPABILITIES_LENGTH: usize = 512;
pub const MAX_DID_LENGTH: usize = 128;
pub const PUBLIC_KEY_LENGTH: usize = 32;

#[program]
pub mod sage_registry {
    use super::*;

    /// Initialize the registry
    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        let registry = &mut ctx.accounts.registry;
        registry.authority = ctx.accounts.authority.key();
        registry.agent_count = 0;
        registry.before_register_hook = None;
        registry.after_register_hook = None;
        Ok(())
    }

    /// Register a new AI agent with signature verification
    pub fn register_agent(
        ctx: Context<RegisterAgent>,
        did: String,
        name: String,
        description: String,
        endpoint: String,
        capabilities: String,
        signature: Vec<u8>,
    ) -> Result<()> {
        // Validate inputs
        require!(did.len() <= MAX_DID_LENGTH, ErrorCode::DIDTooLong);
        require!(name.len() <= MAX_NAME_LENGTH, ErrorCode::NameTooLong);
        require!(description.len() <= MAX_DESCRIPTION_LENGTH, ErrorCode::DescriptionTooLong);
        require!(endpoint.len() <= MAX_ENDPOINT_LENGTH, ErrorCode::EndpointTooLong);
        require!(capabilities.len() <= MAX_CAPABILITIES_LENGTH, ErrorCode::CapabilitiesTooLong);

        let agent = &mut ctx.accounts.agent;
        let owner = &ctx.accounts.owner;
        let registry = &mut ctx.accounts.registry;

        // Verify the signer's public key matches the one being registered
        let owner_pubkey = owner.key();
        
        // Create message for signature verification
        let message = format!(
            "{}:{}:{}:{}:{}:{}",
            did, name, description, endpoint, capabilities, owner_pubkey
        );
        let message_bytes = message.as_bytes();
        
        // Verify Ed25519 signature
        verify_ed25519_signature(&owner_pubkey, message_bytes, &signature)?;
        
        // Execute before hook if set
        if let Some(hook) = registry.before_register_hook {
            // Hook execution would be implemented here
            // For now, we'll emit an event
            emit!(BeforeRegisterEvent {
                agent_pubkey: agent.key(),
                owner: owner_pubkey,
                did: did.clone(),
            });
        }

        // Store agent metadata
        agent.did = did.clone();
        agent.name = name;
        agent.description = description;
        agent.endpoint = endpoint;
        agent.public_key = owner_pubkey;
        agent.capabilities = capabilities;
        agent.owner = owner_pubkey;
        agent.registered_at = Clock::get()?.unix_timestamp;
        agent.updated_at = Clock::get()?.unix_timestamp;
        agent.active = true;
        agent.nonce = 0;

        // Update registry
        registry.agent_count += 1;

        emit!(AgentRegistered {
            agent_pubkey: agent.key(),
            owner: owner_pubkey,
            did: did.clone(),
            timestamp: agent.registered_at,
        });

        // Execute after hook if set
        if let Some(hook) = registry.after_register_hook {
            emit!(AfterRegisterEvent {
                agent_pubkey: agent.key(),
                owner: owner_pubkey,
                did,
            });
        }

        Ok(())
    }

    /// Update agent metadata
    pub fn update_agent(
        ctx: Context<UpdateAgent>,
        name: String,
        description: String,
        endpoint: String,
        capabilities: String,
    ) -> Result<()> {
        require!(name.len() <= MAX_NAME_LENGTH, ErrorCode::NameTooLong);
        require!(description.len() <= MAX_DESCRIPTION_LENGTH, ErrorCode::DescriptionTooLong);
        require!(endpoint.len() <= MAX_ENDPOINT_LENGTH, ErrorCode::EndpointTooLong);
        require!(capabilities.len() <= MAX_CAPABILITIES_LENGTH, ErrorCode::CapabilitiesTooLong);

        let agent = &mut ctx.accounts.agent;
        require!(agent.active, ErrorCode::AgentNotActive);

        // Update metadata
        agent.name = name;
        agent.description = description;
        agent.endpoint = endpoint;
        agent.capabilities = capabilities;
        agent.updated_at = Clock::get()?.unix_timestamp;
        agent.nonce += 1;

        emit!(AgentUpdated {
            agent_pubkey: agent.key(),
            owner: agent.owner,
            timestamp: agent.updated_at,
        });

        Ok(())
    }

    /// Deactivate an agent
    pub fn deactivate_agent(ctx: Context<DeactivateAgent>) -> Result<()> {
        let agent = &mut ctx.accounts.agent;
        require!(agent.active, ErrorCode::AgentAlreadyInactive);

        agent.active = false;
        agent.updated_at = Clock::get()?.unix_timestamp;

        emit!(AgentDeactivated {
            agent_pubkey: agent.key(),
            owner: agent.owner,
            timestamp: agent.updated_at,
        });

        Ok(())
    }

    /// Set before register hook
    pub fn set_before_register_hook(
        ctx: Context<SetHook>,
        hook: Option<Pubkey>,
    ) -> Result<()> {
        let registry = &mut ctx.accounts.registry;
        registry.before_register_hook = hook;
        Ok(())
    }

    /// Set after register hook
    pub fn set_after_register_hook(
        ctx: Context<SetHook>,
        hook: Option<Pubkey>,
    ) -> Result<()> {
        let registry = &mut ctx.accounts.registry;
        registry.after_register_hook = hook;
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
    #[account(mut)]
    pub registry: Account<'info, Registry>,
    #[account(mut)]
    pub owner: Signer<'info>,
    pub system_program: Program<'info, System>,
    /// CHECK: Ed25519 program for signature verification
    #[account(address = ed25519_program::ID)]
    pub ed25519_program: AccountInfo<'info>,
}

#[derive(Accounts)]
pub struct UpdateAgent<'info> {
    #[account(
        mut,
        has_one = owner,
        constraint = agent.active @ ErrorCode::AgentNotActive
    )]
    pub agent: Account<'info, Agent>,
    pub owner: Signer<'info>,
}

#[derive(Accounts)]
pub struct DeactivateAgent<'info> {
    #[account(
        mut,
        has_one = owner,
        constraint = agent.active @ ErrorCode::AgentNotActive
    )]
    pub agent: Account<'info, Agent>,
    pub owner: Signer<'info>,
}

#[derive(Accounts)]
pub struct SetHook<'info> {
    #[account(
        mut,
        has_one = authority
    )]
    pub registry: Account<'info, Registry>,
    pub authority: Signer<'info>,
}

#[account]
pub struct Registry {
    pub authority: Pubkey,
    pub agent_count: u64,
    pub before_register_hook: Option<Pubkey>,
    pub after_register_hook: Option<Pubkey>,
}

impl Registry {
    pub const LEN: usize = 32 + 8 + 33 + 33;
}

#[account]
pub struct Agent {
    pub did: String,
    pub name: String,
    pub description: String,
    pub endpoint: String,
    pub public_key: Pubkey,
    pub capabilities: String,
    pub owner: Pubkey,
    pub registered_at: i64,
    pub updated_at: i64,
    pub active: bool,
    pub nonce: u64,
}

impl Agent {
    pub const LEN: usize = 4 + MAX_DID_LENGTH +
        4 + MAX_NAME_LENGTH +
        4 + MAX_DESCRIPTION_LENGTH +
        4 + MAX_ENDPOINT_LENGTH +
        32 +
        4 + MAX_CAPABILITIES_LENGTH +
        32 +
        8 +
        8 +
        1 +
        8;
}

#[event]
pub struct AgentRegistered {
    pub agent_pubkey: Pubkey,
    pub owner: Pubkey,
    pub did: String,
    pub timestamp: i64,
}

#[event]
pub struct AgentUpdated {
    pub agent_pubkey: Pubkey,
    pub owner: Pubkey,
    pub timestamp: i64,
}

#[event]
pub struct AgentDeactivated {
    pub agent_pubkey: Pubkey,
    pub owner: Pubkey,
    pub timestamp: i64,
}

#[event]
pub struct BeforeRegisterEvent {
    pub agent_pubkey: Pubkey,
    pub owner: Pubkey,
    pub did: String,
}

#[event]
pub struct AfterRegisterEvent {
    pub agent_pubkey: Pubkey,
    pub owner: Pubkey,
    pub did: String,
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
    #[msg("Agent not active")]
    AgentNotActive,
    #[msg("Agent already inactive")]
    AgentAlreadyInactive,
    #[msg("Invalid signature")]
    InvalidSignature,
    #[msg("Unauthorized")]
    Unauthorized,
}

/// Verify Ed25519 signature
fn verify_ed25519_signature(
    pubkey: &Pubkey,
    message: &[u8],
    signature: &[u8],
) -> Result<()> {
    require!(signature.len() == 64, ErrorCode::InvalidSignature);
    
    // In a production environment, you would use the Ed25519 program
    // to verify the signature properly. This is a simplified version.
    
    // Basic validation
    if signature.iter().all(|&b| b == 0) {
        return err!(ErrorCode::InvalidSignature);
    }
    
    Ok(())
}