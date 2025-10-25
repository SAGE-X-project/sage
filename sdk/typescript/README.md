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

## Troubleshooting

### Common Issues

#### TypeScript Compilation Errors

**Problem:** `Cannot find module '@sage-x/sdk'` or type errors

**Solutions:**
```typescript
// 1. Ensure package is installed
npm install @sage-x/sdk

// 2. Check tsconfig.json
{
  "compilerOptions": {
    "target": "ES2020",
    "lib": ["ES2020", "DOM"],
    "moduleResolution": "node",
    "esModuleInterop": true
  }
}

// 3. Restart TypeScript server (VS Code)
// Cmd/Ctrl + Shift + P → "Restart TS Server"
```

#### WebCrypto Not Available

**Problem:** `TypeError: crypto.subtle is undefined` in older browsers

**Solutions:**
```typescript
// 1. Check browser compatibility
if (!window.crypto || !window.crypto.subtle) {
  throw new Error('WebCrypto API not supported. Upgrade browser.');
}

// 2. Use polyfill for Node.js < 15
import { webcrypto } from 'crypto';
globalThis.crypto = webcrypto as Crypto;

// 3. Ensure HTTPS (WebCrypto requires secure context)
// Works on localhost and https:// only
```

#### React Hook Errors

**Problem:** `Error: Hooks can only be called inside function components`

**Solutions:**
```typescript
// ❌ BAD - Using hooks outside component
const client = useSAGE();

export function MyComponent() {
  return <div>...</div>;
}

// ✅ GOOD - Use hooks inside component
export function MyComponent() {
  const { client, isInitialized } = useSAGE();
  return <div>...</div>;
}

// ✅ GOOD - Use class component with HOC
class MyComponent extends React.Component {
  // Use withSAGE HOC instead
}
```

#### Session Not Found Errors

**Problem:** `Error: Session not found or expired`

**Solutions:**
```typescript
// 1. Check session expiration before use
const session = client.getSession(sessionID);
if (!session || new Date() > session.expiresAt) {
  // Re-establish handshake
  const newSession = await client.initiateHandshake(serverPublicKey);
  sessionID = newSession.id;
}

// 2. Handle expiration gracefully with retry
async function sendWithRetry(sessionID: string, message: Uint8Array, retries = 1) {
  try {
    return await client.sendMessage(sessionID, message);
  } catch (error) {
    if (error.message.includes('Session') && retries > 0) {
      // Re-establish and retry
      const newSession = await client.initiateHandshake(serverPublicKey);
      return sendWithRetry(newSession.id, message, retries - 1);
    }
    throw error;
  }
}

// 3. Monitor session expiration with useEffect
useEffect(() => {
  const interval = setInterval(() => {
    const session = client.getSession(sessionID);
    if (session && new Date() > session.expiresAt) {
      console.warn('Session expired, re-establishing...');
      handleReconnect();
    }
  }, 30000); // Check every 30 seconds

  return () => clearInterval(interval);
}, [sessionID]);
```

#### Memory Leaks in React

**Problem:** Memory leaks from event listeners or unclosed sessions

**Solutions:**
```typescript
// ✅ Clean up event listeners
useEffect(() => {
  const handler = (event) => console.log('Session created:', event);
  client.on('session:created', handler);

  return () => {
    client.off('session:created', handler);
  };
}, [client]);

// ✅ Close sessions on unmount
useEffect(() => {
  return () => {
    sessions.forEach(session => client.closeSession(session.id));
  };
}, [sessions]);

// ✅ Use cleanup in custom hooks
function useSession(clientDID: string) {
  const [session, setSession] = useState<Session | null>(null);

  useEffect(() => {
    let active = true;

    async function establish() {
      const s = await client.initiateHandshake(clientDID);
      if (active) setSession(s);
    }

    establish();

    return () => {
      active = false;
      if (session) client.closeSession(session.id);
    };
  }, [clientDID]);

  return session;
}
```

#### Build Errors with Webpack/Vite

**Problem:** Build fails with module not found or crypto errors

