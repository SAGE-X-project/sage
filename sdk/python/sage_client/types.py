"""
SAGE Type Definitions
"""

from typing import Dict, Any, Optional
from datetime import datetime
from pydantic import BaseModel, Field


class KeyPair(BaseModel):
    """Cryptographic key pair"""

    private_key: bytes
    public_key: bytes
    key_type: str = "Ed25519"


class Message(BaseModel):
    """Generic message structure"""

    sender_did: str
    receiver_did: str
    content: bytes
    timestamp: int
    signature: Optional[bytes] = None


class HandshakeRequest(BaseModel):
    """HPKE handshake request"""

    sender_did: str
    receiver_did: str
    message: str = Field(..., description="Base64-encoded encrypted handshake payload")
    timestamp: int
    signature: str = Field(..., description="Base64-encoded Ed25519 signature")


class HandshakeResponse(BaseModel):
    """HPKE handshake response"""

    session_id: str
    response: str = Field(..., description="Base64-encoded encrypted response")


class MessageRequest(BaseModel):
    """Message send request"""

    sender_did: str
    receiver_did: str
    message: str = Field(..., description="Base64-encoded encrypted message")
    timestamp: int
    signature: str = Field(..., description="Base64-encoded Ed25519 signature")


class MessageResponse(BaseModel):
    """Message send response"""

    response: str = Field(..., description="Base64-encoded encrypted response")
    session_id: Optional[str] = None


class AgentMetadata(BaseModel):
    """Agent metadata for registration"""

    did: str
    name: str
    is_active: bool = True
    public_key: str = Field(..., description="Base64-encoded Ed25519 public key")
    public_kem_key: str = Field(..., description="Base64-encoded X25519 public key")


class SessionInfo(BaseModel):
    """Session information"""

    session_id: str
    client_did: str
    server_did: str
    created_at: datetime
    expires_at: datetime
    last_activity: datetime
    metadata: Dict[str, Any] = Field(default_factory=dict)


class HealthStatus(BaseModel):
    """Server health status"""

    status: str
    timestamp: datetime
    sessions: Optional[Dict[str, int]] = None


class ErrorResponse(BaseModel):
    """Error response from server"""

    error: str
    code: Optional[str] = None
    details: Optional[Dict[str, Any]] = None
