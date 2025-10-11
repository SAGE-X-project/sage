package com.sage.client.types;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Handshake request payload
 */
public class HandshakeRequest {
    @JsonProperty("sender_did")
    private String senderDid;

    @JsonProperty("receiver_did")
    private String receiverDid;

    @JsonProperty("message")
    private String message;

    @JsonProperty("timestamp")
    private long timestamp;

    @JsonProperty("signature")
    private String signature;

    public HandshakeRequest() {
    }

    public HandshakeRequest(String senderDid, String receiverDid, String message, long timestamp, String signature) {
        this.senderDid = senderDid;
        this.receiverDid = receiverDid;
        this.message = message;
        this.timestamp = timestamp;
        this.signature = signature;
    }

    public String getSenderDid() {
        return senderDid;
    }

    public void setSenderDid(String senderDid) {
        this.senderDid = senderDid;
    }

    public String getReceiverDid() {
        return receiverDid;
    }

    public void setReceiverDid(String receiverDid) {
        this.receiverDid = receiverDid;
    }

    public String getMessage() {
        return message;
    }

    public void setMessage(String message) {
        this.message = message;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }

    public String getSignature() {
        return signature;
    }

    public void setSignature(String signature) {
        this.signature = signature;
    }
}
