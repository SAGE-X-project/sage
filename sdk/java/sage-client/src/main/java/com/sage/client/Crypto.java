package com.sage.client;

import com.sage.client.types.KeyPair;
import org.bouncycastle.crypto.AsymmetricCipherKeyPair;
import org.bouncycastle.crypto.generators.Ed25519KeyPairGenerator;
import org.bouncycastle.crypto.generators.X25519KeyPairGenerator;
import org.bouncycastle.crypto.params.*;
import org.bouncycastle.crypto.signers.Ed25519Signer;
import org.bouncycastle.jce.provider.BouncyCastleProvider;

import javax.crypto.Cipher;
import javax.crypto.KeyAgreement;
import javax.crypto.spec.GCMParameterSpec;
import javax.crypto.spec.SecretKeySpec;
import java.security.*;
import java.util.Arrays;
import java.util.Base64;

/**
 * Cryptography utilities for SAGE client
 */
public class Crypto {
    private static final int GCM_TAG_LENGTH = 128;
    private static final int GCM_IV_LENGTH = 12;

    static {
        Security.addProvider(new BouncyCastleProvider());
    }

    /**
     * Generate Ed25519 keypair for signing
     */
    public static KeyPair generateEd25519KeyPair() throws SageException.CryptoException {
        try {
            Ed25519KeyPairGenerator generator = new Ed25519KeyPairGenerator();
            generator.init(new Ed25519KeyGenerationParameters(new SecureRandom()));
            AsymmetricCipherKeyPair keyPair = generator.generateKeyPair();

            Ed25519PrivateKeyParameters privateKey = (Ed25519PrivateKeyParameters) keyPair.getPrivate();
            Ed25519PublicKeyParameters publicKey = (Ed25519PublicKeyParameters) keyPair.getPublic();

            return new KeyPair(publicKey.getEncoded(), privateKey.getEncoded());
        } catch (Exception e) {
            throw new SageException.CryptoException("Failed to generate Ed25519 keypair", e);
        }
    }

    /**
     * Generate X25519 keypair for key exchange
     */
    public static KeyPair generateX25519KeyPair() throws SageException.CryptoException {
        try {
            X25519KeyPairGenerator generator = new X25519KeyPairGenerator();
            generator.init(new X25519KeyGenerationParameters(new SecureRandom()));
            AsymmetricCipherKeyPair keyPair = generator.generateKeyPair();

            X25519PrivateKeyParameters privateKey = (X25519PrivateKeyParameters) keyPair.getPrivate();
            X25519PublicKeyParameters publicKey = (X25519PublicKeyParameters) keyPair.getPublic();

            return new KeyPair(publicKey.getEncoded(), privateKey.getEncoded());
        } catch (Exception e) {
            throw new SageException.CryptoException("Failed to generate X25519 keypair", e);
        }
    }

    /**
     * Sign message with Ed25519 private key
     */
    public static byte[] sign(byte[] message, byte[] privateKeyBytes) throws SageException.CryptoException {
        try {
            Ed25519PrivateKeyParameters privateKey = new Ed25519PrivateKeyParameters(privateKeyBytes, 0);
            Ed25519Signer signer = new Ed25519Signer();
            signer.init(true, privateKey);
            signer.update(message, 0, message.length);
            return signer.generateSignature();
        } catch (Exception e) {
            throw new SageException.CryptoException("Failed to sign message", e);
        }
    }

    /**
     * Verify Ed25519 signature
     */
    public static boolean verify(byte[] message, byte[] signature, byte[] publicKeyBytes) throws SageException.CryptoException {
        try {
            Ed25519PublicKeyParameters publicKey = new Ed25519PublicKeyParameters(publicKeyBytes, 0);
            Ed25519Signer verifier = new Ed25519Signer();
            verifier.init(false, publicKey);
            verifier.update(message, 0, message.length);
            return verifier.verifySignature(signature);
        } catch (Exception e) {
            throw new SageException.CryptoException("Failed to verify signature", e);
        }
    }

    /**
     * Perform X25519 key exchange
     */
    public static byte[] deriveSharedSecret(byte[] privateKeyBytes, byte[] publicKeyBytes) throws SageException.CryptoException {
        try {
            X25519PrivateKeyParameters privateKey = new X25519PrivateKeyParameters(privateKeyBytes, 0);
            X25519PublicKeyParameters publicKey = new X25519PublicKeyParameters(publicKeyBytes, 0);

            byte[] sharedSecret = new byte[32];
            privateKey.generateSecret(publicKey, sharedSecret, 0);
            return sharedSecret;
        } catch (Exception e) {
            throw new SageException.CryptoException("Failed to derive shared secret", e);
        }
    }

