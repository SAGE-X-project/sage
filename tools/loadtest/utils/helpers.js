// SAGE Load Testing Helpers

import { check, group, sleep } from 'k6';
import http from 'k6/http';
import { Trend, Rate, Counter } from 'k6/metrics';
import crypto from 'k6/crypto';
import encoding from 'k6/encoding';

// Custom metrics
export const customMetrics = {
  handshakeDuration: new Trend('sage_handshake_duration'),
  messageDuration: new Trend('sage_message_duration'),
  signatureVerifyDuration: new Trend('sage_signature_verify_duration'),
  successRate: new Rate('sage_success_rate'),
  sessionCreated: new Counter('sage_sessions_created'),
  messagesPerSession: new Trend('sage_messages_per_session'),
};

// Generate random test data
export function generateRandomDID() {
  const randomAddress = '0x' + crypto.randomBytes(20).toString('hex');
  return `did:sage:ethereum:${randomAddress}`;
}

export function generateRandomKey() {
  return encoding.b64encode(crypto.randomBytes(32));
}

export function generateTimestamp() {
  return Math.floor(Date.now() / 1000);
}

// Mock signature generation (for load testing)
export function generateMockSignature(message) {
  const hash = crypto.sha256(message, 'hex');
  return encoding.b64encode(hash);
}

// Health check helper
export function healthCheck(baseUrl) {
  return group('Health Check', () => {
    const res = http.get(`${baseUrl}/debug/health`);

    const success = check(res, {
      'health check status is 200': (r) => r.status === 200,
      'health check has status field': (r) => {
        try {
          return JSON.parse(r.body).status === 'healthy';
        } catch {
          return false;
        }
      },
    });

    customMetrics.successRate.add(success);
    return res;
  });
}

// Get server KEM public key
export function getServerKemKey(baseUrl) {
  return group('Get Server KEM Key', () => {
    const res = http.get(`${baseUrl}/debug/kem-pub`);

    const success = check(res, {
      'kem key status is 200': (r) => r.status === 200,
      'kem key response has key': (r) => {
        try {
          return JSON.parse(r.body).kem_public_key !== undefined;
        } catch {
          return false;
        }
      },
    });

    customMetrics.successRate.add(success);

    if (success) {
      return JSON.parse(res.body).kem_public_key;
    }
    return null;
  });
}

// Get server DID
export function getServerDID(baseUrl) {
  return group('Get Server DID', () => {
    const res = http.get(`${baseUrl}/debug/server-did`);

    const success = check(res, {
      'server did status is 200': (r) => r.status === 200,
      'server did response has did': (r) => {
        try {
          return JSON.parse(r.body).did !== undefined;
        } catch {
          return false;
        }
      },
    });

    customMetrics.successRate.add(success);

    if (success) {
      return JSON.parse(res.body).did;
    }
    return null;
  });
}

// Register test agent
export function registerAgent(baseUrl, agentData) {
  return group('Register Agent', () => {
    const payload = JSON.stringify({
      did: agentData.did || generateRandomDID(),
      name: agentData.name || 'Load Test Agent',
      is_active: agentData.isActive !== undefined ? agentData.isActive : true,
      public_key: agentData.publicKey || generateRandomKey(),
      public_kem_key: agentData.publicKemKey || generateRandomKey(),
    });

    const params = {
      headers: { 'Content-Type': 'application/json' },
      tags: { operation: 'register_agent' },
    };

    const res = http.post(`${baseUrl}/debug/register-agent`, payload, params);

    const success = check(res, {
      'register status is 200': (r) => r.status === 200,
      'register response has message': (r) => {
        try {
          return JSON.parse(r.body).message !== undefined;
        } catch {
          return false;
        }
      },
    });

    customMetrics.successRate.add(success);
    return success;
  });
}

