/**
 * SAGE TypeScript SDK
 *
 * @packageDocumentation
 */

// Core client
export { SAGEClient } from './client';
export { SAGECrypto, utils as cryptoUtils } from './crypto';
export { SAGESessionManager } from './session';

// Types
export type {
  KeyType,
  KeyPair,
  DID,
  Session,
  HandshakeInitiation,
  HandshakeResponse,
  HandshakeCompletion,
  EncryptedMessage,
  SignedMessage,
  HTTPSignature,
  SAGEConfig,
  SAGEClientOptions,
  BlockchainProvider,
  SessionManager,
  CryptoProvider,
  DIDResolver,
  DIDDocument,
  VerificationMethod,
  ServiceEndpoint,
  EventType,
  SAGEEvent,
  EventHandler,
} from './types';

// React hooks (optional peer dependency)
export {
  useSAGE,
  useSessions,
  useSecureMessaging,
  useSAGEEvents,
  useHandshake,
  useCrypto,
} from './react/hooks';

// Version
export const VERSION = '1.0.0';
