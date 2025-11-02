# SAGE Python Client

Python client library for SAGE (Secure Agent Guarantee Engine) - providing secure, decentralized identity and communication for AI agents.

## Features

-  **Ed25519 Signatures**: Cryptographic signing and verification
-  **HPKE Encryption**: Hybrid Public Key Encryption for secure sessions
-  **DID Support**: Decentralized identifiers for agent identity
-  **Session Management**: Efficient stateful communication
-  **Async/Await**: Full async support with `httpx`
-  **Type Hints**: Complete type annotations for better IDE support

## Installation

```bash
# From source
cd sdk/python
pip install -e .

# With development dependencies
pip install -e ".[dev]"

# From PyPI (coming soon)
pip install sage-client
```

## Quick Start

```python
import asyncio
from sage_client import SAGEClient

async def main():
    # Initialize client
    async with SAGEClient("http://localhost:8080") as client:
        # Register agent
        await client.register_agent("did:sage:ethereum:0xAlice", "Alice")

        # Initiate handshake
        session_id = await client.handshake("did:sage:ethereum:0xServer")

        # Send message
        response = await client.send_message(session_id, b"Hello!")
        print(response)

asyncio.run(main())
```

## Requirements

- Python 3.8+
- `cryptography>=41.0.0`
- `httpx>=0.24.0`
- `pydantic>=2.0.0`

## Documentation

### Client Initialization

```python
from sage_client import SAGEClient

# Basic initialization
client = SAGEClient("http://localhost:8080")
await client.initialize()

# With custom keypairs
from sage_client import Crypto

identity_keypair = Crypto.generate_ed25519_keypair()
kem_keypair = Crypto.generate_x25519_keypair()

client = SAGEClient(
    "http://localhost:8080",
    identity_keypair=identity_keypair,
    kem_keypair=kem_keypair
)
await client.initialize()

# Using context manager (recommended)
async with SAGEClient("http://localhost:8080") as client:
    # Client automatically initialized and closed
    pass
```

### Agent Registration

```python
# Register agent (development only)
await client.register_agent(
    did="did:sage:ethereum:0xAlice",
    name="Alice Agent",
    is_active=True
)
```

### Secure Communication

```python
# Get server DID
server_did = await client.get_server_did()

# Initiate HPKE handshake
session_id = await client.handshake(server_did)

# Send encrypted message
message = b"Hello, Server!"
response = await client.send_message(session_id, message)

# Send multiple messages in same session
for i in range(5):
    msg = f"Message {i}".encode()
    resp = await client.send_message(session_id, msg)
    print(resp)
```

### Session Management

```python
# Get active sessions
sessions = client.get_active_sessions()
for session in sessions:
    print(f"Session: {session.session_id}")
    print(f"  Client: {session.client_did}")
    print(f"  Server: {session.server_did}")
    print(f"  Messages: {session.message_count}")
    print(f"  Expired: {session.is_expired()}")

# Close specific session
client.close_session(session_id)

# Cleanup expired sessions
client.session_manager.cleanup_expired()
```

### Health Check

```python
health = await client.health_check()
print(f"Status: {health.status}")
print(f"Active sessions: {health.sessions}")
```

### Cryptography

```python
from sage_client import Crypto

# Generate keypairs
ed25519_keypair = Crypto.generate_ed25519_keypair()
x25519_keypair = Crypto.generate_x25519_keypair()

# Sign and verify
message = b"Important message"
signature = Crypto.sign(message, ed25519_keypair.private_key)
is_valid = Crypto.verify(message, signature, ed25519_keypair.public_key)

# HPKE encryption
from sage_client.crypto import setup_hpke_sender, setup_hpke_receiver

sender_ctx, encapsulated_key = setup_hpke_sender(receiver_public_key)
ciphertext = sender_ctx.seal(b"Secret")

receiver_ctx = setup_hpke_receiver(encapsulated_key, receiver_private_key)
plaintext = receiver_ctx.open(ciphertext)

# Base64 encoding
encoded = Crypto.base64_encode(b"data")
decoded = Crypto.base64_decode(encoded)
```

### DID Management