// Simulate handshake (mock HPKE for load testing)
export function performHandshake(baseUrl, clientDID, serverDID) {
  return group('Handshake', () => {
    const startTime = Date.now();

    // Generate mock encrypted message
    const mockMessage = encoding.b64encode(
      crypto.randomBytes(100) // Simulate HPKE encrypted payload
    );

    const timestamp = generateTimestamp();
    const signatureData = `${clientDID}|${serverDID}|${mockMessage}|${timestamp}`;
    const signature = generateMockSignature(signatureData);

    const payload = JSON.stringify({
      sender_did: clientDID,
      receiver_did: serverDID,
      message: mockMessage,
      timestamp: timestamp,
      signature: signature,
    });

    const params = {
      headers: { 'Content-Type': 'application/json' },
      tags: { operation: 'handshake' },
    };

    const res = http.post(`${baseUrl}/v1/a2a:sendMessage`, payload, params);

    const duration = Date.now() - startTime;
    customMetrics.handshakeDuration.add(duration);

    const success = check(res, {
      'handshake status is 200 or 400': (r) => [200, 400].includes(r.status),
    });

    customMetrics.successRate.add(success);

    // Extract session ID if successful
    if (res.status === 200) {
      try {
        const body = JSON.parse(res.body);
        if (body.session_id) {
          customMetrics.sessionCreated.add(1);
          return body.session_id;
        }
      } catch {
        // Parse error
      }
    }

    return null;
  });
}

// Send message in existing session
export function sendMessage(baseUrl, clientDID, serverDID, sessionID) {
  return group('Send Message', () => {
    const startTime = Date.now();

    const mockMessage = encoding.b64encode(
      crypto.randomBytes(150) // Simulate encrypted message
    );

    const timestamp = generateTimestamp();
    const signatureData = `${clientDID}|${serverDID}|${mockMessage}|${timestamp}`;
    const signature = generateMockSignature(signatureData);

    const payload = JSON.stringify({
      sender_did: clientDID,
      receiver_did: serverDID,
      message: mockMessage,
      timestamp: timestamp,
      signature: signature,
    });

    const params = {
      headers: {
        'Content-Type': 'application/json',
        'X-Session-ID': sessionID,
      },
      tags: { operation: 'send_message' },
    };

    const res = http.post(`${baseUrl}/v1/a2a:sendMessage`, payload, params);

    const duration = Date.now() - startTime;
    customMetrics.messageDuration.add(duration);

    const success = check(res, {
      'message status is 200 or 400': (r) => [200, 400].includes(r.status),
    });

    customMetrics.successRate.add(success);
    return success;
  });
}

// Full flow: register + handshake + messages
export function fullSessionFlow(baseUrl, messageCount = 5) {
  const clientDID = generateRandomDID();
  const serverDID = getServerDID(baseUrl);

  if (!serverDID) {
    console.error('Failed to get server DID');
    return;
  }

  // Register client agent
  const registered = registerAgent(baseUrl, {
    did: clientDID,
    name: `Load Test Client ${clientDID.slice(-8)}`,
  });

  if (!registered) {
    console.error('Failed to register agent');
    return;
  }

  sleep(0.1);

  // Perform handshake
  const sessionID = performHandshake(baseUrl, clientDID, serverDID);

  if (!sessionID) {
    // Handshake might fail in mock mode, that's expected
    return;
  }

  sleep(0.1);

  // Send multiple messages
  let successfulMessages = 0;
  for (let i = 0; i < messageCount; i++) {
    const success = sendMessage(baseUrl, clientDID, serverDID, sessionID);
    if (success) {
      successfulMessages++;
    }
    sleep(0.2);
  }

  customMetrics.messagesPerSession.add(successfulMessages);
}

// Stress test helper: rapid-fire requests
export function rapidFireRequests(baseUrl, count) {
  const clientDID = generateRandomDID();
  const serverDID = getServerDID(baseUrl);

  if (!serverDID) return;

  for (let i = 0; i < count; i++) {
    performHandshake(baseUrl, clientDID, serverDID);
  }
}

// Wait for system to stabilize
export function waitForStabilization(baseUrl, maxAttempts = 10) {
  for (let i = 0; i < maxAttempts; i++) {
    const res = healthCheck(baseUrl);
    if (res && res.status === 200) {
      return true;
    }
    sleep(1);
  }
  return false;
}

// Summary helper
export function printSummary(data) {
  console.log('\n=== Load Test Summary ===');
  console.log(`Handshake avg duration: ${data.metrics.sage_handshake_duration.values.avg.toFixed(2)}ms`);
  console.log(`Message avg duration: ${data.metrics.sage_message_duration.values.avg.toFixed(2)}ms`);
  console.log(`Success rate: ${(data.metrics.sage_success_rate.values.rate * 100).toFixed(2)}%`);
  console.log(`Sessions created: ${data.metrics.sage_sessions_created.values.count}`);
  console.log('========================\n');
}
