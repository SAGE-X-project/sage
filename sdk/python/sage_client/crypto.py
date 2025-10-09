"""
SAGE Cryptography Module

Provides Ed25519 signing, X25519 key exchange, and HPKE encryption.
"""

import os
import base64
import hashlib
from typing import Tuple, Optional
from cryptography.hazmat.primitives.asymmetric import ed25519, x25519
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.kdf.hkdf import HKDF
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

from sage_client.types import KeyPair
from sage_client.exceptions import CryptoError, SignatureError, EncryptionError


class Crypto:
    """Cryptographic operations for SAGE"""

    @staticmethod
    def generate_ed25519_keypair() -> KeyPair:
        """
        Generate Ed25519 key pair for signing

        Returns:
            KeyPair: Private and public key pair
        """
        try:
            private_key = ed25519.Ed25519PrivateKey.generate()
            public_key = private_key.public_key()

            private_bytes = private_key.private_bytes(
                encoding=serialization.Encoding.Raw,
                format=serialization.PrivateFormat.Raw,
                encryption_algorithm=serialization.NoEncryption(),
            )

            public_bytes = public_key.public_bytes(
                encoding=serialization.Encoding.Raw,
                format=serialization.PublicFormat.Raw,
            )

            return KeyPair(
                private_key=private_bytes, public_key=public_bytes, key_type="Ed25519"
            )
        except Exception as e:
            raise CryptoError(f"Failed to generate Ed25519 keypair: {e}")

    @staticmethod
    def generate_x25519_keypair() -> KeyPair:
        """
        Generate X25519 key pair for key exchange

        Returns:
            KeyPair: Private and public key pair
        """
        try:
            private_key = x25519.X25519PrivateKey.generate()
            public_key = private_key.public_key()

            private_bytes = private_key.private_bytes(
                encoding=serialization.Encoding.Raw,
                format=serialization.PrivateFormat.Raw,
                encryption_algorithm=serialization.NoEncryption(),
            )

            public_bytes = public_key.public_bytes(
                encoding=serialization.Encoding.Raw,
                format=serialization.PublicFormat.Raw,
            )

            return KeyPair(
                private_key=private_bytes, public_key=public_bytes, key_type="X25519"
            )
        except Exception as e:
            raise CryptoError(f"Failed to generate X25519 keypair: {e}")

    @staticmethod
    def sign(message: bytes, private_key_bytes: bytes) -> bytes:
        """
        Sign message with Ed25519 private key

        Args:
            message: Message to sign
            private_key_bytes: Ed25519 private key (32 bytes)

        Returns:
            Signature bytes (64 bytes)
        """
        try:
            private_key = ed25519.Ed25519PrivateKey.from_private_bytes(private_key_bytes)
            signature = private_key.sign(message)
            return signature
        except Exception as e:
            raise CryptoError(f"Failed to sign message: {e}")

    @staticmethod
    def verify(message: bytes, signature: bytes, public_key_bytes: bytes) -> bool:
        """
        Verify Ed25519 signature

        Args:
            message: Original message
            signature: Signature to verify (64 bytes)
            public_key_bytes: Ed25519 public key (32 bytes)

        Returns:
            True if signature is valid
        """
        try:
            public_key = ed25519.Ed25519PublicKey.from_public_bytes(public_key_bytes)
            public_key.verify(signature, message)
            return True
        except Exception as e:
            raise SignatureError(f"Signature verification failed: {e}")

    @staticmethod
    def compute_dh(private_key_bytes: bytes, public_key_bytes: bytes) -> bytes:
        """
        Compute X25519 Diffie-Hellman shared secret

        Args:
            private_key_bytes: X25519 private key (32 bytes)
            public_key_bytes: X25519 public key (32 bytes)

        Returns:
            Shared secret (32 bytes)
        """
        try:
            private_key = x25519.X25519PrivateKey.from_private_bytes(private_key_bytes)
            public_key = x25519.X25519PublicKey.from_public_bytes(public_key_bytes)
            shared_secret = private_key.exchange(public_key)
            return shared_secret
        except Exception as e:
            raise CryptoError(f"Failed to compute DH: {e}")

    @staticmethod
    def derive_key(shared_secret: bytes, info: bytes, length: int = 32) -> bytes:
        """
        Derive key from shared secret using HKDF

        Args:
            shared_secret: Shared secret from key exchange
            info: Context information
            length: Desired key length (default: 32 bytes)

        Returns:
            Derived key
        """
        try:
            hkdf = HKDF(
                algorithm=hashes.SHA256(), length=length, salt=None, info=info
            )
            key = hkdf.derive(shared_secret)
            return key
        except Exception as e:
            raise CryptoError(f"Failed to derive key: {e}")

    @staticmethod
    def encrypt_aes_gcm(
        plaintext: bytes, key: bytes, nonce: Optional[bytes] = None
    ) -> Tuple[bytes, bytes]:
        """
        Encrypt with AES-GCM

        Args:
            plaintext: Data to encrypt
            key: Encryption key (32 bytes for AES-256)
            nonce: Nonce (12 bytes). If None, generates random nonce.

        Returns:
            Tuple of (ciphertext, nonce)
        """
        try:
            if nonce is None:
                nonce = os.urandom(12)

            aesgcm = AESGCM(key)
            ciphertext = aesgcm.encrypt(nonce, plaintext, None)
            return (ciphertext, nonce)
        except Exception as e:
            raise EncryptionError(f"Failed to encrypt: {e}")

    @staticmethod
    def decrypt_aes_gcm(ciphertext: bytes, key: bytes, nonce: bytes) -> bytes:
        """
        Decrypt with AES-GCM

        Args:
            ciphertext: Data to decrypt
            key: Decryption key (32 bytes)
            nonce: Nonce (12 bytes)

        Returns:
            Plaintext
        """
        try:
            aesgcm = AESGCM(key)
            plaintext = aesgcm.decrypt(nonce, ciphertext, None)
            return plaintext
        except Exception as e:
            raise EncryptionError(f"Failed to decrypt: {e}")

    @staticmethod
    def hash_sha256(data: bytes) -> bytes:
        """
        Compute SHA-256 hash

        Args:
            data: Data to hash

        Returns:
            Hash bytes (32 bytes)
        """
        return hashlib.sha256(data).digest()

    @staticmethod
    def base64_encode(data: bytes) -> str:
        """Encode bytes to base64 string"""
        return base64.b64encode(data).decode("utf-8")

    @staticmethod
    def base64_decode(data: str) -> bytes:
        """Decode base64 string to bytes"""
        try:
            return base64.b64decode(data)
        except Exception as e:
            raise CryptoError(f"Failed to decode base64: {e}")