```python
from sage_client import DID, DIDDocument, DIDResolver

# Parse DID
did = DID("did:sage:ethereum:0x742d35Cc...")
print(did.network)  # "ethereum"
print(did.address)  # "0x742d35Cc..."

# Create DID from components
did = DID.from_address("ethereum", "0x742d35Cc...")

# DID resolver
resolver = DIDResolver()

# Register DID document (development)
did_doc = DIDDocument(
    did="did:sage:ethereum:0xAlice",
    public_key=keypair.public_key,
    public_kem_key=kem_keypair.public_key,
    owner_address="0xAlice",
)
resolver.register(did_doc)

# Resolve DID
doc = await resolver.resolve("did:sage:ethereum:0xAlice")
```

## Examples

See [`examples/`](examples/) directory:

- `basic_usage.py`: Simple client usage
- More examples coming soon...

## Testing

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# With coverage
pytest --cov=sage_client --cov-report=html

# Run specific test
pytest tests/test_crypto.py -v
```

## Development

```bash
# Install in editable mode with dev dependencies
pip install -e ".[dev]"

# Format code
black sage_client tests examples

# Lint
ruff check sage_client tests examples

# Type check
mypy sage_client
```

## Architecture

```
sage_client/
├── __init__.py       # Main exports
├── client.py         # SAGEClient (main API)
├── crypto.py         # Cryptography (Ed25519, X25519, HPKE)
├── did.py            # DID parsing and resolution
├── session.py        # Session management
├── types.py          # Pydantic models
└── exceptions.py     # Custom exceptions
```

## Security

- All communication is encrypted with HPKE
- Messages are signed with Ed25519
- Replay attack prevention via timestamps and nonces
- Session expiration (default: 1 hour)

## Error Handling

```python
from sage_client.exceptions import (
    SAGEError,
    CryptoError,
    SessionError,
    NetworkError,
    ValidationError,
)

try:
    session_id = await client.handshake(server_did)
except NetworkError as e:
    print(f"Network error: {e}")
except SessionError as e:
    print(f"Session error: {e}")
except SAGEError as e:
    print(f"SAGE error: {e}")
```

## License

LGPL-3.0 - See [LICENSE](../../LICENSE) for details

## Contributing

Contributions welcome! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md)

## Support

- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://github.com/sage-x-project/sage/tree/main/docs
- Label: `python-sdk`

## Changelog

### 0.1.0 (2025-10-10)

- Initial release
- Ed25519 signing and verification
- HPKE encryption support
- DID parsing and resolution
- Session management
- Async HTTP client
- Type hints and Pydantic models
- Basic test coverage

## Troubleshooting

### Common Issues

#### Connection Errors

**Problem:** `NetworkError: Connection refused` or timeout errors

**Solutions:**
```python
# 1. Check server is running
import httpx

async with httpx.AsyncClient() as http:
    try:
        response = await http.get("http://localhost:8080/health")
        print(f"Server status: {response.json()}")
    except httpx.ConnectError:
        print("Server not responding - ensure SAGE server is running")

# 2. Increase timeout for slow networks
client = SAGEClient(
    "http://localhost:8080",
    timeout=60.0  # 60 seconds
)

# 3. Check firewall settings
# Ensure port 8080 is open and accessible
```

#### Session Expired Errors

**Problem:** `SessionError: Session expired or not found`

**Solutions:**
```python
# 1. Check session expiration before sending messages
session = client.session_manager.get_session(session_id)
if session.is_expired():
    # Re-establish handshake
    session_id = await client.handshake(server_did)

# 2. Use longer session TTL (server-side configuration)
# Or implement automatic session renewal

# 3. Handle expiration gracefully
from sage_client.exceptions import SessionError

try:
    response = await client.send_message(session_id, message)
except SessionError:
    # Automatically re-establish session
    session_id = await client.handshake(server_did)
    response = await client.send_message(session_id, message)
```

#### Signature Verification Failures

**Problem:** `ValidationError: Invalid signature`

**Solutions:**
```python
# 1. Ensure keypairs match between client and server
# Verify public key is correctly registered
did_doc = await resolver.resolve(client_did)
print(f"Registered public key: {did_doc.public_key.hex()}")
print(f"Client public key: {client.identity_keypair.public_key.hex()}")

# 2. Check timestamp synchronization
import time
print(f"Local time: {time.time()}")
# Ensure system clocks are synchronized (NTP)

# 3. Verify message hasn't been tampered with
# Check network proxy/middleware isn't modifying requests
```

#### DID Resolution Errors

**Problem:** `ValidationError: DID not found` or blockchain connection errors

**Solutions:**
```python
# 1. Check DID format
from sage_client import DID

