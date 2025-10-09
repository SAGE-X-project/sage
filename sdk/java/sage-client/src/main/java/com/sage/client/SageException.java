package com.sage.client;

/**
 * Base exception for SAGE client errors
 */
public class SageException extends Exception {
    public SageException(String message) {
        super(message);
    }

    public SageException(String message, Throwable cause) {
        super(message, cause);
    }

    /**
     * Cryptography-related errors
     */
    public static class CryptoException extends SageException {
        public CryptoException(String message) {
            super(message);
        }

        public CryptoException(String message, Throwable cause) {
            super(message, cause);
        }
    }

    /**
     * Session-related errors
     */
    public static class SessionException extends SageException {
        public SessionException(String message) {
            super(message);
        }

        public SessionException(String message, Throwable cause) {
            super(message, cause);
        }
    }

    /**
     * Network-related errors
     */
    public static class NetworkException extends SageException {
        public NetworkException(String message) {
            super(message);
        }

        public NetworkException(String message, Throwable cause) {
            super(message, cause);
        }
    }

    /**
     * DID-related errors
     */
    public static class DidException extends SageException {
        public DidException(String message) {
            super(message);
        }

        public DidException(String message, Throwable cause) {
            super(message, cause);
        }
    }

    /**
     * Session expired error
     */
    public static class SessionExpiredException extends SessionException {
        public SessionExpiredException(String sessionId) {
            super("Session expired: " + sessionId);
        }
    }

    /**
     * Initialization error
     */
    public static class NotInitializedException extends SageException {
        public NotInitializedException(String message) {
            super(message);
        }
    }
}
