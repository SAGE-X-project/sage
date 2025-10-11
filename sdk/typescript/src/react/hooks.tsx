/**
 * SAGE React Hooks
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { SAGEClient } from '../client';
import type {
  SAGEClientOptions,
  Session,
  KeyPair,
  EncryptedMessage,
  SAGEEvent,
  EventType,
} from '../types';

/**
 * Hook to use SAGE client
 */
export function useSAGE(options?: SAGEClientOptions) {
  const [client] = useState(() => new SAGEClient(options));
  const [isInitialized, setIsInitialized] = useState(false);
  const [did, setDID] = useState<string | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const initialize = useCallback(async (keyPair?: KeyPair) => {
    try {
      await client.initialize(keyPair);
      setDID(client.getDID());
      setIsInitialized(true);
      setError(null);
    } catch (err) {
      setError(err as Error);
      throw err;
    }
  }, [client]);

  return {
    client,
    isInitialized,
    did,
    error,
    initialize,
  };
}

/**
 * Hook to manage sessions
 */
export function useSessions(client: SAGEClient) {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const refreshSessions = useCallback(() => {
    try {
      const activeSessions = client.getAllSessions();
      setSessions(activeSessions);
      setError(null);
    } catch (err) {
      setError(err as Error);
    }
  }, [client]);

  const createSession = useCallback(async (
    clientKey: Uint8Array,
    serverKey: Uint8Array
  ): Promise<Session> => {
    setLoading(true);
    try {
      const session = await client.getSession(''); // Would use sessionManager
      setError(null);
      refreshSessions();
      return session!;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [client, refreshSessions]);

  const closeSession = useCallback(async (sessionID: string) => {
    setLoading(true);
    try {
      await client.closeSession(sessionID);
      setError(null);
      refreshSessions();
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [client, refreshSessions]);

  useEffect(() => {
    refreshSessions();
    const interval = setInterval(refreshSessions, 5000);
    return () => clearInterval(interval);
  }, [refreshSessions]);

  return {
    sessions,
    loading,
    error,
    createSession,
    closeSession,
    refreshSessions,
  };
}

/**
 * Hook to send and receive encrypted messages
 */
export function useSecureMessaging(client: SAGEClient, sessionID: string) {
  const [sending, setSending] = useState(false);
  const [receiving, setReceiving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const sendMessage = useCallback(async (message: string | Uint8Array): Promise<EncryptedMessage> => {
    setSending(true);
    try {
      const messageBytes = typeof message === 'string'
        ? new TextEncoder().encode(message)
        : message;

      const encrypted = await client.sendMessage(sessionID, messageBytes);
      setError(null);
      return encrypted;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setSending(false);
    }
  }, [client, sessionID]);

  const receiveMessage = useCallback(async (encrypted: EncryptedMessage): Promise<string> => {
    setReceiving(true);
    try {
      const plaintext = await client.receiveMessage(sessionID, encrypted);
      const message = new TextDecoder().decode(plaintext);
      setError(null);
      return message;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setReceiving(false);
    }
  }, [client, sessionID]);

  return {
    sendMessage,
    receiveMessage,
    sending,
    receiving,
    error,
  };
}

/**
 * Hook to listen to SAGE events
 */
export function useSAGEEvents(client: SAGEClient, eventType: EventType, handler: (event: SAGEEvent) => void) {
  const handlerRef = useRef(handler);

  useEffect(() => {
    handlerRef.current = handler;
  }, [handler]);

  useEffect(() => {
    const eventHandler = (event: SAGEEvent) => {
      handlerRef.current(event);
    };

    client.on(eventType, eventHandler);

    return () => {
      client.off(eventType, eventHandler);
    };
  }, [client, eventType]);
}

/**
 * Hook for handshake management
 */
export function useHandshake(client: SAGEClient) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [session, setSession] = useState<Session | null>(null);

  const initiateHandshake = useCallback(async (serverPublicKey: Uint8Array) => {
    setLoading(true);
    setError(null);

    try {
      const initiation = await client.initiateHandshake(serverPublicKey);
      // In real implementation, send initiation to server and wait for response
      return initiation;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [client]);

  const completeHandshake = useCallback(async (
    response: any,
    myEphemeralPrivateKey: Uint8Array
  ) => {
    setLoading(true);
    setError(null);

    try {
      const newSession = await client.completeHandshake(response, myEphemeralPrivateKey);
      setSession(newSession);
      return newSession;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [client]);

  return {
    initiateHandshake,
    completeHandshake,
    loading,
    error,
    session,
  };
}

/**
 * Hook for crypto operations
 */
export function useCrypto(client: SAGEClient) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const generateKeyPair = useCallback(async (type: 'Ed25519' | 'Secp256k1' | 'X25519') => {
    setLoading(true);
    try {
      const keyPair = await client.generateKeyPair(type);
      setError(null);
      return keyPair;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [client]);

  const sign = useCallback(async (message: string | Uint8Array) => {
    setLoading(true);
    try {
      const messageBytes = typeof message === 'string'
        ? new TextEncoder().encode(message)
        : message;

      const signature = await client.sign(messageBytes);
      setError(null);
      return signature;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [client]);

  const verify = useCallback(async (
    message: string | Uint8Array,
    signature: Uint8Array,
    publicKey: Uint8Array
  ) => {
    setLoading(true);
    try {
      const messageBytes = typeof message === 'string'
        ? new TextEncoder().encode(message)
        : message;

      const isValid = await client.verify(messageBytes, signature, publicKey);
      setError(null);
      return isValid;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [client]);

  return {
    generateKeyPair,
    sign,
    verify,
    loading,
    error,
  };
}
