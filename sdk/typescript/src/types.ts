/**
 * SAGE TypeScript SDK - Type Definitions
 */

export type KeyType = 'Ed25519' | 'Secp256k1' | 'X25519';

export interface KeyPair {
  publicKey: Uint8Array;
  privateKey: Uint8Array;
  type: KeyType;
}

export interface DID {
  did: string;
  method: string;
  identifier: string;
}

export interface Session {
  id: string;
  clientPublicKey: Uint8Array;
  serverPublicKey: Uint8Array;
  createdAt: Date;
  expiresAt: Date;
  metadata?: Record<string, unknown>;
}

export interface HandshakeInitiation {
  clientDID: string;
  clientEphemeralKey: Uint8Array;
  serverPublicKey: Uint8Array;
  timestamp: Date;
  signature: Uint8Array;
}

export interface HandshakeResponse {
  serverDID: string;
  serverEphemeralKey: Uint8Array;
  sessionID: string;
  timestamp: Date;
  signature: Uint8Array;
}

export interface HandshakeCompletion {
  sessionID: string;
  clientSignature: Uint8Array;
}

export interface EncryptedMessage {
  ciphertext: Uint8Array;
  nonce: Uint8Array;
  tag: Uint8Array;
}

export interface SignedMessage {
  message: Uint8Array;
  signature: Uint8Array;
  publicKey: Uint8Array;
  algorithm: string;
}

export interface HTTPSignature {
  signature: string;
  signatureInput: string;
  keyId: string;
}

export interface SAGEConfig {
  blockchainProvider?: string;
  registryAddress?: string;
  network?: 'local' | 'sepolia' | 'mainnet';
  sessionMaxAge?: number;
  sessionIdleTimeout?: number;
}

export interface BlockchainProvider {
  getPublicKey(did: string): Promise<Uint8Array>;
  registerPublicKey(did: string, publicKey: Uint8Array): Promise<string>;
  revokePublicKey(did: string): Promise<string>;
  isRevoked(did: string): Promise<boolean>;
}

export interface SessionManager {
  create(clientKey: Uint8Array, serverKey: Uint8Array): Promise<Session>;
  get(sessionID: string): Promise<Session | null>;
  delete(sessionID: string): Promise<void>;
  encrypt(sessionID: string, plaintext: Uint8Array): Promise<EncryptedMessage>;
  decrypt(sessionID: string, encrypted: EncryptedMessage): Promise<Uint8Array>;
}

export interface CryptoProvider {
  generateKeyPair(type: KeyType): Promise<KeyPair>;
  sign(message: Uint8Array, privateKey: Uint8Array, type: KeyType): Promise<Uint8Array>;
  verify(message: Uint8Array, signature: Uint8Array, publicKey: Uint8Array, type: KeyType): Promise<boolean>;
  deriveSessionKeys(
    myPrivateKey: Uint8Array,
    peerPublicKey: Uint8Array
  ): Promise<{ encryptKey: Uint8Array; decryptKey: Uint8Array }>;
}

export interface DIDResolver {
  resolve(did: string): Promise<DIDDocument>;
  register(didDocument: DIDDocument): Promise<string>;
}

export interface DIDDocument {
  id: string;
  verificationMethod: VerificationMethod[];
  authentication: string[];
  assertionMethod?: string[];
  keyAgreement?: string[];
  service?: ServiceEndpoint[];
}

export interface VerificationMethod {
  id: string;
  type: string;
  controller: string;
  publicKeyJwk?: Record<string, unknown>;
  publicKeyMultibase?: string;
}

export interface ServiceEndpoint {
  id: string;
  type: string;
  serviceEndpoint: string;
}

export type EventType =
  | 'handshake:initiated'
  | 'handshake:completed'
  | 'session:created'
  | 'session:expired'
  | 'message:sent'
  | 'message:received'
  | 'error';

export interface SAGEEvent {
  type: EventType;
  timestamp: Date;
  data: unknown;
}

export type EventHandler = (event: SAGEEvent) => void;

export interface SAGEClientOptions {
  config?: SAGEConfig;
  cryptoProvider?: CryptoProvider;
  sessionManager?: SessionManager;
  blockchainProvider?: BlockchainProvider;
  didResolver?: DIDResolver;
}
