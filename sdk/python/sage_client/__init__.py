"""
SAGE Python Client

A Python client library for the Secure Agent Guarantee Engine (SAGE).
Provides secure, decentralized identity and communication for AI agents.
"""

from sage_client.client import SAGEClient
from sage_client.crypto import Crypto
from sage_client.did import DID, DIDDocument
from sage_client.session import Session, SessionManager
from sage_client.types import (
    KeyPair,
    Message,
    HandshakeRequest,
    HandshakeResponse,
    MessageRequest,
    MessageResponse,
)
from sage_client.exceptions import (
    SAGEError,
    CryptoError,
    SessionError,
    NetworkError,
    ValidationError,
)

__version__ = "0.1.0"
__all__ = [
    "SAGEClient",
    "Crypto",
    "DID",
    "DIDDocument",
    "Session",
    "SessionManager",
    "KeyPair",
    "Message",
    "HandshakeRequest",
    "HandshakeResponse",
    "MessageRequest",
    "MessageResponse",
    "SAGEError",
    "CryptoError",
    "SessionError",
    "NetworkError",
    "ValidationError",
]