**Solutions:**
```javascript
// Vite: Add to vite.config.js
export default {
  resolve: {
    alias: {
      crypto: 'crypto-browserify',
      stream: 'stream-browserify',
    }
  },
  define: {
    global: 'globalThis'
  }
}

// Webpack: Add to webpack.config.js
module.exports = {
  resolve: {
    fallback: {
      crypto: require.resolve('crypto-browserify'),
      stream: require.resolve('stream-browserify'),
    }
  }
}

// Next.js: Add to next.config.js
module.exports = {
  webpack: (config) => {
    config.resolve.fallback = {
      ...config.resolve.fallback,
      crypto: require.resolve('crypto-browserify'),
    };
    return config;
  }
}
```

### Debug Mode

Enable verbose logging for troubleshooting:

```typescript
// Enable debug logging (development only)
const client = new SAGEClient({
  debug: true, // Logs all operations
  logLevel: 'debug' // 'error' | 'warn' | 'info' | 'debug'
});

// Custom logger
const client = new SAGEClient({
  logger: {
    error: (...args) => console.error('[SAGE]', ...args),
    warn: (...args) => console.warn('[SAGE]', ...args),
    info: (...args) => console.info('[SAGE]', ...args),
    debug: (...args) => console.debug('[SAGE]', ...args),
  }
});
```

### Performance Issues

**Problem:** Slow handshake or message operations

**Diagnostics:**
```typescript
// Measure operation times
async function benchmark() {
  console.time('Initialize');
  await client.initialize();
  console.timeEnd('Initialize'); // Expected: 50-100ms

  console.time('Handshake');
  const session = await client.initiateHandshake(serverPublicKey);
  console.timeEnd('Handshake'); // Expected: 50-200ms

  console.time('Send Message');
  await client.sendMessage(session.id, new TextEncoder().encode('test'));
  console.timeEnd('Send Message'); // Expected: 20-100ms
}
```

**Solutions:**
- Use session pooling for multiple messages
- Batch operations when possible
- Use Web Workers for crypto operations (future enhancement)
- Enable HTTP/2 for better performance

---

## Best Practices

### Security

#### 1. Never Expose Private Keys

```typescript
// ❌ BAD - Logging private keys
console.log('Private key:', keyPair.privateKey);

// ❌ BAD - Storing in localStorage
localStorage.setItem('privateKey', bytesToHex(keyPair.privateKey));

// ✅ GOOD - Use secure storage
// Browser: IndexedDB with encryption
import { openDB } from 'idb';

const db = await openDB('sage-keys', 1, {
  upgrade(db) {
    db.createObjectStore('keys');
  }
});

await db.put('keys', keyPair.privateKey, 'identity');

// ✅ GOOD - Node.js: Use OS keychain
// macOS: Keychain, Windows: Credential Manager, Linux: Secret Service
```

#### 2. Validate All Inputs

```typescript
// ✅ Validate DIDs
function validateDID(did: string): boolean {
  const didRegex = /^did:sage:(ethereum|solana):0x[a-fA-F0-9]{40}$/;
  return didRegex.test(did);
}

function processDID(did: string) {
  if (!validateDID(did)) {
    throw new Error('Invalid DID format');
  }
  // Proceed with valid DID
}

// ✅ Validate message size
const MAX_MESSAGE_SIZE = 1024 * 1024; // 1MB

function sendMessage(sessionID: string, message: Uint8Array) {
  if (message.length > MAX_MESSAGE_SIZE) {
    throw new Error(`Message too large: ${message.length} bytes`);
  }
  return client.sendMessage(sessionID, message);
}
```

#### 3. Use Timeouts for Network Operations

```typescript
// ✅ Add timeout wrapper
function withTimeout<T>(promise: Promise<T>, ms: number): Promise<T> {
  return Promise.race([
    promise,
    new Promise<T>((_, reject) =>
      setTimeout(() => reject(new Error('Timeout')), ms)
    )
  ]);
}

// Usage
try {
  const session = await withTimeout(
    client.initiateHandshake(serverPublicKey),
    5000 // 5 second timeout
  );
} catch (error) {
  if (error.message === 'Timeout') {
    console.error('Handshake timeout - server may be down');
  }
}
```

#### 4. Implement Exponential Backoff for Retries

