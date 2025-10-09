package com.sage.client.types;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Handshake response payload
 */
public class HandshakeResponse {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("server_did")
    private String serverDid;

    @JsonProperty("message")
    private String message;

    public HandshakeResponse() {
    }

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public String getServerDid() {
        return serverDid;
    }

    public void setServerDid(String serverDid) {
        this.serverDid = serverDid;
    }

    public String getMessage() {
        return message;
    }

    public void setMessage(String message) {
        this.message = message;
    }
}
