package com.sage.client.types;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Agent metadata for registration
 */
public class AgentMetadata {
    @JsonProperty("did")
    private String did;

    @JsonProperty("name")
    private String name;

    @JsonProperty("is_active")
    private boolean isActive;

    @JsonProperty("public_key")
    private String publicKey;

    @JsonProperty("public_kem_key")
    private String publicKemKey;

    public AgentMetadata() {
    }

    public AgentMetadata(String did, String name, boolean isActive, String publicKey, String publicKemKey) {
        this.did = did;
        this.name = name;
        this.isActive = isActive;
        this.publicKey = publicKey;
        this.publicKemKey = publicKemKey;
    }

    public String getDid() {
        return did;
    }

    public void setDid(String did) {
        this.did = did;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public boolean isActive() {
        return isActive;
    }

    public void setActive(boolean active) {
        isActive = active;
    }

    public String getPublicKey() {
        return publicKey;
    }

    public void setPublicKey(String publicKey) {
        this.publicKey = publicKey;
    }

    public String getPublicKemKey() {
        return publicKemKey;
    }

    public void setPublicKemKey(String publicKemKey) {
        this.publicKemKey = publicKemKey;
    }
}
