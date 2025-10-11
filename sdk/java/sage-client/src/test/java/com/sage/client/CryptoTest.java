package com.sage.client;

import com.sage.client.types.KeyPair;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Crypto utilities
 */
class CryptoTest {

    @Test
    void testEd25519KeyPairGeneration() throws SageException.CryptoException {
        KeyPair keyPair = Crypto.generateEd25519KeyPair();

        assertNotNull(keyPair);
        assertNotNull(keyPair.getPublicKey());
        assertNotNull(keyPair.getPrivateKey());
        assertEquals(32, keyPair.getPublicKey().length);
        assertEquals(32, keyPair.getPrivateKey().length);
    }

    @Test
    void testX25519KeyPairGeneration() throws SageException.CryptoException {
        KeyPair keyPair = Crypto.generateX25519KeyPair();

        assertNotNull(keyPair);
        assertNotNull(keyPair.getPublicKey());
        assertNotNull(keyPair.getPrivateKey());
        assertEquals(32, keyPair.getPublicKey().length);
        assertEquals(32, keyPair.getPrivateKey().length);
    }

    @Test
    void testSignAndVerify() throws SageException.CryptoException {
        KeyPair keyPair = Crypto.generateEd25519KeyPair();
        byte[] message = "Test message".getBytes();

        byte[] signature = Crypto.sign(message, keyPair.getPrivateKey());
        assertNotNull(signature);
        assertEquals(64, signature.length);

        boolean isValid = Crypto.verify(message, signature, keyPair.getPublicKey());
        assertTrue(isValid);
    }

    @Test
    void testVerifyInvalidSignature() throws SageException.CryptoException {
        KeyPair keyPair = Crypto.generateEd25519KeyPair();
        byte[] message = "Test message".getBytes();
        byte[] signature = Crypto.sign(message, keyPair.getPrivateKey());

        byte[] tamperedMessage = "Tampered message".getBytes();
        boolean isValid = Crypto.verify(tamperedMessage, signature, keyPair.getPublicKey());
        assertFalse(isValid);
    }

    @Test
    void testBase64EncodeAndDecode() throws SageException.CryptoException {
        byte[] data = "Hello, World!".getBytes();
        String encoded = Crypto.base64Encode(data);
        assertNotNull(encoded);

        byte[] decoded = Crypto.base64Decode(encoded);
        assertArrayEquals(data, decoded);
    }

    @Test
    void testDeriveSharedSecret() throws SageException.CryptoException {
        KeyPair aliceKeyPair = Crypto.generateX25519KeyPair();
        KeyPair bobKeyPair = Crypto.generateX25519KeyPair();

        byte[] aliceShared = Crypto.deriveSharedSecret(
                aliceKeyPair.getPrivateKey(),
                bobKeyPair.getPublicKey()
        );

        byte[] bobShared = Crypto.deriveSharedSecret(
                bobKeyPair.getPrivateKey(),
                aliceKeyPair.getPublicKey()
        );

        assertArrayEquals(aliceShared, bobShared);
        assertEquals(32, aliceShared.length);
    }

    @Test
    void testHpkeEncryptAndDecrypt() throws SageException.CryptoException {
        KeyPair recipientKeyPair = Crypto.generateX25519KeyPair();
        byte[] plaintext = "Secret message".getBytes();

        Crypto.HpkeSetupResult senderSetup = Crypto.setupHpkeSender(recipientKeyPair.getPublicKey());
        byte[] ciphertext = senderSetup.getContext().seal(plaintext);

        Crypto.HpkeContext receiverContext = Crypto.setupHpkeReceiver(
                senderSetup.getEncapsulatedKey(),
                recipientKeyPair.getPrivateKey()
        );
        byte[] decrypted = receiverContext.open(ciphertext);

        assertArrayEquals(plaintext, decrypted);
    }

    @Test
    void testHpkeMultipleMessages() throws SageException.CryptoException {
        KeyPair recipientKeyPair = Crypto.generateX25519KeyPair();

        Crypto.HpkeSetupResult senderSetup = Crypto.setupHpkeSender(recipientKeyPair.getPublicKey());
        Crypto.HpkeContext senderContext = senderSetup.getContext();

        Crypto.HpkeContext receiverContext = Crypto.setupHpkeReceiver(
                senderSetup.getEncapsulatedKey(),
                recipientKeyPair.getPrivateKey()
        );

        for (int i = 0; i < 5; i++) {
            byte[] plaintext = ("Message " + i).getBytes();
            byte[] ciphertext = senderContext.seal(plaintext);
            byte[] decrypted = receiverContext.open(ciphertext);
            assertArrayEquals(plaintext, decrypted);
        }
    }
}
