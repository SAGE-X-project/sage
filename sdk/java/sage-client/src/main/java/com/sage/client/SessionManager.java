package com.sage.client;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Session manager for handling multiple sessions
 */
public class SessionManager {
    private final Map<String, Session> sessions;
    private final int maxSessions;

    public SessionManager(int maxSessions) {
        this.sessions = new ConcurrentHashMap<>();
        this.maxSessions = maxSessions;
    }

    /**
     * Add session
     */
    public void addSession(Session session) throws SageException.SessionException {
        cleanupExpired();

        if (sessions.size() >= maxSessions) {
            throw new SageException.SessionException(
                    String.format("Too many sessions (%d/%d)", sessions.size(), maxSessions)
            );
        }

        sessions.put(session.getSessionId(), session);
    }

    /**
     * Get session by ID
     */
    public Session getSession(String sessionId) {
        Session session = sessions.get(sessionId);
        if (session != null && session.isExpired()) {
            sessions.remove(sessionId);
            return null;
        }
        return session;
    }

    /**
     * Remove session
     */
    public void removeSession(String sessionId) {
        sessions.remove(sessionId);
    }

    /**
     * Cleanup expired sessions
     */
    public int cleanupExpired() {
        List<String> expiredIds = new ArrayList<>();
        for (Map.Entry<String, Session> entry : sessions.entrySet()) {
            if (entry.getValue().isExpired()) {
                expiredIds.add(entry.getKey());
            }
        }

        for (String id : expiredIds) {
            sessions.remove(id);
        }

        return expiredIds.size();
    }

    /**
     * Count active sessions
     */
    public int count() {
        cleanupExpired();
        return sessions.size();
    }

    /**
     * Clear all sessions
     */
    public void clear() {
        sessions.clear();
    }
}
