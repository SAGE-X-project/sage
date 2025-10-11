# SAGE TypeScript SDK

Official TypeScript/JavaScript SDK for SAGE (Secure Agent Guarantee Engine).

## Features

- 🔐 **End-to-End Encryption**: Secure agent-to-agent communication
- 🔑 **Cryptographic Operations**: Ed25519, Secp256k1, X25519 key management
- 🤝 **Handshake Protocol**: Secure session establishment with DID verification
- 📝 **Message Signing**: RFC 9421 HTTP message signatures
- ⚛️ **React Hooks**: Ready-to-use hooks for React applications
- 🌐 **Blockchain Integration**: Ethereum/EVM-compatible DID registry
- 📦 **Zero Dependencies**: Core crypto operations use well-audited libraries
- 🎯 **TypeScript First**: Full type safety and IntelliSense support

## Installation

```bash
npm install @sage-x/sdk
```

For React applications:

```bash
npm install @sage-x/sdk react
```

## Quick Start

### Basic Usage

```typescript
import { SAGEClient } from '@sage-x/sdk';

// Initialize client
const client = new SAGEClient();
await client.initialize();

// Get DID
const did = client.getDID();
console.log('My DID:', did);

// Generate key pair
const keyPair = await client.generateKeyPair('Ed25519');

// Sign a message
const message = new TextEncoder().encode('Hello SAGE!');
const signature = await client.sign(message);

// Verify signature
const isValid = await client.verify(message, signature, keyPair.publicKey);
```

### Secure Messaging

```typescript
import { SAGEClient } from '@sage-x/sdk';

const alice = new SAGEClient();
const bob = new SAGEClient();

await alice.initialize();
await bob.initialize();

// Initiate handshake
const initiation = await alice.initiateHandshake(bob.getPublicKey());

// Complete handshake (simplified)
// In production, server would process and respond
const session = await alice.completeHandshake(response, ephemeralPrivateKey);

// Send encrypted message
const plaintext = new TextEncoder().encode('Secret message');
const encrypted = await alice.sendMessage(session.id, plaintext);

// Receive and decrypt
const decrypted = await bob.receiveMessage(session.id, encrypted);
const message = new TextDecoder().decode(decrypted);
```

### React Hooks

```tsx
import { useSAGE, useSessions, useSecureMessaging } from '@sage-x/sdk';

function MyComponent() {
  const { client, isInitialized, did, initialize } = useSAGE();
  const { sessions, createSession, closeSession } = useSessions(client);
  const { sendMessage, receiveMessage } = useSecureMessaging(client, sessionID);

  useEffect(() => {
    initialize();
  }, []);

  const handleSend = async () => {
    const encrypted = await sendMessage('Hello!');
    // Send encrypted to peer
  };

  return (
    <div>
      <p>DID: {did}</p>
      <p>Active Sessions: {sessions.length}</p>
      <button onClick={handleSend}>Send Message</button>
    </div>
  );
}
```

## API Reference

### SAGEClient

Main client for SAGE operations.

#### Constructor

```typescript
const client = new SAGEClient(options?: SAGEClientOptions);
```

**Options:**
- `config`: SAGE configuration
- `cryptoProvider`: Custom crypto provider
- `sessionManager`: Custom session manager
- `blockchainProvider`: Blockchain provider for DID registry

#### Methods

##### initialize(keyPair?: KeyPair): Promise<void>

Initialize the client with an identity key pair.

```typescript
await client.initialize();
// or with existing key pair
await client.initialize(myKeyPair);
```

##### getDID(): string

Get the client's DID.

```typescript
const did = client.getDID();
// Returns: "did:sage:base64encodedpublickey"
```

##### getPublicKey(): Uint8Array

Get the client's public key.

```typescript
const publicKey = client.getPublicKey();
```

##### generateKeyPair(type: KeyType): Promise<KeyPair>

Generate a new key pair.

```typescript
const keyPair = await client.generateKeyPair('Ed25519');
// Types: 'Ed25519' | 'Secp256k1' | 'X25519'
```

##### sign(message: Uint8Array): Promise<Uint8Array>

Sign a message with the identity key.

```typescript
const message = new TextEncoder().encode('Hello');
const signature = await client.sign(message);
```

##### verify(message: Uint8Array, signature: Uint8Array, publicKey: Uint8Array, type?: KeyType): Promise<boolean>

Verify a signature.

```typescript
const isValid = await client.verify(message, signature, publicKey);
```

##### initiateHandshake(serverPublicKey: Uint8Array): Promise<HandshakeInitiation>

Initiate a handshake with a server.

```typescript
const initiation = await client.initiateHandshake(serverPublicKey);
```

##### completeHandshake(response: HandshakeResponse, myEphemeralPrivateKey: Uint8Array): Promise<Session>

Complete a handshake after receiving server response.

```typescript
const session = await client.completeHandshake(response, ephemeralKey);
```

##### sendMessage(sessionID: string, message: Uint8Array): Promise<EncryptedMessage>

Send an encrypted message.

```typescript
const encrypted = await client.sendMessage(sessionID, plaintext);
```

##### receiveMessage(sessionID: string, encrypted: EncryptedMessage): Promise<Uint8Array>

Receive and decrypt a message.

```typescript
const plaintext = await client.receiveMessage(sessionID, encrypted);
```

##### on(eventType: EventType, handler: EventHandler): void

Register an event handler.

```typescript
client.on('session:created', (event) => {
  console.log('Session created:', event.data);
});
```

##### off(eventType: EventType, handler: EventHandler): void

Unregister an event handler.

```typescript
client.off('session:created', handler);
```

