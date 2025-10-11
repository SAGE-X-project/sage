package com.sage.client.types;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Server DID response
 */
public class ServerDidResponse {
    @JsonProperty("did")
    private String did;

    public ServerDidResponse() {
    }

    public String getDid() {
        return did;
    }

    public void setDid(String did) {
        this.did = did;
    }
}
