//! Session management module

use crate::crypto::HpkeContext;
use crate::error::{Error, Result};
use chrono::{DateTime, Duration, Utc};
use std::collections::HashMap;

/// Session represents a secure session between client and server
pub struct Session {
    pub session_id: String,
    pub client_did: String,
    pub server_did: String,
    pub hpke_context: HpkeContext,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub last_activity: DateTime<Utc>,
    pub message_count: u64,
}

impl Session {
    /// Create new session
    pub fn new(
        session_id: String,
        client_did: String,
        server_did: String,
        hpke_context: HpkeContext,
        max_age_seconds: i64,
    ) -> Self {
        let now = Utc::now();
        Self {
            session_id,
            client_did,
            server_did,
            hpke_context,
            created_at: now,
            expires_at: now + Duration::seconds(max_age_seconds),
            last_activity: now,
            message_count: 0,
        }
    }

    /// Check if session is expired
    pub fn is_expired(&self) -> bool {
        Utc::now() > self.expires_at
    }

    /// Update last activity timestamp
    pub fn update_activity(&mut self) -> Result<()> {
        if self.is_expired() {
            return Err(Error::SessionExpired(self.session_id.clone()));
        }
        self.last_activity = Utc::now();
        Ok(())
    }

    /// Encrypt message using session context
    pub fn encrypt(&mut self, plaintext: &[u8]) -> Result<Vec<u8>> {
        if self.is_expired() {
            return Err(Error::SessionExpired(self.session_id.clone()));
        }
        self.update_activity()?;
        self.message_count += 1;
        self.hpke_context.seal(plaintext)
    }

    /// Decrypt message using session context
    pub fn decrypt(&mut self, ciphertext: &[u8]) -> Result<Vec<u8>> {
        if self.is_expired() {
            return Err(Error::SessionExpired(self.session_id.clone()));
        }
        self.update_activity()?;
        self.hpke_context.open(ciphertext)
    }
}

/// Session manager
pub struct SessionManager {
    sessions: HashMap<String, Session>,
    max_sessions: usize,
}

impl SessionManager {
    /// Create new session manager
    pub fn new(max_sessions: usize) -> Self {
        Self {
            sessions: HashMap::new(),
            max_sessions,
        }
    }

    /// Add session
    pub fn add_session(&mut self, session: Session) -> Result<()> {
        self.cleanup_expired();

        if self.sessions.len() >= self.max_sessions {
            return Err(Error::Session(format!(
                "Too many sessions ({}/{})",
                self.sessions.len(),
                self.max_sessions
            )));
        }

        self.sessions
            .insert(session.session_id.clone(), session);
        Ok(())
    }

    /// Get session by ID
    pub fn get_session(&mut self, session_id: &str) -> Option<&mut Session> {
        if let Some(session) = self.sessions.get_mut(session_id) {
            if session.is_expired() {
                self.sessions.remove(session_id);
                return None;
            }
            return Some(session);
        }
        None
    }

    /// Remove session
    pub fn remove_session(&mut self, session_id: &str) {
        self.sessions.remove(session_id);
    }

    /// Cleanup expired sessions
    pub fn cleanup_expired(&mut self) -> usize {
        let expired: Vec<String> = self
            .sessions
            .iter()
            .filter(|(_, session)| session.is_expired())
            .map(|(id, _)| id.clone())
            .collect();

        let count = expired.len();
        for id in expired {
            self.sessions.remove(&id);
        }
        count
    }

    /// Count active sessions
    pub fn count(&mut self) -> usize {
        self.cleanup_expired();
        self.sessions.len()
    }

    /// Clear all sessions
    pub fn clear(&mut self) {
        self.sessions.clear();
    }
}

impl Default for SessionManager {
    fn default() -> Self {
        Self::new(100)
    }
}
