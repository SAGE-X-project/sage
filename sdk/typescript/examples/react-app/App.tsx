/**
 * SAGE React Example App
 * Demonstrates using SAGE SDK with React hooks
 */

import React, { useState, useEffect } from 'react';
import {
  useSAGE,
  useSessions,
  useSecureMessaging,
  useSAGEEvents,
  useHandshake,
  useCrypto,
} from '@sage-x/sdk';
import type { SAGEEvent } from '@sage-x/sdk';

function App() {
  return (
    <div className="app">
      <h1>SAGE React Example</h1>
      <SAGEDemo />
    </div>
  );
}

function SAGEDemo() {
  const { client, isInitialized, did, error: initError, initialize } = useSAGE();
  const [message, setMessage] = useState('');
  const [events, setEvents] = useState<SAGEEvent[]>([]);

  // Listen to all SAGE events
  useSAGEEvents(client, 'handshake:initiated', (event) => {
    setEvents(prev => [...prev, event]);
  });

  useSAGEEvents(client, 'handshake:completed', (event) => {
    setEvents(prev => [...prev, event]);
  });

  useSAGEEvents(client, 'session:created', (event) => {
    setEvents(prev => [...prev, event]);
  });

  useSAGEEvents(client, 'message:sent', (event) => {
    setEvents(prev => [...prev, event]);
  });

  useSAGEEvents(client, 'message:received', (event) => {
    setEvents(prev => [...prev, event]);
  });

  useEffect(() => {
    initialize();
  }, [initialize]);

  if (!isInitialized) {
    return (
      <div className="loading">
        <p>Initializing SAGE client...</p>
        {initError && <p className="error">Error: {initError.message}</p>}
      </div>
    );
  }

  return (
    <div className="sage-demo">
      <IdentitySection did={did} />
      <CryptoSection client={client} />
      <SessionsSection client={client} />
      <MessagingSection client={client} />
      <EventsSection events={events} />
    </div>
  );
}

function IdentitySection({ did }: { did: string | null }) {
  return (
    <section className="identity-section">
      <h2>Identity</h2>
      <div className="identity-info">
        <label>DID:</label>
        <code>{did}</code>
      </div>
    </section>
  );
}

function CryptoSection({ client }: { client: any }) {
  const { generateKeyPair, sign, verify, loading, error } = useCrypto(client);
  const [keyPair, setKeyPair] = useState<any>(null);
  const [signature, setSignature] = useState<Uint8Array | null>(null);
  const [verifyResult, setVerifyResult] = useState<boolean | null>(null);

  const handleGenerateKey = async () => {
    const kp = await generateKeyPair('Ed25519');
    setKeyPair(kp);
  };

  const handleSign = async () => {
    const sig = await sign('Hello SAGE!');
    setSignature(sig);
  };

  const handleVerify = async () => {
    if (signature && keyPair) {
      const result = await verify('Hello SAGE!', signature, keyPair.publicKey);
      setVerifyResult(result);
    }
  };

  return (
    <section className="crypto-section">
      <h2>Cryptographic Operations</h2>

      <div className="crypto-actions">
        <button onClick={handleGenerateKey} disabled={loading}>
          Generate Ed25519 Key Pair
        </button>

        {keyPair && (
          <>
            <div className="key-info">
              <p>Public Key: {keyPair.publicKey.length} bytes</p>
              <p>Private Key: {keyPair.privateKey.length} bytes</p>
            </div>

            <button onClick={handleSign} disabled={loading}>
              Sign Message
            </button>
          </>
        )}

        {signature && (
          <>
            <div className="signature-info">
              <p>Signature: {signature.length} bytes</p>
            </div>

            <button onClick={handleVerify} disabled={loading}>
              Verify Signature
            </button>
          </>
        )}

        {verifyResult !== null && (
          <div className={`verify-result ${verifyResult ? 'success' : 'error'}`}>
            {verifyResult ? '✓ Signature Valid' : '✗ Signature Invalid'}
          </div>
        )}

        {error && <p className="error">Error: {error.message}</p>}
      </div>
    </section>
  );
}

function SessionsSection({ client }: { client: any }) {
  const { sessions, loading, error, refreshSessions } = useSessions(client);

  return (
    <section className="sessions-section">
      <h2>Active Sessions</h2>

      <div className="sessions-header">
        <span>Total: {sessions.length}</span>
        <button onClick={refreshSessions} disabled={loading}>
          Refresh
        </button>
      </div>

      <div className="sessions-list">
        {sessions.map((session) => (
          <div key={session.id} className="session-item">
            <div>
              <strong>ID:</strong> {session.id.substring(0, 16)}...
            </div>
            <div>
              <strong>Created:</strong> {session.createdAt.toLocaleString()}
            </div>
            <div>
              <strong>Expires:</strong> {session.expiresAt.toLocaleString()}
            </div>
          </div>
        ))}

        {sessions.length === 0 && <p>No active sessions</p>}
      </div>

      {error && <p className="error">Error: {error.message}</p>}
    </section>
  );
}

function MessagingSection({ client }: { client: any }) {
  const [sessionID, setSessionID] = useState('demo-session');
  const [messageText, setMessageText] = useState('');
  const [sentMessages, setSentMessages] = useState<string[]>([]);
  const [receivedMessages, setReceivedMessages] = useState<string[]>([]);

  const { sendMessage, receiveMessage, sending, receiving, error } = useSecureMessaging(
    client,
    sessionID
  );

  const handleSend = async () => {
    if (!messageText.trim()) return;

    try {
      await sendMessage(messageText);
      setSentMessages(prev => [...prev, messageText]);
      setMessageText('');
    } catch (err) {
      console.error('Failed to send message:', err);
    }
  };

  return (
    <section className="messaging-section">
      <h2>Secure Messaging</h2>

      <div className="messaging-controls">
        <input
          type="text"
          placeholder="Session ID"
          value={sessionID}
          onChange={(e) => setSessionID(e.target.value)}
        />

        <textarea
          placeholder="Type your message..."
          value={messageText}
          onChange={(e) => setMessageText(e.target.value)}
          rows={3}
        />

        <button onClick={handleSend} disabled={sending || !messageText.trim()}>
          {sending ? 'Sending...' : 'Send Encrypted Message'}
        </button>
      </div>

      <div className="messages-display">
        <div className="sent-messages">
          <h3>Sent Messages ({sentMessages.length})</h3>
          {sentMessages.map((msg, i) => (
            <div key={i} className="message sent">
              {msg}
            </div>
          ))}
        </div>

        <div className="received-messages">
          <h3>Received Messages ({receivedMessages.length})</h3>
          {receivedMessages.map((msg, i) => (
            <div key={i} className="message received">
              {msg}
            </div>
          ))}
        </div>
      </div>

      {error && <p className="error">Error: {error.message}</p>}
    </section>
  );
}

function EventsSection({ events }: { events: SAGEEvent[] }) {
  return (
    <section className="events-section">
      <h2>SAGE Events</h2>

      <div className="events-list">
        {events.map((event, i) => (
          <div key={i} className="event-item">
            <span className="event-type">{event.type}</span>
            <span className="event-time">
              {event.timestamp.toLocaleTimeString()}
            </span>
            <code className="event-data">
              {JSON.stringify(event.data, null, 2)}
            </code>
          </div>
        ))}

        {events.length === 0 && <p>No events yet</p>}
      </div>
    </section>
  );
}

export default App;
