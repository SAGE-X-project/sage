package com.sage.client.types;

/**
 * Cryptographic key pair
 */
public class KeyPair {
    private final byte[] publicKey;
    private final byte[] privateKey;

    public KeyPair(byte[] publicKey, byte[] privateKey) {
        this.publicKey = publicKey;
        this.privateKey = privateKey;
    }

    public byte[] getPublicKey() {
        return publicKey;
    }

    public byte[] getPrivateKey() {
        return privateKey;
    }
}
