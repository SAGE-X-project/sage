package com.sage.client.types;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Message response payload
 */
public class MessageResponse {
    @JsonProperty("response")
    private String response;

    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("timestamp")
    private long timestamp;

    public MessageResponse() {
    }

    public String getResponse() {
        return response;
    }

    public void setResponse(String response) {
        this.response = response;
    }

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }
}