### React Hooks

#### useSAGE(options?: SAGEClientOptions)

Main hook for SAGE client.

```typescript
const { client, isInitialized, did, error, initialize } = useSAGE();
```

**Returns:**
- `client`: SAGEClient instance
- `isInitialized`: Boolean indicating if client is ready
- `did`: Client's DID
- `error`: Error if initialization failed
- `initialize`: Function to initialize client

#### useSessions(client: SAGEClient)

Manage sessions.

```typescript
const {
  sessions,
  loading,
  error,
  createSession,
  closeSession,
  refreshSessions
} = useSessions(client);
```

**Returns:**
- `sessions`: Array of active sessions
- `loading`: Boolean indicating loading state
- `error`: Error if operation failed
- `createSession`: Function to create new session
- `closeSession`: Function to close session
- `refreshSessions`: Function to refresh session list

#### useSecureMessaging(client: SAGEClient, sessionID: string)

Send and receive encrypted messages.

```typescript
const {
  sendMessage,
  receiveMessage,
  sending,
  receiving,
  error
} = useSecureMessaging(client, sessionID);
```

**Returns:**
- `sendMessage`: Function to send encrypted message
- `receiveMessage`: Function to receive and decrypt message
- `sending`: Boolean indicating sending state
- `receiving`: Boolean indicating receiving state
- `error`: Error if operation failed

#### useSAGEEvents(client: SAGEClient, eventType: EventType, handler: EventHandler)

Listen to SAGE events.

```typescript
useSAGEEvents(client, 'message:sent', (event) => {
  console.log('Message sent:', event);
});
```

#### useHandshake(client: SAGEClient)

Manage handshake operations.

```typescript
const {
  initiateHandshake,
  completeHandshake,
  loading,
  error,
  session
} = useHandshake(client);
```

#### useCrypto(client: SAGEClient)

Cryptographic operations.

```typescript
const {
  generateKeyPair,
  sign,
  verify,
  loading,
  error
} = useCrypto(client);
```

## Types

### KeyType

```typescript
type KeyType = 'Ed25519' | 'Secp256k1' | 'X25519';
```

### KeyPair

```typescript
interface KeyPair {
  publicKey: Uint8Array;
  privateKey: Uint8Array;
  type: KeyType;
}
```

### Session

```typescript
interface Session {
  id: string;
  clientPublicKey: Uint8Array;
  serverPublicKey: Uint8Array;
  createdAt: Date;
  expiresAt: Date;
  metadata?: Record<string, unknown>;
}
```

### EncryptedMessage

```typescript
interface EncryptedMessage {
  ciphertext: Uint8Array;
  nonce: Uint8Array;
  tag: Uint8Array;
}
```

### EventType

```typescript
type EventType =
  | 'handshake:initiated'
  | 'handshake:completed'
  | 'session:created'
  | 'session:expired'
  | 'message:sent'
  | 'message:received'
  | 'error';
```

## Examples

### MCP Chat Example

See [examples/mcp-chat](./examples/mcp-chat) for a complete agent-to-agent chat example.

```bash
cd examples/mcp-chat
npm install
npm start
```

### React App Example

See [examples/react-app](./examples/react-app) for a React application using SAGE hooks.

```bash
cd examples/react-app
npm install
npm start
```

## Configuration

### SAGEConfig

```typescript
interface SAGEConfig {
  blockchainProvider?: string;
  registryAddress?: string;
  network?: 'local' | 'sepolia' | 'mainnet';
  sessionMaxAge?: number; // milliseconds
  sessionIdleTimeout?: number; // milliseconds
}
```

Example:

```typescript
const client = new SAGEClient({
  config: {
    network: 'sepolia',
    registryAddress: '0x...',
    sessionMaxAge: 3600000, // 1 hour
    sessionIdleTimeout: 600000, // 10 minutes
  }
});
```

## Utilities

### cryptoUtils

Utility functions for encoding/decoding.

```typescript
import { cryptoUtils } from '@sage-x/sdk';

// Hex conversion
const bytes = cryptoUtils.hexToBytes('0x1234...');
const hex = cryptoUtils.bytesToHex(bytes);

// Base64 conversion
const bytes = cryptoUtils.base64ToBytes('SGVsbG8=');
const base64 = cryptoUtils.bytesToBase64(bytes);

// Random bytes
const random = cryptoUtils.randomBytes(32);
```

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Node.js 18+

Requires WebCrypto API support.

## Security

### Cryptographic Primitives

- **Ed25519**: Digital signatures (via @noble/ed25519)
- **Secp256k1**: Ethereum-compatible signatures (via @noble/secp256k1)
- **X25519**: Key agreement (via @noble/curves)
- **AES-GCM**: Symmetric encryption (via WebCrypto)
- **HKDF**: Key derivation (via @noble/hashes)

### Security Best Practices

1. **Key Storage**: Store private keys securely (not in localStorage)
2. **Session Management**: Use appropriate timeouts
3. **Nonce Validation**: Prevent replay attacks
4. **TLS**: Always use HTTPS in production
5. **DID Verification**: Verify DIDs against blockchain registry

## Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run tests
npm test

# Type check
npm run typecheck

# Lint
npm run lint
```

## License

MIT

## Links

- [Documentation](https://github.com/sage-x-project/sage/tree/main/sdk/typescript)
- [GitHub](https://github.com/sage-x-project/sage)
- [Issues](https://github.com/sage-x-project/sage/issues)
- [Main SAGE Documentation](https://github.com/sage-x-project/sage)

## Support

For questions and support:
- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://github.com/sage-x-project/sage/tree/main/docs