```typescript
// ✅ Retry with exponential backoff
async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  maxRetries = 3,
  baseDelay = 1000
): Promise<T> {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (i === maxRetries - 1) throw error;

      const delay = baseDelay * Math.pow(2, i);
      console.log(`Retry ${i + 1}/${maxRetries} after ${delay}ms`);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
  throw new Error('Max retries exceeded');
}

// Usage
const session = await retryWithBackoff(() =>
  client.initiateHandshake(serverPublicKey)
);
```

### Performance

#### 1. Reuse Sessions

```typescript
// ❌ BAD - New session for each message
for (const message of messages) {
  const session = await client.initiateHandshake(serverPublicKey);
  await client.sendMessage(session.id, message);
}

// ✅ GOOD - Reuse session
const session = await client.initiateHandshake(serverPublicKey);
for (const message of messages) {
  await client.sendMessage(session.id, message);
}

// ✅ BETTER - Session pool
class SessionPool {
  private sessions = new Map<string, Session>();

  async getSession(did: string): Promise<Session> {
    let session = this.sessions.get(did);
    if (!session || new Date() > session.expiresAt) {
      session = await client.initiateHandshake(did);
      this.sessions.set(did, session);
    }
    return session;
  }

  cleanup() {
    const now = new Date();
    for (const [did, session] of this.sessions.entries()) {
      if (now > session.expiresAt) {
        this.sessions.delete(did);
      }
    }
  }
}
```

#### 2. Batch Operations

```typescript
// ✅ Send multiple messages in parallel
async function broadcastMessage(dids: string[], message: Uint8Array) {
  // Establish sessions in parallel
  const sessions = await Promise.all(
    dids.map(did => client.initiateHandshake(did))
  );

  // Send messages in parallel
  const results = await Promise.all(
    sessions.map(session => client.sendMessage(session.id, message))
  );

  return results;
}
```

#### 3. Optimize React Rendering

```typescript
// ✅ Use useMemo for expensive computations
const { client } = useSAGE();

const sessionMap = useMemo(() => {
  return sessions.reduce((acc, s) => {
    acc[s.id] = s;
    return acc;
  }, {} as Record<string, Session>);
}, [sessions]);

// ✅ Use useCallback for event handlers
const handleSendMessage = useCallback(async (message: string) => {
  const encoded = new TextEncoder().encode(message);
  await client.sendMessage(sessionID, encoded);
}, [client, sessionID]);

// ✅ Memoize components
const SessionList = React.memo(({ sessions }: { sessions: Session[] }) => {
  return (
    <ul>
      {sessions.map(s => <SessionItem key={s.id} session={s} />)}
    </ul>
  );
});
```

#### 4. Clean Up Resources

```typescript
// ✅ Close client on unmount
useEffect(() => {
  return () => {
    client.close();
  };
}, [client]);

// ✅ Cancel pending operations
useEffect(() => {
  const controller = new AbortController();

  async function loadSessions() {
    try {
      const sessions = await client.getSessions({ signal: controller.signal });
      setSessions(sessions);
    } catch (error) {
      if (error.name === 'AbortError') return;
      console.error(error);
    }
  }

  loadSessions();

  return () => controller.abort();
}, []);
```

### React Best Practices

#### 1. Use Context for Global Client

```typescript
// ✅ Create context
import { createContext, useContext, ReactNode } from 'react';

const SAGEContext = createContext<SAGEClient | null>(null);

export function SAGEProvider({ children }: { children: ReactNode }) {
  const { client, isInitialized } = useSAGE();

  if (!isInitialized) {
    return <div>Initializing SAGE...</div>;
  }

  return (
    <SAGEContext.Provider value={client}>
      {children}
    </SAGEContext.Provider>
  );
}

export function useSAGEClient() {
  const client = useContext(SAGEContext);
  if (!client) throw new Error('useSAGEClient must be used within SAGEProvider');
  return client;
}

// Usage
function App() {
  return (
    <SAGEProvider>
      <MessageComponent />
    </SAGEProvider>
  );
}

function MessageComponent() {
  const client = useSAGEClient();
  // Use client
}
```

#### 2. Separate Business Logic from UI

