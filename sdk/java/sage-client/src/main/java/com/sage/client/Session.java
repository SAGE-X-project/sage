package com.sage.client;

import java.time.Instant;

/**
 * Represents a secure session between client and server
 */
public class Session {
    private final String sessionId;
    private final String clientDid;
    private final String serverDid;
    private final Crypto.HpkeContext hpkeContext;
    private final Instant createdAt;
    private final Instant expiresAt;
    private Instant lastActivity;
    private long messageCount;

    public Session(String sessionId, String clientDid, String serverDid,
                   Crypto.HpkeContext hpkeContext, long maxAgeSeconds) {
        this.sessionId = sessionId;
        this.clientDid = clientDid;
        this.serverDid = serverDid;
        this.hpkeContext = hpkeContext;
        this.createdAt = Instant.now();
        this.expiresAt = this.createdAt.plusSeconds(maxAgeSeconds);
        this.lastActivity = this.createdAt;
        this.messageCount = 0;
    }

    /**
     * Check if session is expired
     */
    public boolean isExpired() {
        return Instant.now().isAfter(expiresAt);
    }

    /**
     * Update last activity timestamp
     */
    public void updateActivity() throws SageException.SessionExpiredException {
        if (isExpired()) {
            throw new SageException.SessionExpiredException(sessionId);
        }
        this.lastActivity = Instant.now();
    }

    /**
     * Encrypt message using session context
     */
    public byte[] encrypt(byte[] plaintext) throws SageException {
        if (isExpired()) {
            throw new SageException.SessionExpiredException(sessionId);
        }
        updateActivity();
        messageCount++;
        return hpkeContext.seal(plaintext);
    }

    /**
     * Decrypt message using session context
     */
    public byte[] decrypt(byte[] ciphertext) throws SageException {
        if (isExpired()) {
            throw new SageException.SessionExpiredException(sessionId);
        }
        updateActivity();
        return hpkeContext.open(ciphertext);
    }

    // Getters
    public String getSessionId() {
        return sessionId;
    }

    public String getClientDid() {
        return clientDid;
    }

    public String getServerDid() {
        return serverDid;
    }

    public Crypto.HpkeContext getHpkeContext() {
        return hpkeContext;
    }

    public Instant getCreatedAt() {
        return createdAt;
    }

    public Instant getExpiresAt() {
        return expiresAt;
    }

    public Instant getLastActivity() {
        return lastActivity;
    }

    public long getMessageCount() {
        return messageCount;
    }
}
