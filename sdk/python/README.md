# SAGE Python Client

Python client library for SAGE (Secure Agent Guarantee Engine) - providing secure, decentralized identity and communication for AI agents.

## Features

- ✅ **Ed25519 Signatures**: Cryptographic signing and verification
- ✅ **HPKE Encryption**: Hybrid Public Key Encryption for secure sessions
- ✅ **DID Support**: Decentralized identifiers for agent identity
- ✅ **Session Management**: Efficient stateful communication
- ✅ **Async/Await**: Full async support with `httpx`
- ✅ **Type Hints**: Complete type annotations for better IDE support

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

## Roadmap

- [ ] Full HPKE library integration (pyhpke)
- [ ] Blockchain DID resolution
- [ ] WebSocket support
- [ ] Batch message operations
- [ ] Advanced error recovery
- [ ] Performance optimizations
- [ ] Complete test coverage

---

**Made with ❤️ by the SAGE Team**
