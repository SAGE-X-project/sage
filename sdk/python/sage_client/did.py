"""
SAGE DID (Decentralized Identifier) Module
"""

import re
from typing import Optional, Dict, Any
from pydantic import BaseModel

from sage_client.exceptions import DIDError, ValidationError


class DID:
    """
    DID (Decentralized Identifier) representation

    Format: did:sage:<network>:<address>
    Example: did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb
    """

    DID_PATTERN = re.compile(r"^did:sage:([a-z]+):(.+)$")

    def __init__(self, did_string: str):
        """
        Initialize DID from string

        Args:
            did_string: DID string

        Raises:
            ValidationError: If DID format is invalid
        """
        self.did_string = did_string
        self._parse()

    def _parse(self) -> None:
        """Parse DID string into components"""
        match = self.DID_PATTERN.match(self.did_string)
        if not match:
            raise ValidationError(f"Invalid DID format: {self.did_string}")

        self.network = match.group(1)
        self.address = match.group(2)

    @classmethod
    def from_address(cls, network: str, address: str) -> "DID":
        """
        Create DID from network and address

        Args:
            network: Network name (e.g., "ethereum", "kaia")
            address: Address on the network

        Returns:
            DID instance
        """
        did_string = f"did:sage:{network}:{address}"
        return cls(did_string)

    def __str__(self) -> str:
        return self.did_string

    def __repr__(self) -> str:
        return f"DID('{self.did_string}')"

    def __eq__(self, other: object) -> bool:
        if not isinstance(other, DID):
            return False
        return self.did_string == other.did_string

    def __hash__(self) -> int:
        return hash(self.did_string)


class DIDDocument(BaseModel):
    """
    DID Document containing identity information

    Represents the resolved DID information including public keys
    """

    did: str
    public_key: bytes  # Ed25519 public key for signing
    public_kem_key: bytes  # X25519 public key for encryption
    owner_address: str
    key_type: str = "Ed25519"
    is_active: bool = True
    revoked: bool = False
    metadata: Dict[str, Any] = {}

    class Config:
        arbitrary_types_allowed = True


class DIDResolver:
    """
    DID Resolver interface

    Resolves DIDs to DID Documents. In production, this would query
    blockchain smart contracts. For development, uses local cache.
    """

    def __init__(self) -> None:
        """Initialize DID resolver"""
        self._cache: Dict[str, DIDDocument] = {}

    async def resolve(self, did: str) -> Optional[DIDDocument]:
        """
        Resolve DID to DID Document

        Args:
            did: DID string to resolve

        Returns:
            DID Document if found, None otherwise

        Raises:
            DIDError: If resolution fails
        """
        # Validate DID format
        try:
            did_obj = DID(did)
        except ValidationError as e:
            raise DIDError(f"Invalid DID: {e}")

        # Check cache
        if did in self._cache:
            doc = self._cache[did]
            if not doc.revoked and doc.is_active:
                return doc
            return None

        # In production, this would query blockchain
        # For now, return None (must be registered first)
        return None

    def register(self, did_doc: DIDDocument) -> None:
        """
        Register DID Document (for development/testing)

        Args:
            did_doc: DID Document to register
        """
        self._cache[did_doc.did] = did_doc

    def revoke(self, did: str) -> None:
        """
        Revoke DID (mark as inactive)

        Args:
            did: DID to revoke
        """
        if did in self._cache:
            self._cache[did].revoked = True

    def clear_cache(self) -> None:
        """Clear DID cache"""
        self._cache.clear()

    def get_public_key(self, did: str) -> Optional[bytes]:
        """
        Get public signing key for DID

        Args:
            did: DID to query

        Returns:
            Public key bytes if found
        """
        if did in self._cache:
            doc = self._cache[did]
            if not doc.revoked and doc.is_active:
                return doc.public_key
        return None

    def get_kem_public_key(self, did: str) -> Optional[bytes]:
        """
        Get public KEM key for DID

        Args:
            did: DID to query

        Returns:
            Public KEM key bytes if found
        """
        if did in self._cache:
            doc = self._cache[did]
            if not doc.revoked and doc.is_active:
                return doc.public_kem_key
        return None
