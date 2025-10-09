package com.sage.client.types;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * KEM public key response
 */
public class KemPublicKeyResponse {
    @JsonProperty("kem_public_key")
    private String kemPublicKey;

    public KemPublicKeyResponse() {
    }

    public String getKemPublicKey() {
        return kemPublicKey;
    }

    public void setKemPublicKey(String kemPublicKey) {
        this.kemPublicKey = kemPublicKey;
    }
}