class HPKEContext:
    """
    Simplified HPKE context for encryption/decryption

    Note: This is a simplified implementation. For production use,
    consider using a full HPKE library like pyhpke.
    """

    def __init__(self, key: bytes):
        """
        Initialize HPKE context

        Args:
            key: Shared secret key
        """
        self.key = key
        self.sequence = 0

    def seal(self, plaintext: bytes, aad: Optional[bytes] = None) -> bytes:
        """
        Encrypt plaintext (HPKE seal operation)

        Args:
            plaintext: Data to encrypt
            aad: Additional authenticated data (optional)

        Returns:
            Ciphertext (includes nonce)
        """
        try:
            # Derive per-message key
            nonce = self.sequence.to_bytes(12, byteorder="big")
            self.sequence += 1

            aesgcm = AESGCM(self.key)
            ciphertext = aesgcm.encrypt(nonce, plaintext, aad)

            # Prepend nonce to ciphertext
            return nonce + ciphertext
        except Exception as e:
            raise EncryptionError(f"HPKE seal failed: {e}")

    def open(self, ciphertext: bytes, aad: Optional[bytes] = None) -> bytes:
        """
        Decrypt ciphertext (HPKE open operation)

        Args:
            ciphertext: Data to decrypt (includes nonce)
            aad: Additional authenticated data (optional)

        Returns:
            Plaintext
        """
        try:
            # Extract nonce from ciphertext
            nonce = ciphertext[:12]
            actual_ciphertext = ciphertext[12:]

            aesgcm = AESGCM(self.key)
            plaintext = aesgcm.decrypt(nonce, actual_ciphertext, aad)

            self.sequence += 1
            return plaintext
        except Exception as e:
            raise EncryptionError(f"HPKE open failed: {e}")


def setup_hpke_sender(
    receiver_public_key: bytes, sender_keypair: Optional[KeyPair] = None
) -> Tuple[HPKEContext, bytes]:
    """
    Setup HPKE as sender (encapsulation)

    Args:
        receiver_public_key: Receiver's X25519 public key
        sender_keypair: Sender's X25519 keypair (optional, generates ephemeral if None)

    Returns:
        Tuple of (HPKE context, encapsulated key)
    """
    if sender_keypair is None:
        sender_keypair = Crypto.generate_x25519_keypair()

    # Compute shared secret
    shared_secret = Crypto.compute_dh(sender_keypair.private_key, receiver_public_key)

    # Derive encryption key
    info = b"SAGE HPKE v1"
    key = Crypto.derive_key(shared_secret, info)

    context = HPKEContext(key)
    return (context, sender_keypair.public_key)


def setup_hpke_receiver(
    encapsulated_key: bytes, receiver_private_key: bytes
) -> HPKEContext:
    """
    Setup HPKE as receiver (decapsulation)

    Args:
        encapsulated_key: Sender's ephemeral public key
        receiver_private_key: Receiver's X25519 private key

    Returns:
        HPKE context
    """
    # Compute shared secret
    shared_secret = Crypto.compute_dh(receiver_private_key, encapsulated_key)

    # Derive encryption key
    info = b"SAGE HPKE v1"
    key = Crypto.derive_key(shared_secret, info)

    return HPKEContext(key)