    /**
     * Encode bytes to Base64
     */
    public static String base64Encode(byte[] data) {
        return Base64.getEncoder().encodeToString(data);
    }

    /**
     * Decode Base64 string
     */
    public static byte[] base64Decode(String encoded) throws SageException.CryptoException {
        try {
            return Base64.getDecoder().decode(encoded);
        } catch (Exception e) {
            throw new SageException.CryptoException("Failed to decode Base64", e);
        }
    }

    /**
     * HPKE Context for encryption/decryption
     */
    public static class HpkeContext {
        private final byte[] key;
        private long sequence;

        public HpkeContext(byte[] key) {
            this.key = Arrays.copyOf(key, key.length);
            this.sequence = 0;
        }

        /**
         * Encrypt (seal) data
         */
        public byte[] seal(byte[] plaintext) throws SageException.CryptoException {
            try {
                SecretKeySpec keySpec = new SecretKeySpec(key, "AES");
                Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");

                byte[] nonce = new byte[GCM_IV_LENGTH];
                SecureRandom random = new SecureRandom();
                random.nextBytes(nonce);

                GCMParameterSpec gcmSpec = new GCMParameterSpec(GCM_TAG_LENGTH, nonce);
                cipher.init(Cipher.ENCRYPT_MODE, keySpec, gcmSpec);

                byte[] ciphertext = cipher.doFinal(plaintext);

                // Prepend nonce to ciphertext
                byte[] result = new byte[nonce.length + ciphertext.length];
                System.arraycopy(nonce, 0, result, 0, nonce.length);
                System.arraycopy(ciphertext, 0, result, nonce.length, ciphertext.length);

                sequence++;
                return result;
            } catch (Exception e) {
                throw new SageException.CryptoException("Failed to encrypt data", e);
            }
        }

        /**
         * Decrypt (open) data
         */
        public byte[] open(byte[] ciphertext) throws SageException.CryptoException {
            try {
                if (ciphertext.length < GCM_IV_LENGTH) {
                    throw new SageException.CryptoException("Ciphertext too short");
                }

                byte[] nonce = new byte[GCM_IV_LENGTH];
                System.arraycopy(ciphertext, 0, nonce, 0, GCM_IV_LENGTH);

                byte[] actualCiphertext = new byte[ciphertext.length - GCM_IV_LENGTH];
                System.arraycopy(ciphertext, GCM_IV_LENGTH, actualCiphertext, 0, actualCiphertext.length);

                SecretKeySpec keySpec = new SecretKeySpec(key, "AES");
                Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
                GCMParameterSpec gcmSpec = new GCMParameterSpec(GCM_TAG_LENGTH, nonce);
                cipher.init(Cipher.DECRYPT_MODE, keySpec, gcmSpec);

                return cipher.doFinal(actualCiphertext);
            } catch (Exception e) {
                throw new SageException.CryptoException("Failed to decrypt data", e);
            }
        }

        public long getSequence() {
            return sequence;
        }
    }

    /**
     * Setup HPKE sender (simplified implementation)
     */
    public static HpkeSetupResult setupHpkeSender(byte[] recipientPublicKey) throws SageException.CryptoException {
        // Generate ephemeral X25519 keypair
        KeyPair ephemeralKeyPair = generateX25519KeyPair();

        // Derive shared secret
        byte[] sharedSecret = deriveSharedSecret(ephemeralKeyPair.getPrivateKey(), recipientPublicKey);

        // Create HPKE context with shared secret as key (simplified - real HPKE uses KDF)
        HpkeContext context = new HpkeContext(sharedSecret);

        return new HpkeSetupResult(context, ephemeralKeyPair.getPublicKey());
    }

    /**
     * Setup HPKE receiver (simplified implementation)
     */
    public static HpkeContext setupHpkeReceiver(byte[] encapsulatedKey, byte[] recipientPrivateKey) throws SageException.CryptoException {
        // Derive shared secret
        byte[] sharedSecret = deriveSharedSecret(recipientPrivateKey, encapsulatedKey);

        // Create HPKE context
        return new HpkeContext(sharedSecret);
    }

    /**
     * HPKE setup result
     */
    public static class HpkeSetupResult {
        private final HpkeContext context;
        private final byte[] encapsulatedKey;

        public HpkeSetupResult(HpkeContext context, byte[] encapsulatedKey) {
            this.context = context;
            this.encapsulatedKey = encapsulatedKey;
        }

        public HpkeContext getContext() {
            return context;
        }

        public byte[] getEncapsulatedKey() {
            return encapsulatedKey;
        }
    }
}
