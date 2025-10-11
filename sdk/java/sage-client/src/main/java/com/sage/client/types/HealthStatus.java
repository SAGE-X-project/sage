package com.sage.client.types;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Health status response
 */
public class HealthStatus {
    @JsonProperty("status")
    private String status;

    @JsonProperty("timestamp")
    private long timestamp;

    @JsonProperty("sessions")
    private SessionInfo sessions;

    public HealthStatus() {
    }

    public String getStatus() {
        return status;
    }

    public void setStatus(String status) {
        this.status = status;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }

    public SessionInfo getSessions() {
        return sessions;
    }

    public void setSessions(SessionInfo sessions) {
        this.sessions = sessions;
    }

    public static class SessionInfo {
        @JsonProperty("active")
        private int active;

        @JsonProperty("total")
        private int total;

        public SessionInfo() {
        }

        public int getActive() {
            return active;
        }

        public void setActive(int active) {
            this.active = active;
        }

        public int getTotal() {
            return total;
        }

        public void setTotal(int total) {
            this.total = total;
        }
    }
}
