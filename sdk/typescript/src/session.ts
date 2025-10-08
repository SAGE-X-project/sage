/**
 * SAGE Session Management
 */

import { SAGECrypto } from './crypto';
import type { Session, SessionManager, EncryptedMessage } from './types';

export interface SessionOptions {
  maxAge?: number; // milliseconds
  idleTimeout?: number; // milliseconds
}

export class SAGESessionManager implements SessionManager {
  private sessions: Map<string, SessionData> = new Map();
  private crypto: SAGECrypto;
  private options: Required<SessionOptions>;

  constructor(options: SessionOptions = {}) {
    this.crypto = new SAGECrypto();
    this.options = {
      maxAge: options.maxAge ?? 3600000, // 1 hour
      idleTimeout: options.idleTimeout ?? 600000, // 10 minutes
    };

    // Start cleanup interval
    this.startCleanup();
  }

  /**
   * Create a new session
   */
  async create(clientKey: Uint8Array, serverKey: Uint8Array): Promise<Session> {
    // Generate session ID
    const sessionID = this.generateSessionID();

    // Derive session keys (this would use X25519 private key in real implementation)
    const now = new Date();
    const expiresAt = new Date(now.getTime() + this.options.maxAge);

    const session: Session = {
      id: sessionID,
      clientPublicKey: clientKey,
      serverPublicKey: serverKey,
      createdAt: now,
      expiresAt,
    };

    // Store session with encryption keys
    // In real implementation, derive keys using X25519 + HKDF
    const sessionData: SessionData = {
      session,
      encryptKey: new Uint8Array(32), // Placeholder
      decryptKey: new Uint8Array(32), // Placeholder
      lastAccessTime: now,
    };

    this.sessions.set(sessionID, sessionData);

    return session;
  }

  /**
   * Get an existing session
   */
  async get(sessionID: string): Promise<Session | null> {
    const sessionData = this.sessions.get(sessionID);

    if (!sessionData) {
      return null;
    }

    // Check expiration
    if (this.isExpired(sessionData)) {
      this.sessions.delete(sessionID);
      return null;
    }

    // Update last access time
    sessionData.lastAccessTime = new Date();

    return sessionData.session;
  }

  /**
   * Delete a session
   */
  async delete(sessionID: string): Promise<void> {
    this.sessions.delete(sessionID);
  }

  /**
   * Encrypt a message using session keys
   */
  async encrypt(sessionID: string, plaintext: Uint8Array): Promise<EncryptedMessage> {
    const sessionData = this.sessions.get(sessionID);

    if (!sessionData) {
      throw new Error('Session not found');
    }

    if (this.isExpired(sessionData)) {
      this.sessions.delete(sessionID);
      throw new Error('Session expired');
    }

    // Update last access time
    sessionData.lastAccessTime = new Date();

    // Encrypt using session encrypt key
    const { ciphertext, nonce, tag } = await this.crypto.encrypt(
      plaintext,
      sessionData.encryptKey
    );

    return { ciphertext, nonce, tag };
  }

  /**
   * Decrypt a message using session keys
   */
  async decrypt(sessionID: string, encrypted: EncryptedMessage): Promise<Uint8Array> {
    const sessionData = this.sessions.get(sessionID);

    if (!sessionData) {
      throw new Error('Session not found');
    }

    if (this.isExpired(sessionData)) {
      this.sessions.delete(sessionID);
      throw new Error('Session expired');
    }

    // Update last access time
    sessionData.lastAccessTime = new Date();

    // Decrypt using session decrypt key
    const plaintext = await this.crypto.decrypt(
      encrypted.ciphertext,
      encrypted.nonce,
      encrypted.tag,
      sessionData.decryptKey
    );

    return plaintext;
  }

  /**
   * Get all active sessions
   */
  getAllSessions(): Session[] {
    const sessions: Session[] = [];

    for (const [_, sessionData] of this.sessions) {
      if (!this.isExpired(sessionData)) {
        sessions.push(sessionData.session);
      }
    }

    return sessions;
  }

  /**
   * Clear all sessions
   */
  clearAll(): void {
    this.sessions.clear();
  }

  // Private helper methods

  private generateSessionID(): string {
    const bytes = crypto.getRandomValues(new Uint8Array(32));
    return Array.from(bytes)
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
  }

  private isExpired(sessionData: SessionData): boolean {
    const now = new Date();

    // Check max age
    if (now > sessionData.session.expiresAt) {
      return true;
    }

    // Check idle timeout
    const idleDuration = now.getTime() - sessionData.lastAccessTime.getTime();
    if (idleDuration > this.options.idleTimeout) {
      return true;
    }

    return false;
  }

  private startCleanup(): void {
    // Cleanup expired sessions every 30 seconds
    setInterval(() => {
      this.cleanup();
    }, 30000);
  }

  private cleanup(): void {
    const expiredSessions: string[] = [];

    for (const [sessionID, sessionData] of this.sessions) {
      if (this.isExpired(sessionData)) {
        expiredSessions.push(sessionID);
      }
    }

    for (const sessionID of expiredSessions) {
      this.sessions.delete(sessionID);
    }
  }
}

interface SessionData {
  session: Session;
  encryptKey: Uint8Array;
  decryptKey: Uint8Array;
  lastAccessTime: Date;
}
