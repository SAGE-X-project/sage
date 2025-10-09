package com.sage.client;

/**
 * Client configuration
 */
public class ClientConfig {
    private final String baseUrl;
    private final int timeoutSeconds;
    private final int maxSessions;

    private ClientConfig(Builder builder) {
        this.baseUrl = builder.baseUrl.replaceAll("/$", "");
        this.timeoutSeconds = builder.timeoutSeconds;
        this.maxSessions = builder.maxSessions;
    }

    public String getBaseUrl() {
        return baseUrl;
    }

    public int getTimeoutSeconds() {
        return timeoutSeconds;
    }

    public int getMaxSessions() {
        return maxSessions;
    }

    /**
     * Create new builder
     */
    public static Builder builder(String baseUrl) {
        return new Builder(baseUrl);
    }

    /**
     * Builder for ClientConfig
     */
    public static class Builder {
        private final String baseUrl;
        private int timeoutSeconds = 30;
        private int maxSessions = 100;

        public Builder(String baseUrl) {
            this.baseUrl = baseUrl;
        }

        public Builder timeoutSeconds(int timeoutSeconds) {
            this.timeoutSeconds = timeoutSeconds;
            return this;
        }

        public Builder maxSessions(int maxSessions) {
            this.maxSessions = maxSessions;
            return this;
        }

        public ClientConfig build() {
            return new ClientConfig(this);
        }
    }
}