try:
    did = DID("did:sage:ethereum:0xAddress")
    print(f"Network: {did.network}")  # Should be "ethereum"
except ValueError as e:
    print(f"Invalid DID format: {e}")

# 2. Verify blockchain connection (if using on-chain resolution)
# For development, use mock resolver
from sage_client import DIDResolver, DIDDocument

resolver = DIDResolver()
did_doc = DIDDocument(
    did="did:sage:ethereum:0xTest",
    public_key=keypair.public_key,
    public_kem_key=kem_keypair.public_key,
    owner_address="0xTest"
)
resolver.register(did_doc)  # Mock registration
```

#### Memory Issues with Large Messages

**Problem:** High memory usage or timeouts with large payloads

**Solutions:**
```python
# 1. Chunk large messages
def chunk_message(data: bytes, chunk_size: int = 1024 * 1024):
    """Split data into chunks (default 1MB)"""
    for i in range(0, len(data), chunk_size):
        yield data[i:i + chunk_size]

# Send large file in chunks
large_data = open("large_file.bin", "rb").read()
for i, chunk in enumerate(chunk_message(large_data)):
    metadata = {"chunk": i, "total": len(list(chunk_message(large_data)))}
    await client.send_message(session_id, chunk)

# 2. Use streaming for very large transfers
# Consider external storage (S3, IPFS) and send reference via SAGE
```

### Debug Mode

Enable verbose logging for troubleshooting:

```python
import logging

# Enable debug logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

# SAGE client will output detailed logs
client = SAGEClient("http://localhost:8080")
# Now all crypto, network, and session operations are logged
```

### Performance Issues

**Problem:** Slow handshake or message sending

**Diagnostics:**
```python
import time

# Measure handshake time
start = time.time()
session_id = await client.handshake(server_did)
print(f"Handshake took: {time.time() - start:.2f}s")

# Measure message sending time
start = time.time()
response = await client.send_message(session_id, b"test")
print(f"Message took: {time.time() - start:.2f}s")

