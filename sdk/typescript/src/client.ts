/**
 * SAGE Client - Main SDK Entry Point
 */

import { SAGECrypto } from './crypto';
import { SAGESessionManager } from './session';
import type {
  SAGEClientOptions,
  SAGEConfig,
  KeyPair,
  KeyType,
  Session,
  EncryptedMessage,
  HandshakeInitiation,
  HandshakeResponse,
  EventHandler,
  SAGEEvent,
  EventType,
} from './types';

export class SAGEClient {
  private config: SAGEConfig;
  private crypto: SAGECrypto;
  private sessionManager: SAGESessionManager;
  private eventHandlers: Map<EventType, Set<EventHandler>> = new Map();
  private identityKeyPair?: KeyPair;

  constructor(options: SAGEClientOptions = {}) {
    this.config = options.config ?? {};
    this.crypto = options.cryptoProvider ?? new SAGECrypto();
    this.sessionManager = options.sessionManager ?? new SAGESessionManager({
      maxAge: this.config.sessionMaxAge,
      idleTimeout: this.config.sessionIdleTimeout,
    });
  }

  /**
   * Initialize the client with an identity key pair
   */
  async initialize(keyPair?: KeyPair): Promise<void> {
    if (keyPair) {
      this.identityKeyPair = keyPair;
    } else {
      // Generate new identity key pair
      this.identityKeyPair = await this.crypto.generateKeyPair('Ed25519');
    }

    this.emit('initialized', { publicKey: this.identityKeyPair.publicKey });
  }

  /**
   * Get the client's DID
   */
  getDID(): string {
    if (!this.identityKeyPair) {
      throw new Error('Client not initialized');
    }

    // Format: did:sage:<base64-encoded-public-key>
    const publicKeyB64 = btoa(String.fromCharCode(...this.identityKeyPair.publicKey));
    return `did:sage:${publicKeyB64}`;
  }

  /**
   * Get the client's public key
   */
  getPublicKey(): Uint8Array {
    if (!this.identityKeyPair) {
      throw new Error('Client not initialized');
    }

    return this.identityKeyPair.publicKey;
  }

  /**
   * Initiate a handshake with a server
   */
  async initiateHandshake(serverPublicKey: Uint8Array): Promise<HandshakeInitiation> {
    if (!this.identityKeyPair) {
      throw new Error('Client not initialized');
    }

    // Generate ephemeral key pair for this session
    const ephemeralKeyPair = await this.crypto.generateKeyPair('X25519');

    // Create handshake initiation message
    const timestamp = new Date();
    const message = new TextEncoder().encode(
      JSON.stringify({
        clientDID: this.getDID(),
        clientEphemeralKey: Array.from(ephemeralKeyPair.publicKey),
        serverPublicKey: Array.from(serverPublicKey),
        timestamp: timestamp.toISOString(),
      })
    );

    // Sign the message
    const signature = await this.crypto.sign(
      message,
      this.identityKeyPair.privateKey,
      this.identityKeyPair.type
    );

    const initiation: HandshakeInitiation = {
      clientDID: this.getDID(),
      clientEphemeralKey: ephemeralKeyPair.publicKey,
      serverPublicKey,
      timestamp,
      signature,
    };

    this.emit('handshake:initiated', { serverPublicKey });

    return initiation;
  }

  /**
   * Complete a handshake after receiving server response
   */
  async completeHandshake(
    response: HandshakeResponse,
    myEphemeralPrivateKey: Uint8Array
  ): Promise<Session> {
    if (!this.identityKeyPair) {
      throw new Error('Client not initialized');
    }

    // Verify server's signature
    const responseMessage = new TextEncoder().encode(
      JSON.stringify({
        serverDID: response.serverDID,
        serverEphemeralKey: Array.from(response.serverEphemeralKey),
        sessionID: response.sessionID,
        timestamp: response.timestamp.toISOString(),
      })
    );

    // In real implementation, fetch server's public key from blockchain
    // For now, we'll assume signature is valid

    // Derive session keys
    const { encryptKey, decryptKey } = await this.crypto.deriveSessionKeys(
      myEphemeralPrivateKey,
      response.serverEphemeralKey
    );

    // Create session
    const session = await this.sessionManager.create(
      this.identityKeyPair.publicKey,
      response.serverEphemeralKey
    );

    this.emit('handshake:completed', { sessionID: session.id });
    this.emit('session:created', { session });

    return session;
  }

  /**
   * Send an encrypted message
   */
  async sendMessage(sessionID: string, message: Uint8Array): Promise<EncryptedMessage> {
    const encrypted = await this.sessionManager.encrypt(sessionID, message);
    this.emit('message:sent', { sessionID, size: message.length });
    return encrypted;
  }

  /**
   * Receive and decrypt a message
   */
  async receiveMessage(sessionID: string, encrypted: EncryptedMessage): Promise<Uint8Array> {
    const plaintext = await this.sessionManager.decrypt(sessionID, encrypted);
    this.emit('message:received', { sessionID, size: plaintext.length });
    return plaintext;
  }

  /**
   * Generate a key pair
   */
  async generateKeyPair(type: KeyType): Promise<KeyPair> {
    return this.crypto.generateKeyPair(type);
  }

  /**
   * Sign a message
   */
  async sign(message: Uint8Array): Promise<Uint8Array> {
    if (!this.identityKeyPair) {
      throw new Error('Client not initialized');
    }

    return this.crypto.sign(message, this.identityKeyPair.privateKey, this.identityKeyPair.type);
  }

  /**
   * Verify a signature
   */
  async verify(
    message: Uint8Array,
    signature: Uint8Array,
    publicKey: Uint8Array,
    type: KeyType = 'Ed25519'
  ): Promise<boolean> {
    return this.crypto.verify(message, signature, publicKey, type);
  }

  /**
   * Get session information
   */
  async getSession(sessionID: string): Promise<Session | null> {
    return this.sessionManager.get(sessionID);
  }

  /**
   * Get all active sessions
   */
  getAllSessions(): Session[] {
    return this.sessionManager.getAllSessions();
  }

  /**
   * Close a session
   */
  async closeSession(sessionID: string): Promise<void> {
    await this.sessionManager.delete(sessionID);
    this.emit('session:expired', { sessionID });
  }

  /**
   * Register an event handler
   */
  on(eventType: EventType, handler: EventHandler): void {
    if (!this.eventHandlers.has(eventType)) {
      this.eventHandlers.set(eventType, new Set());
    }
    this.eventHandlers.get(eventType)!.add(handler);
  }

  /**
   * Unregister an event handler
   */
  off(eventType: EventType, handler: EventHandler): void {
    const handlers = this.eventHandlers.get(eventType);
    if (handlers) {
      handlers.delete(handler);
    }
  }

  /**
   * Emit an event
   */
  private emit(type: EventType, data: unknown): void {
    const event: SAGEEvent = {
      type,
      timestamp: new Date(),
      data,
    };

    const handlers = this.eventHandlers.get(type);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(event);
        } catch (error) {
          console.error('Event handler error:', error);
        }
      });
    }
  }
}