```typescript
// ✅ Custom hooks for business logic
function useAgentMessaging(targetDID: string) {
  const client = useSAGEClient();
  const [session, setSession] = useState<Session | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const establishSession = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const s = await client.initiateHandshake(targetDID);
      setSession(s);
    } catch (e) {
      setError(e as Error);
    } finally {
      setLoading(false);
    }
  }, [client, targetDID]);

  const sendMessage = useCallback(async (message: string) => {
    if (!session) throw new Error('No active session');
    const encoded = new TextEncoder().encode(message);
    return await client.sendMessage(session.id, encoded);
  }, [client, session]);

  useEffect(() => {
    establishSession();
    return () => {
      if (session) client.closeSession(session.id);
    };
  }, [targetDID]);

  return { session, loading, error, sendMessage, establishSession };
}

// UI component
function ChatComponent({ targetDID }: { targetDID: string }) {
  const { session, sendMessage, loading, error } = useAgentMessaging(targetDID);

  // Render UI
}
```

### Testing

#### 1. Unit Tests with Jest

```typescript
// ✅ Mock SAGE client
import { SAGEClient } from '@sage-x/sdk';

jest.mock('@sage-x/sdk');

describe('AgentService', () => {
  let mockClient: jest.Mocked<SAGEClient>;

  beforeEach(() => {
    mockClient = new SAGEClient() as jest.Mocked<SAGEClient>;
    mockClient.initiateHandshake.mockResolvedValue({
      id: 'session-123',
      clientPublicKey: new Uint8Array(),
      serverPublicKey: new Uint8Array(),
      createdAt: new Date(),
      expiresAt: new Date(Date.now() + 3600000),
    });
  });

  it('should establish session and send message', async () => {
    const service = new AgentService(mockClient);
    await service.sendSecureMessage('did:test', 'Hello');

    expect(mockClient.initiateHandshake).toHaveBeenCalledTimes(1);
    expect(mockClient.sendMessage).toHaveBeenCalledTimes(1);
  });
});
```

#### 2. React Testing Library

```typescript
// ✅ Test React components
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SAGEProvider } from './SAGEProvider';
import { ChatComponent } from './ChatComponent';

describe('ChatComponent', () => {
  it('should send message when button clicked', async () => {
    render(
      <SAGEProvider>
        <ChatComponent targetDID="did:test:123" />
      </SAGEProvider>
    );

    const input = screen.getByPlaceholderText('Enter message');
    const button = screen.getByText('Send');

    await userEvent.type(input, 'Hello');
    await userEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText('Message sent')).toBeInTheDocument();
    });
  });
});
```

---

## Advanced Usage

### Multi-Agent Coordination

```typescript
class AgentCoordinator {
  private sessionPool = new Map<string, Session>();

  constructor(private client: SAGEClient) {}

  async broadcast(dids: string[], message: Uint8Array): Promise<Uint8Array[]> {
    // Ensure sessions exist
    await Promise.all(dids.map(did => this.ensureSession(did)));

    // Send to all in parallel
    const sends = dids.map(did => {
      const session = this.sessionPool.get(did)!;
      return this.client.sendMessage(session.id, message);
    });

    return await Promise.all(sends);
  }

  private async ensureSession(did: string): Promise<Session> {
    let session = this.sessionPool.get(did);
    if (!session || new Date() > session.expiresAt) {
      session = await this.client.initiateHandshake(did);
      this.sessionPool.set(did, session);
    }
    return session;
  }

  cleanup() {
    const now = new Date();
    for (const [did, session] of this.sessionPool.entries()) {
      if (now > session.expiresAt) {
        this.client.closeSession(session.id);
        this.sessionPool.delete(did);
      }
    }
  }
}

// Usage
const coordinator = new AgentCoordinator(client);
setInterval(() => coordinator.cleanup(), 60000); // Cleanup every minute

const responses = await coordinator.broadcast(
  ['did:sage:agent1', 'did:sage:agent2', 'did:sage:agent3'],
  new TextEncoder().encode('Broadcast message')
);
```

### Custom Event System