# Expected timings:
# - Handshake: 50-200ms (depends on network)
# - Message: 20-100ms (small message)
```

**Solutions:**
- Reuse sessions instead of creating new ones
- Use connection pooling (httpx handles this automatically)
- Check network latency with `ping`
- Profile crypto operations if needed

---

## Best Practices

### Security

1. **Never log or expose private keys**
   ```python
   #  BAD
   print(f"Private key: {keypair.private_key.hex()}")

   #  GOOD
   print(f"Public key: {keypair.public_key.hex()}")
   ```

2. **Validate all inputs**
   ```python
   from sage_client import DID

   def process_did(did_string: str):
       try:
           did = DID(did_string)
           # Proceed with valid DID
       except ValueError:
           # Handle invalid DID
           raise ValidationError("Invalid DID format")
   ```

3. **Use secure key storage**
   ```python
   # For production, use OS keyring or HSM
   import keyring

   # Store private key securely
   keyring.set_password("sage", "private_key", keypair.private_key.hex())

   # Retrieve when needed
   private_key_hex = keyring.get_password("sage", "private_key")
   ```

4. **Implement retry logic with exponential backoff**
   ```python
   import asyncio

   async def send_with_retry(client, session_id, message, max_retries=3):
       for attempt in range(max_retries):
           try:
               return await client.send_message(session_id, message)
           except NetworkError:
               if attempt == max_retries - 1:
                   raise
               await asyncio.sleep(2 ** attempt)  # Exponential backoff
   ```

### Performance

1. **Reuse sessions**
   ```python
   #  BAD - Creates new session for each message
   for msg in messages:
       session_id = await client.handshake(server_did)
       await client.send_message(session_id, msg)

   #  GOOD - Reuse session
   session_id = await client.handshake(server_did)
   for msg in messages:
       await client.send_message(session_id, msg)
   ```

2. **Close client properly**
   ```python
   #  Use context manager (auto-close)
   async with SAGEClient("http://localhost:8080") as client:
       # Client automatically closed
       pass

   # Or close manually
   client = SAGEClient("http://localhost:8080")
   try:
       await client.initialize()
       # Use client
   finally:
       await client.close()
   ```

3. **Batch operations when possible**
   ```python
   # Instead of multiple handshakes, reuse sessions
   sessions = {}
   for did in target_dids:
       if did not in sessions:
           sessions[did] = await client.handshake(did)
       await client.send_message(sessions[did], message)
   ```

4. **Monitor session cleanup**
   ```python
   # Periodically cleanup expired sessions
   import asyncio

   async def cleanup_task(client):
       while True:
           client.session_manager.cleanup_expired()
           await asyncio.sleep(300)  # Every 5 minutes

   # Run as background task
   asyncio.create_task(cleanup_task(client))
   ```

### Code Organization

1. **Separate configuration from code**
   ```python
   # config.py
   from pydantic import BaseSettings

   class Settings(BaseSettings):
       sage_base_url: str = "http://localhost:8080"
       sage_timeout: float = 30.0
       sage_did: str = ""

       class Config:
           env_file = ".env"

   # main.py
   from config import Settings

   settings = Settings()
   client = SAGEClient(settings.sage_base_url, timeout=settings.sage_timeout)
   ```

2. **Use dependency injection**
   ```python
   class AgentService:
       def __init__(self, client: SAGEClient):
           self.client = client

       async def send_secure_message(self, target_did: str, message: bytes):
           session_id = await self.client.handshake(target_did)
           return await self.client.send_message(session_id, message)

   # Easy to test with mock client
   async def test_send_message():
       mock_client = MockSAGEClient()
       service = AgentService(mock_client)
       await service.send_secure_message("did:sage:test", b"test")
   ```

3. **Handle errors consistently**
   ```python
   from sage_client.exceptions import SAGEError, NetworkError, SessionError

   class MessageHandler:
       async def send(self, session_id, message):
           try:
               return await self.client.send_message(session_id, message)
           except NetworkError as e:
               logger.error(f"Network error: {e}")
               # Retry logic
           except SessionError as e:
               logger.warning(f"Session error: {e}")
               # Re-establish session
           except SAGEError as e:
               logger.critical(f"SAGE error: {e}")
               # Alert monitoring
   ```

### Testing

1. **Use mock client for unit tests**
   ```python
   # tests/test_service.py
   import pytest
   from unittest.mock import AsyncMock

   @pytest.mark.asyncio
   async def test_agent_communication():
       mock_client = AsyncMock(spec=SAGEClient)
       mock_client.handshake.return_value = "session-123"
       mock_client.send_message.return_value = b"response"

       service = AgentService(mock_client)
       result = await service.send_secure_message("did:test", b"hello")

       assert result == b"response"
       mock_client.handshake.assert_called_once()
   ```

2. **Integration tests with test server**
   ```python
   # tests/integration/test_client.py
   import pytest
   import os

   @pytest.mark.integration
   @pytest.mark.asyncio
   async def test_full_handshake():
       # Skip if test server not available
       if not os.environ.get("SAGE_TEST_SERVER"):
           pytest.skip("Test server not available")

       async with SAGEClient(os.environ["SAGE_TEST_SERVER"]) as client:
           # Test with real server
           session_id = await client.handshake("did:sage:test:server")
           assert session_id is not None
   ```

---

## Advanced Usage

### Multi-Agent Coordination

```python
class AgentCoordinator:
    def __init__(self, base_url: str):
        self.client = None
        self.base_url = base_url
        self.sessions = {}

    async def __aenter__(self):
        self.client = SAGEClient(self.base_url)
        await self.client.initialize()
        return self

    async def __aexit__(self, *args):
        await self.client.close()

    async def broadcast(self, target_dids: list[str], message: bytes):
        """Send message to multiple agents"""
        # Establish sessions in parallel
        import asyncio

        async def ensure_session(did):
            if did not in self.sessions:
                self.sessions[did] = await self.client.handshake(did)
            return self.sessions[did]

        session_ids = await asyncio.gather(*[ensure_session(did) for did in target_dids])

        # Send messages in parallel
        responses = await asyncio.gather(
            *[self.client.send_message(sid, message) for sid in session_ids]
        )
        return responses

# Usage
async def main():
    agents = [
        "did:sage:ethereum:0xAgent1",
        "did:sage:ethereum:0xAgent2",
        "did:sage:ethereum:0xAgent3"
    ]

    async with AgentCoordinator("http://localhost:8080") as coordinator:
        responses = await coordinator.broadcast(agents, b"Broadcast message")
        for i, resp in enumerate(responses):
            print(f"Agent {i} response: {resp}")
