"""
SAGE Session Manager
"""

import time
from typing import Dict, Optional
from datetime import datetime, timedelta

from sage_client.crypto import HPKEContext
from sage_client.exceptions import SessionError, SessionExpiredError


class Session:
    """
    Represents a secure session between client and server
    """

    def __init__(
        self,
        session_id: str,
        client_did: str,
        server_did: str,
        hpke_context: HPKEContext,
        max_age: int = 3600,  # 1 hour default
    ):
        """
        Initialize session

        Args:
            session_id: Unique session identifier
            client_did: Client DID
            server_did: Server DID
            hpke_context: HPKE encryption context
            max_age: Maximum session age in seconds
        """
        self.session_id = session_id
        self.client_did = client_did
        self.server_did = server_did
        self.hpke_context = hpke_context
        self.created_at = datetime.now()
        self.expires_at = self.created_at + timedelta(seconds=max_age)
        self.last_activity = self.created_at
        self.message_count = 0

    def is_expired(self) -> bool:
        """Check if session has expired"""
        return datetime.now() > self.expires_at

    def update_activity(self) -> None:
        """Update last activity timestamp"""
        if self.is_expired():
            raise SessionExpiredError(f"Session {self.session_id} has expired")
        self.last_activity = datetime.now()

    def encrypt(self, plaintext: bytes) -> bytes:
        """
        Encrypt message using session context

        Args:
            plaintext: Data to encrypt

        Returns:
            Ciphertext
        """
        if self.is_expired():
            raise SessionExpiredError(f"Session {self.session_id} has expired")

        self.update_activity()
        self.message_count += 1
        return self.hpke_context.seal(plaintext)

    def decrypt(self, ciphertext: bytes) -> bytes:
        """
        Decrypt message using session context

        Args:
            ciphertext: Data to decrypt

        Returns:
            Plaintext
        """
        if self.is_expired():
            raise SessionExpiredError(f"Session {self.session_id} has expired")

        self.update_activity()
        return self.hpke_context.open(ciphertext)

    def __repr__(self) -> str:
        return (
            f"Session(id='{self.session_id}', "
            f"client='{self.client_did}', "
            f"server='{self.server_did}', "
            f"messages={self.message_count}, "
            f"expired={self.is_expired()})"
        )


class SessionManager:
    """
    Manages active sessions
    """

    def __init__(self, max_sessions: int = 100):
        """
        Initialize session manager

        Args:
            max_sessions: Maximum number of concurrent sessions
        """
        self._sessions: Dict[str, Session] = {}
        self._max_sessions = max_sessions

    def add_session(self, session: Session) -> None:
        """
        Add session to manager

        Args:
            session: Session to add

        Raises:
            SessionError: If too many sessions
        """
        # Clean up expired sessions first
        self.cleanup_expired()

        if len(self._sessions) >= self._max_sessions:
            raise SessionError(
                f"Too many sessions ({len(self._sessions)}). Max: {self._max_sessions}"
            )

        self._sessions[session.session_id] = session

    def get_session(self, session_id: str) -> Optional[Session]:
        """
        Get session by ID

        Args:
            session_id: Session identifier

        Returns:
            Session if found and not expired, None otherwise
        """
        session = self._sessions.get(session_id)
        if session is None:
            return None

        if session.is_expired():
            self.remove_session(session_id)
            return None

        return session

    def remove_session(self, session_id: str) -> None:
        """
        Remove session

        Args:
            session_id: Session identifier
        """
        self._sessions.pop(session_id, None)

    def cleanup_expired(self) -> int:
        """
        Remove all expired sessions

        Returns:
            Number of sessions removed
        """
        expired_ids = [
            sid for sid, session in self._sessions.items() if session.is_expired()
        ]

        for sid in expired_ids:
            del self._sessions[sid]

        return len(expired_ids)

    def get_active_sessions(self) -> list[Session]:
        """
        Get all active (non-expired) sessions

        Returns:
            List of active sessions
        """
        self.cleanup_expired()
        return list(self._sessions.values())

    def count(self) -> int:
        """
        Count active sessions

        Returns:
            Number of active sessions
        """
        self.cleanup_expired()
        return len(self._sessions)

    def clear(self) -> None:
        """Clear all sessions"""
        self._sessions.clear()

    def __repr__(self) -> str:
        active = len(self.get_active_sessions())
        return f"SessionManager(active={active}, max={self._max_sessions})"