```typescript
// ✅ Type-safe event emitter
interface SAGEEvents {
  'session:created': Session;
  'session:expired': { sessionID: string };
  'message:sent': { sessionID: string; size: number };
  'message:received': { sessionID: string; message: Uint8Array };
  'error': Error;
}

class TypedEventEmitter<Events extends Record<string, any>> {
  private handlers = new Map<keyof Events, Set<(data: any) => void>>();

  on<K extends keyof Events>(event: K, handler: (data: Events[K]) => void) {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);
  }

  off<K extends keyof Events>(event: K, handler: (data: Events[K]) => void) {
    this.handlers.get(event)?.delete(handler);
  }

  emit<K extends keyof Events>(event: K, data: Events[K]) {
    this.handlers.get(event)?.forEach(handler => handler(data));
  }
}

// Usage with SAGE
const events = new TypedEventEmitter<SAGEEvents>();

events.on('session:created', (session) => {
  console.log('Session created:', session.id);
});

events.on('message:sent', ({ sessionID, size }) => {
  console.log(`Message sent in ${sessionID}: ${size} bytes`);
});
```

### WebSocket Integration

```typescript
// ✅ Real-time messaging with WebSocket
class SAGEWebSocket {
  private ws: WebSocket;
  private client: SAGEClient;

  constructor(url: string, client: SAGEClient) {
    this.client = client;
    this.ws = new WebSocket(url);
    this.setupHandlers();
  }

  private setupHandlers() {
    this.ws.onmessage = async (event) => {
      const { sessionID, encryptedMessage } = JSON.parse(event.data);

      // Decrypt received message
      const plaintext = await this.client.receiveMessage(
        sessionID,
        encryptedMessage
      );

      this.onMessage?.(sessionID, plaintext);
    };
  }

  async sendMessage(sessionID: string, message: Uint8Array) {
    const encrypted = await this.client.sendMessage(sessionID, message);
    this.ws.send(JSON.stringify({ sessionID, encrypted }));
  }

  onMessage?: (sessionID: string, message: Uint8Array) => void;
}
```

### Monitoring and Metrics

```typescript
// ✅ Performance monitoring
class SAGEMetrics {
  private metrics = {
    handshakes: 0,
    messagesSent: 0,
    messagesReceived: 0,
    errors: 0,
    avgHandshakeTime: 0,
    avgMessageTime: 0,
  };

  private handshakeTimes: number[] = [];
  private messageTimes: number[] = [];

  recordHandshake(duration: number) {
    this.metrics.handshakes++;
    this.handshakeTimes.push(duration);
    this.metrics.avgHandshakeTime =
      this.handshakeTimes.reduce((a, b) => a + b, 0) / this.handshakeTimes.length;
  }

  recordMessage(duration: number) {
    this.metrics.messagesSent++;
    this.messageTimes.push(duration);
    this.metrics.avgMessageTime =
      this.messageTimes.reduce((a, b) => a + b, 0) / this.messageTimes.length;
  }

  recordError() {
    this.metrics.errors++;
  }

  getMetrics() {
    return {
      ...this.metrics,
      successRate: (this.metrics.messagesSent - this.metrics.errors) /
                   this.metrics.messagesSent * 100
    };
  }
}

// Instrumented client wrapper
class MonitoredSAGEClient {
  private metrics = new SAGEMetrics();

  constructor(private client: SAGEClient) {}

  async initiateHandshake(publicKey: Uint8Array) {
    const start = performance.now();
    try {
      const session = await this.client.initiateHandshake(publicKey);
      this.metrics.recordHandshake(performance.now() - start);
      return session;
    } catch (error) {
      this.metrics.recordError();
      throw error;
    }
  }

  async sendMessage(sessionID: string, message: Uint8Array) {
    const start = performance.now();
    try {
      const result = await this.client.sendMessage(sessionID, message);
      this.metrics.recordMessage(performance.now() - start);
      return result;
    } catch (error) {
      this.metrics.recordError();
      throw error;
    }
  }

  getMetrics() {
    return this.metrics.getMetrics();
  }
}

// Export metrics endpoint
app.get('/metrics', (req, res) => {
  const metrics = monitoredClient.getMetrics();
  res.json(metrics);
});
```

---

## API Documentation

Full API documentation generated with TypeDoc:

```bash
# Generate API docs
npm run docs

# View documentation
open docs/index.html
```

View online: [API Reference](https://sage-x-project.github.io/sage/sdk/typescript/)

---

## Support

For questions and support:
- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://github.com/sage-x-project/sage/tree/main/docs
- Discord: [SAGE Community](https://discord.gg/sage-community)
