"""
SAGE Client Exceptions
"""


class SAGEError(Exception):
    """Base exception for all SAGE errors"""

    pass


class CryptoError(SAGEError):
    """Raised when cryptographic operations fail"""

    pass


class SessionError(SAGEError):
    """Raised when session operations fail"""

    pass


class NetworkError(SAGEError):
    """Raised when network operations fail"""

    pass


class ValidationError(SAGEError):
    """Raised when validation fails"""

    pass


class DIDError(SAGEError):
    """Raised when DID operations fail"""

    pass


class SignatureError(CryptoError):
    """Raised when signature verification fails"""

    pass


class EncryptionError(CryptoError):
    """Raised when encryption/decryption fails"""

    pass


class SessionExpiredError(SessionError):
    """Raised when session has expired"""

    pass


class ReplayAttackError(SAGEError):
    """Raised when replay attack is detected"""

    pass
