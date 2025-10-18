// SAGE - Secure Agent Guarantee Engine
// Copyright (c) 2025 SAGE-X-project
// SPDX-License-Identifier: MIT

use anchor_lang::prelude::*;
use ed25519_dalek::{Signature, Verifier, VerifyingKey};

declare_id!("Hook1111111111111111111111111111111111111111");

pub const MAX_REGISTRATIONS_PER_DAY: u8 = 5;
pub const REGISTRATION_COOLDOWN: i64 = 60; // 1 minute in seconds

#[program]
pub mod sage_verification_hook {
    use super::*;

    /// Initialize the verification hook
    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        let hook_state = &mut ctx.accounts.hook_state;
        hook_state.authority = ctx.accounts.authority.key();
        hook_state.enabled = true;
        Ok(())
    }

    /// Initialize user state (must be called before verify_registration)
    pub fn initialize_user_state(ctx: Context<InitializeUserState>) -> Result<()> {
        let user_state = &mut ctx.accounts.user_state;
        user_state.registration_count = 0;
        user_state.last_registration = 0;
        user_state.last_day = 0;
        user_state.blacklisted = false;
        Ok(())
    }

    /// Verify registration before it happens
    pub fn verify_registration(
        ctx: Context<VerifyRegistration>,
        did: String,
        signature: Vec<u8>,
        message: Vec<u8>,
    ) -> Result<()> {
        let user_state = &mut ctx.accounts.user_state;
        let clock = Clock::get()?;

        // Check if user is blacklisted
        require!(!user_state.blacklisted, ErrorCode::Blacklisted);

        // Check cooldown
        if user_state.last_registration > 0 {
            require!(
                clock.unix_timestamp >= user_state.last_registration + REGISTRATION_COOLDOWN,
                ErrorCode::CooldownActive
            );
        }

        // Check daily limit
        let current_day = clock.unix_timestamp / 86400;
        if user_state.last_day != current_day {
            user_state.registration_count = 0;
            user_state.last_day = current_day;
        }

        require!(
            user_state.registration_count < MAX_REGISTRATIONS_PER_DAY,
            ErrorCode::DailyLimitReached
        );

        // Verify DID format
        require!(did.starts_with("did:"), ErrorCode::InvalidDIDFormat);
        require!(did.len() >= 10, ErrorCode::InvalidDIDFormat);

        // Verify Ed25519 signature
        verify_ed25519_signature(&ctx.accounts.signer.key(), &message, &signature)?;

        Ok(())
    }

    /// Update user state after registration
    pub fn after_registration(ctx: Context<AfterRegistration>) -> Result<()> {
        let user_state = &mut ctx.accounts.user_state;
        let clock = Clock::get()?;
        
        user_state.registration_count += 1;
        user_state.last_registration = clock.unix_timestamp;
        
        emit!(RegistrationRecorded {
            user: ctx.accounts.signer.key(),
            timestamp: clock.unix_timestamp,
            count: user_state.registration_count,
        });
        
        Ok(())
    }

    /// Add user to blacklist
    pub fn add_to_blacklist(ctx: Context<ManageBlacklist>) -> Result<()> {
        let user_state = &mut ctx.accounts.user_state;
        user_state.blacklisted = true;
        
        emit!(BlacklistUpdated {
            user: ctx.accounts.target_user.key(),
            blacklisted: true,
            authority: ctx.accounts.authority.key(),
        });
        
        Ok(())
    }

    /// Remove user from blacklist
    pub fn remove_from_blacklist(ctx: Context<ManageBlacklist>) -> Result<()> {
        let user_state = &mut ctx.accounts.user_state;
        user_state.blacklisted = false;
        
        emit!(BlacklistUpdated {
            user: ctx.accounts.target_user.key(),
            blacklisted: false,
            authority: ctx.accounts.authority.key(),
        });
        
        Ok(())
    }
}

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(
        init,
        payer = authority,
        space = 8 + HookState::LEN,
        seeds = [b"hook_state"],
        bump
    )]
    pub hook_state: Account<'info, HookState>,
    #[account(mut)]
    pub authority: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct InitializeUserState<'info> {
    #[account(
        init,
        payer = signer,
        space = 8 + UserState::LEN,
        seeds = [b"user_state", signer.key().as_ref()],
        bump
    )]
    pub user_state: Account<'info, UserState>,
    #[account(mut)]
    pub signer: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(did: String)]
