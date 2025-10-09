"""
SAGE Client - Main API
"""

import time
import json
from typing import Optional, Dict, Any
import httpx

from sage_client.crypto import Crypto, setup_hpke_sender, setup_hpke_receiver
from sage_client.did import DID, DIDDocument, DIDResolver
from sage_client.session import Session, SessionManager
from sage_client.types import (
    KeyPair,
    HandshakeRequest,
    HandshakeResponse,
    MessageRequest,
    MessageResponse,
    AgentMetadata,
    HealthStatus,
)
from sage_client.exceptions import (
    SAGEError,
    NetworkError,
    SessionError,
    ValidationError,
)


class SAGEClient:
    """
    SAGE Client for secure agent-to-agent communication

    Example:
        ```python
        client = SAGEClient("http://localhost:8080")
        await client.initialize()

        # Register agent
        await client.register_agent("did:sage:ethereum:0xAlice", "Alice")

        # Initiate handshake
        session_id = await client.handshake("did:sage:ethereum:0xServer")

        # Send message
        response = await client.send_message(session_id, b"Hello, Server!")
        ```
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        identity_keypair: Optional[KeyPair] = None,
        kem_keypair: Optional[KeyPair] = None,
        timeout: float = 30.0,
    ):
        """
        Initialize SAGE client

        Args:
            base_url: SAGE server base URL
            identity_keypair: Ed25519 keypair for signing (generates if None)
            kem_keypair: X25519 keypair for encryption (generates if None)
            timeout: HTTP request timeout in seconds
        """
        self.base_url = base_url.rstrip("/")
        self.identity_keypair = identity_keypair
        self.kem_keypair = kem_keypair
        self.timeout = timeout

        self.did_resolver = DIDResolver()
        self.session_manager = SessionManager()
        self.client_did: Optional[str] = None

        # HTTP client
        self._http_client: Optional[httpx.AsyncClient] = None

    async def initialize(
        self,
        identity_keypair: Optional[KeyPair] = None,
        kem_keypair: Optional[KeyPair] = None,
    ) -> None:
        """
        Initialize client with keypairs

        Args:
            identity_keypair: Ed25519 keypair (generates if None)
            kem_keypair: X25519 keypair (generates if None)
        """
        if identity_keypair:
            self.identity_keypair = identity_keypair
        elif not self.identity_keypair:
            self.identity_keypair = Crypto.generate_ed25519_keypair()

        if kem_keypair:
            self.kem_keypair = kem_keypair
        elif not self.kem_keypair:
            self.kem_keypair = Crypto.generate_x25519_keypair()

        # Initialize HTTP client
        self._http_client = httpx.AsyncClient(timeout=self.timeout)

    async def close(self) -> None:
        """Close HTTP client"""
        if self._http_client:
            await self._http_client.aclose()
            self._http_client = None

    async def __aenter__(self) -> "SAGEClient":
        await self.initialize()
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        await self.close()

    def _ensure_initialized(self) -> None:
        """Ensure client is initialized"""
        if not self.identity_keypair or not self.kem_keypair:
            raise SAGEError("Client not initialized. Call initialize() first.")
        if not self._http_client:
            raise SAGEError("HTTP client not initialized. Use async context manager.")

    async def get_server_kem_key(self) -> bytes:
        """
        Get server's KEM public key

        Returns:
            Server's X25519 public key

        Raises:
            NetworkError: If request fails
        """
        self._ensure_initialized()

        try:
            response = await self._http_client.get(f"{self.base_url}/debug/kem-pub")
            response.raise_for_status()
            data = response.json()
            return Crypto.base64_decode(data["kem_public_key"])
        except httpx.HTTPError as e:
            raise NetworkError(f"Failed to get server KEM key: {e}")

    async def get_server_did(self) -> str:
        """
        Get server's DID

        Returns:
            Server DID string

        Raises:
            NetworkError: If request fails
        """
        self._ensure_initialized()

        try:
            response = await self._http_client.get(f"{self.base_url}/debug/server-did")
            response.raise_for_status()
            data = response.json()
            return data["did"]
        except httpx.HTTPError as e:
            raise NetworkError(f"Failed to get server DID: {e}")

    async def health_check(self) -> HealthStatus:
        """
        Check server health

        Returns:
            Health status

        Raises:
            NetworkError: If request fails
        """
        self._ensure_initialized()

        try:
            response = await self._http_client.get(f"{self.base_url}/debug/health")
            response.raise_for_status()
            data = response.json()
            return HealthStatus(**data)
        except httpx.HTTPError as e:
            raise NetworkError(f"Health check failed: {e}")

    async def register_agent(
        self, did: str, name: str, is_active: bool = True
    ) -> None:
        """
        Register agent (development only)

        Args:
            did: Agent DID
            name: Agent name
            is_active: Whether agent is active

        Raises:
            NetworkError: If registration fails
        """
        self._ensure_initialized()

        if not self.identity_keypair or not self.kem_keypair:
            raise SAGEError("Keypairs not initialized")

        agent_data = AgentMetadata(
            did=did,
            name=name,
            is_active=is_active,
            public_key=Crypto.base64_encode(self.identity_keypair.public_key),
            public_kem_key=Crypto.base64_encode(self.kem_keypair.public_key),
        )

        try:
            response = await self._http_client.post(
                f"{self.base_url}/debug/register-agent",
                json=agent_data.model_dump(),
            )
            response.raise_for_status()

            # Cache DID document locally
            did_doc = DIDDocument(
                did=did,
                public_key=self.identity_keypair.public_key,
                public_kem_key=self.kem_keypair.public_key,
                owner_address=did.split(":")[-1],  # Extract address
                is_active=is_active,
            )
            self.did_resolver.register(did_doc)
            self.client_did = did

        except httpx.HTTPError as e:
            raise NetworkError(f"Failed to register agent: {e}")

    async def handshake(self, server_did: str) -> str:
        """
        Initiate HPKE handshake with server

        Args:
            server_did: Server DID

        Returns:
            Session ID

        Raises:
            NetworkError: If handshake fails
            SessionError: If session cannot be established
        """
        self._ensure_initialized()

        if not self.client_did:
            raise SAGEError("Client DID not set. Call register_agent() first.")

        # Get server's KEM public key
        server_kem_key = await self.get_server_kem_key()

        # Setup HPKE as sender
        hpke_ctx, encapsulated_key = setup_hpke_sender(server_kem_key)

        # Create handshake payload
        handshake_data = {
            "type": "handshake",
            "client_did": self.client_did,
            "timestamp": int(time.time()),
        }
        plaintext = json.dumps(handshake_data).encode("utf-8")

        # Encrypt with HPKE
        ciphertext = hpke_ctx.seal(plaintext)

        # Combine encapsulated key + ciphertext
        message = encapsulated_key + ciphertext
        message_b64 = Crypto.base64_encode(message)

        # Sign the message
        timestamp = int(time.time())
        to_sign = f"{self.client_did}|{server_did}|{message_b64}|{timestamp}".encode(
            "utf-8"
        )
        signature = Crypto.sign(to_sign, self.identity_keypair.private_key)
        signature_b64 = Crypto.base64_encode(signature)

        # Send handshake request
        request = HandshakeRequest(
            sender_did=self.client_did,
            receiver_did=server_did,
            message=message_b64,
            timestamp=timestamp,
            signature=signature_b64,
        )

        try:
            response = await self._http_client.post(
                f"{self.base_url}/v1/a2a:sendMessage",
                json=request.model_dump(),
            )
            response.raise_for_status()
            data = response.json()

            handshake_resp = HandshakeResponse(**data)

            # Decrypt response
            response_bytes = Crypto.base64_decode(handshake_resp.response)
            decrypted = hpke_ctx.open(response_bytes)

            # Parse response
            response_data = json.loads(decrypted)
            session_id = handshake_resp.session_id

            # Create session
            session = Session(
                session_id=session_id,
                client_did=self.client_did,
                server_did=server_did,
                hpke_context=hpke_ctx,
            )
            self.session_manager.add_session(session)

            return session_id

        except httpx.HTTPError as e:
            raise NetworkError(f"Handshake failed: {e}")
        except Exception as e:
            raise SessionError(f"Failed to establish session: {e}")

    async def send_message(
        self, session_id: str, message: bytes, receiver_did: Optional[str] = None
    ) -> bytes:
        """
        Send encrypted message in existing session

        Args:
            session_id: Session ID
            message: Message to send
            receiver_did: Receiver DID (optional, uses session's server DID)

        Returns:
            Decrypted response

        Raises:
            SessionError: If session not found
            NetworkError: If request fails
        """
        self._ensure_initialized()

        session = self.session_manager.get_session(session_id)
        if not session:
            raise SessionError(f"Session not found: {session_id}")

        if not receiver_did:
            receiver_did = session.server_did

        # Encrypt message
        ciphertext = session.encrypt(message)
        message_b64 = Crypto.base64_encode(ciphertext)

        # Sign
        timestamp = int(time.time())
        to_sign = f"{self.client_did}|{receiver_did}|{message_b64}|{timestamp}".encode(
            "utf-8"
        )
        signature = Crypto.sign(to_sign, self.identity_keypair.private_key)
        signature_b64 = Crypto.base64_encode(signature)

        # Send message
        request = MessageRequest(
            sender_did=self.client_did,
            receiver_did=receiver_did,
            message=message_b64,
            timestamp=timestamp,
            signature=signature_b64,
        )

        try:
            response = await self._http_client.post(
                f"{self.base_url}/v1/a2a:sendMessage",
                json=request.model_dump(),
                headers={"X-Session-ID": session_id},
            )
            response.raise_for_status()
            data = response.json()

            msg_resp = MessageResponse(**data)

            # Decrypt response
            response_bytes = Crypto.base64_decode(msg_resp.response)
            decrypted = session.decrypt(response_bytes)

            return decrypted

        except httpx.HTTPError as e:
            raise NetworkError(f"Failed to send message: {e}")

    def get_active_sessions(self) -> list[Session]:
        """Get list of active sessions"""
        return self.session_manager.get_active_sessions()

    def close_session(self, session_id: str) -> None:
        """Close session"""
        self.session_manager.remove_session(session_id)
