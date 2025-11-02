// SAGE Concurrent Sessions Load Test
// This test validates the system's ability to handle multiple concurrent sessions

import { sleep } from 'k6';
import { getConfig } from '../config.js';
import {
  healthCheck,
  getServerDID,
  performHandshake,
  sendMessage,
  generateRandomDID,
  registerAgent,
  customMetrics,
} from '../utils/helpers.js';

const cfg = getConfig('concurrent-sessions');

export const options = {
  stages: [
    { duration: '1m', target: 30 },      // Ramp up to 30 users
    { duration: '3m', target: 30 },      // Hold at 30 users
    { duration: '2m', target: 60 },      // Increase to 60 users
    { duration: '3m', target: 60 },      // Hold at 60 users
    { duration: '1m', target: 0 },       // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000', 'p(99)<2000'],
    http_req_failed: ['rate<0.02'],                    // < 2% errors
    sage_sessions_created: ['count>100'],              // At least 100 sessions
    sage_handshake_duration: ['p(95)<800'],            // Handshake performance
  },
  tags: {
    test_type: 'concurrent-sessions',
    environment: __ENV.SAGE_ENV || 'local',
  },
};

// Setup: runs once at the beginning
export function setup() {
  console.log('=== SAGE Concurrent Sessions Load Test ===');
  console.log(`Base URL: ${cfg.baseUrl}`);
  console.log(`Target: 60 concurrent users maintaining multiple sessions`);
  console.log('==========================================\n');

  // Verify server is healthy
  const healthRes = healthCheck(cfg.baseUrl);
  if (!healthRes || healthRes.status !== 200) {
    throw new Error('Server health check failed. Cannot proceed with test.');
  }

  // Get server DID for test
  const serverDID = getServerDID(cfg.baseUrl);
  if (!serverDID) {
    throw new Error('Failed to retrieve server DID');
  }

  console.log(`Server DID: ${serverDID}`);
  console.log('Focus: Testing concurrent session management\n');

  return {
    baseUrl: cfg.baseUrl,
    serverDID: serverDID,
  };
}

// Main test function: Each VU maintains 3-5 concurrent sessions
export default function (data) {
  const sessionsPerUser = 3 + Math.floor(Math.random() * 3); // 3-5 sessions
  const sessions = [];

  // Phase 1: Create multiple concurrent sessions
  for (let i = 0; i < sessionsPerUser; i++) {
    const clientDID = generateRandomDID();

    // Register agent
    const registered = registerAgent(data.baseUrl, {
      did: clientDID,
      name: `Concurrent Test Client ${clientDID.slice(-8)}`,
    });

    if (!registered) {
      continue;
    }

    sleep(0.2);

    // Perform handshake to create session
    const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

    if (sessionID) {
      sessions.push({
        clientDID: clientDID,
        sessionID: sessionID,
        messageCount: 0,
      });
    }

    sleep(0.3);
  }

  if (sessions.length === 0) {
    sleep(cfg.sleepDuration);
    return;
  }

  // Phase 2: Send messages across all sessions concurrently
  const rounds = 5 + Math.floor(Math.random() * 5); // 5-10 rounds
  for (let round = 0; round < rounds; round++) {
    // Randomly select a session
    const session = sessions[Math.floor(Math.random() * sessions.length)];

    // Send message
    const success = sendMessage(
      data.baseUrl,
      session.clientDID,
      data.serverDID,
      session.sessionID
    );

    if (success) {
      session.messageCount++;
    }

    // Short sleep between messages
    sleep(0.2);
  }

  // Track messages per session
  sessions.forEach((session) => {
    customMetrics.messagesPerSession.add(session.messageCount);
  });

  // Wait before next iteration
  sleep(cfg.sleepDuration);
}

// Teardown: runs once at the end
export function teardown(data) {
  console.log('\n=== Concurrent Sessions Test Complete ===');
  console.log('Check metrics for concurrent session performance.');
  console.log('=========================================\n');
}

// Handle thresholds
export function handleSummary(data) {
  const passed = Object.keys(data.metrics).every((metric) => {
    const m = data.metrics[metric];
    if (m.thresholds) {
      return Object.values(m.thresholds).every((t) => t.ok);
    }
    return true;
  });

  console.log(`\nConcurrent Sessions Test Result: ${passed ? ' PASSED' : ' FAILED'}\n`);

  // Print session statistics
  if (data.metrics.sage_sessions_created) {
    console.log(`Total sessions created: ${data.metrics.sage_sessions_created.values.count}`);
  }
  if (data.metrics.sage_handshake_duration) {
    console.log(
      `Avg handshake duration: ${data.metrics.sage_handshake_duration.values.avg.toFixed(2)}ms`
    );
  }
  if (data.metrics.sage_messages_per_session) {
    console.log(
      `Avg messages per session: ${data.metrics.sage_messages_per_session.values.avg.toFixed(2)}`
    );
  }

  return {
    stdout: JSON.stringify(data, null, 2),
    'loadtest/reports/concurrent-sessions-summary.json': JSON.stringify(data, null, 2),
  };
}