```

### Connection Pooling

```python
# SAGE client uses httpx which has built-in connection pooling
# Configure limits for high-throughput scenarios
import httpx

# Custom httpx client with connection limits
http_client = httpx.AsyncClient(
    limits=httpx.Limits(
        max_keepalive_connections=100,
        max_connections=200,
        keepalive_expiry=30.0
    )
)

# Use with SAGE client (if supported in future version)
# For now, httpx defaults are sufficient for most use cases
```

### Custom Crypto Provider

```python
# For HSM or custom crypto implementations
from sage_client.crypto import Crypto
from cryptography.hazmat.primitives.asymmetric import ed25519

class HSMCrypto(Crypto):
    """Custom crypto provider using HSM"""

    @staticmethod
    def sign(message: bytes, private_key: bytes) -> bytes:
        # Use HSM to sign
        # This is a placeholder - actual HSM integration required
        # Example: PKCS#11, AWS CloudHSM, Azure Key Vault
        return hsm_sign(message, key_id="sage-key")

    @staticmethod
    def verify(message: bytes, signature: bytes, public_key: bytes) -> bool:
        # Verify using public key
        key = ed25519.Ed25519PublicKey.from_public_bytes(public_key)
        try:
            key.verify(signature, message)
            return True
        except Exception:
            return False

# Use custom crypto (future feature)
# client = SAGEClient("http://localhost:8080", crypto_provider=HSMCrypto)
```

### Monitoring and Metrics

```python
import time
import prometheus_client as prom

# Define metrics
handshake_duration = prom.Histogram(
    'sage_handshake_duration_seconds',
    'Time spent in handshake'
)
message_counter = prom.Counter(
    'sage_messages_total',
    'Total messages sent'
)
session_gauge = prom.Gauge(
    'sage_active_sessions',
    'Number of active sessions'
)

# Instrument client
class MonitoredSAGEClient:
    def __init__(self, base_url: str):
        self.client = SAGEClient(base_url)

    async def initialize(self):
        await self.client.initialize()

    @handshake_duration.time()
    async def handshake(self, server_did: str) -> str:
        session_id = await self.client.handshake(server_did)
        session_gauge.inc()
        return session_id

    async def send_message(self, session_id: str, message: bytes) -> bytes:
        message_counter.inc()
        return await self.client.send_message(session_id, message)

# Start metrics server
prom.start_http_server(9090)

# Use monitored client
async with MonitoredSAGEClient("http://localhost:8080") as client:
    # Metrics available at http://localhost:9090
    pass
```

---

## API Reference

Full API documentation is available in the docstrings. Generate HTML documentation:

```bash
# Install pdoc
pip install pdoc

# Generate documentation
pdoc sage_client --html --output-dir docs/

# View in browser
open docs/sage_client/index.html
```

### Core Classes

- **SAGEClient**: Main client interface
  - `initialize()`: Initialize client and generate keys
  - `register_agent()`: Register agent with server
  - `handshake()`: Initiate HPKE handshake
  - `send_message()`: Send encrypted message
  - `health_check()`: Check server health
  - `close()`: Close client and cleanup sessions

- **Crypto**: Cryptographic operations
  - `generate_ed25519_keypair()`: Generate Ed25519 signing keypair
  - `generate_x25519_keypair()`: Generate X25519 KEM keypair
  - `sign()`: Sign message with Ed25519
  - `verify()`: Verify Ed25519 signature
  - `base64_encode()`, `base64_decode()`: Base64 encoding

- **DID**: Decentralized identifier parsing
  - `from_address()`: Create DID from network and address
  - `network`: Get network from DID
  - `address`: Get address from DID

- **DIDResolver**: DID document resolution
  - `register()`: Register DID document (dev mode)
  - `resolve()`: Resolve DID to document

- **SessionManager**: Session lifecycle management
  - `create_session()`: Create new session
  - `get_session()`: Get existing session
  - `close_session()`: Close session
  - `cleanup_expired()`: Remove expired sessions

---

## Roadmap

- [ ] Full HPKE library integration (pyhpke)
- [ ] Blockchain DID resolution
- [ ] WebSocket support
- [ ] Batch message operations
- [ ] Advanced error recovery
- [ ] Performance optimizations
- [ ] Complete test coverage
- [ ] API reference documentation
- [ ] Metrics and monitoring integration
- [ ] HSM/hardware security module support

---

**Made with  by the SAGE Team**
