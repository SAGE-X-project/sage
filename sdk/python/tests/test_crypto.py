"""Tests for cryptography module"""

import pytest
from sage_client.crypto import Crypto, setup_hpke_sender, setup_hpke_receiver


def test_generate_ed25519_keypair():
    """Test Ed25519 keypair generation"""
    keypair = Crypto.generate_ed25519_keypair()
    assert len(keypair.private_key) == 32
    assert len(keypair.public_key) == 32
    assert keypair.key_type == "Ed25519"


def test_generate_x25519_keypair():
    """Test X25519 keypair generation"""
    keypair = Crypto.generate_x25519_keypair()
    assert len(keypair.private_key) == 32
    assert len(keypair.public_key) == 32
    assert keypair.key_type == "X25519"


def test_sign_and_verify():
    """Test Ed25519 signing and verification"""
    keypair = Crypto.generate_ed25519_keypair()
    message = b"Hello, World!"

    signature = Crypto.sign(message, keypair.private_key)
    assert len(signature) == 64

    is_valid = Crypto.verify(message, signature, keypair.public_key)
    assert is_valid


def test_verify_invalid_signature():
    """Test signature verification with invalid signature"""
    keypair = Crypto.generate_ed25519_keypair()
    message = b"Hello, World!"
    wrong_message = b"Goodbye, World!"

    signature = Crypto.sign(message, keypair.private_key)

    with pytest.raises(Exception):
        Crypto.verify(wrong_message, signature, keypair.public_key)


def test_hpke_encryption():
    """Test HPKE encryption/decryption"""
    receiver_keypair = Crypto.generate_x25519_keypair()

    # Sender: setup and encrypt
    sender_ctx, encapsulated_key = setup_hpke_sender(receiver_keypair.public_key)
    plaintext = b"Secret message"
    ciphertext = sender_ctx.seal(plaintext)

    # Receiver: setup and decrypt
    receiver_ctx = setup_hpke_receiver(encapsulated_key, receiver_keypair.private_key)
    decrypted = receiver_ctx.open(ciphertext)

    assert decrypted == plaintext


def test_base64_encoding():
    """Test base64 encoding/decoding"""
    data = b"Test data"
    encoded = Crypto.base64_encode(data)
    decoded = Crypto.base64_decode(encoded)
    assert decoded == data


def test_sha256_hash():
    """Test SHA-256 hashing"""
    data = b"Hash me"
    hash_value = Crypto.hash_sha256(data)
    assert len(hash_value) == 32
