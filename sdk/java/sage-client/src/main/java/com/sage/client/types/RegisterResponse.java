package com.sage.client.types;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Register agent response
 */
public class RegisterResponse {
    @JsonProperty("success")
    private boolean success;

    @JsonProperty("message")
    private String message;

    public RegisterResponse() {
    }

    public boolean isSuccess() {
        return success;
    }

    public void setSuccess(boolean success) {
        this.success = success;
    }

    public String getMessage() {
        return message;
    }

    public void setMessage(String message) {
        this.message = message;
    }
}