pub struct VerifyRegistration<'info> {
    #[account(
        mut,
        seeds = [b"user_state", signer.key().as_ref()],
        bump,
    )]
    pub user_state: Account<'info, UserState>,
    #[account(
        seeds = [b"hook_state"],
        bump,
        constraint = hook_state.enabled @ ErrorCode::HookDisabled
    )]
    pub hook_state: Account<'info, HookState>,
    #[account(mut)]
    pub signer: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct AfterRegistration<'info> {
    #[account(
        mut,
        seeds = [b"user_state", signer.key().as_ref()],
        bump
    )]
    pub user_state: Account<'info, UserState>,
    pub signer: Signer<'info>,
}

#[derive(Accounts)]
pub struct ManageBlacklist<'info> {
    #[account(
        mut,
        seeds = [b"user_state", target_user.key().as_ref()],
        bump
    )]
    pub user_state: Account<'info, UserState>,
    #[account(
        seeds = [b"hook_state"],
        bump,
        has_one = authority
    )]
    pub hook_state: Account<'info, HookState>,
    pub authority: Signer<'info>,
    /// CHECK: The user being blacklisted/unblacklisted
    pub target_user: AccountInfo<'info>,
}

#[account]
pub struct HookState {
    pub authority: Pubkey,
    pub enabled: bool,
}

impl HookState {
    pub const LEN: usize = 32 + 1;
}

#[account]
pub struct UserState {
    pub registration_count: u8,
    pub last_registration: i64,
    pub last_day: i64,
    pub blacklisted: bool,
}

impl UserState {
    pub const LEN: usize = 1 + 8 + 8 + 1;
}

#[event]
pub struct RegistrationRecorded {
    pub user: Pubkey,
    pub timestamp: i64,
    pub count: u8,
}

#[event]
pub struct BlacklistUpdated {
    pub user: Pubkey,
    pub blacklisted: bool,
    pub authority: Pubkey,
}

#[error_code]
pub enum ErrorCode {
    #[msg("User is blacklisted")]
    Blacklisted,
    #[msg("Registration cooldown active")]
    CooldownActive,
    #[msg("Daily registration limit reached")]
    DailyLimitReached,
    #[msg("Invalid DID format")]
    InvalidDIDFormat,
    #[msg("Invalid signature")]
    InvalidSignature,
    #[msg("Hook is disabled")]
    HookDisabled,
}

/// Verify Ed25519 signature using ed25519-dalek
fn verify_ed25519_signature(
    pubkey: &Pubkey,
    message: &[u8],
    signature: &[u8],
) -> Result<()> {
    require!(signature.len() == 64, ErrorCode::InvalidSignature);

    // Convert Pubkey to VerifyingKey (Ed25519 public key is 32 bytes)
    let pubkey_bytes: &[u8; 32] = pubkey.as_ref().try_into()
        .map_err(|_| ErrorCode::InvalidSignature)?;

    let verifying_key = VerifyingKey::from_bytes(pubkey_bytes)
        .map_err(|_| ErrorCode::InvalidSignature)?;

    // Convert signature bytes to Signature
    let sig_bytes: [u8; 64] = signature.try_into()
        .map_err(|_| ErrorCode::InvalidSignature)?;
    let sig = Signature::from_bytes(&sig_bytes);

    // Verify the signature
    verifying_key
        .verify(message, &sig)
        .map_err(|_| ErrorCode::InvalidSignature)?;

    Ok(())
}