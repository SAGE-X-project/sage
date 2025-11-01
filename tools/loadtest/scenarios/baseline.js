// SAGE Baseline Load Test
// This test establishes performance baselines under normal load conditions

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

const cfg = getConfig('baseline');

export const options = {
  stages: [
    { duration: cfg.durations.rampUp, target: cfg.vus.baseline },     // Ramp up to 10 users
    { duration: cfg.durations.steady, target: cfg.vus.baseline },     // Stay at 10 users
    { duration: cfg.durations.rampDown, target: 0 },                   // Ramp down
  ],
  thresholds: cfg.thresholds,
  tags: {
    test_type: 'baseline',
    environment: __ENV.SAGE_ENV || 'local',
  },
};

// Setup: runs once at the beginning
export function setup() {
  console.log('=== SAGE Baseline Load Test ===');
  console.log(`Base URL: ${cfg.baseUrl}`);
  console.log(`VUs: ${cfg.vus.baseline}`);
  console.log('================================\n');

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

  console.log(`Server DID: ${serverDID}\n`);

  return {
    baseUrl: cfg.baseUrl,
    serverDID: serverDID,
  };
}

// Main test function: runs for each VU iteration
export default function (data) {
  const clientDID = generateRandomDID();

  // Step 1: Health check (10% of iterations)
  if (Math.random() < 0.1) {
    healthCheck(data.baseUrl);
    sleep(cfg.sleepDuration);
    return;
  }

  // Step 2: Register agent
  const registered = registerAgent(data.baseUrl, {
    did: clientDID,
    name: `Baseline Test Client ${clientDID.slice(-8)}`,
  });

  if (!registered) {
    sleep(cfg.sleepDuration);
    return;
  }

  sleep(0.5);

  // Step 3: Perform handshake
  const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

  if (!sessionID) {
    // Handshake might fail with mock data, that's expected
    sleep(cfg.sleepDuration);
    return;
  }

  sleep(0.5);

  // Step 4: Send messages (3-5 messages per session)
  const messageCount = 3 + Math.floor(Math.random() * 3);
  for (let i = 0; i < messageCount; i++) {
    sendMessage(data.baseUrl, clientDID, data.serverDID, sessionID);
    sleep(0.3);
  }

  // Wait before next iteration
  sleep(cfg.sleepDuration);
}

// Teardown: runs once at the end
export function teardown(data) {
  console.log('\n=== Baseline Test Complete ===');
  console.log('Check the summary above for performance metrics.');
  console.log('==============================\n');
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

  console.log(`\nBaseline Test Result: ${passed ? ' PASSED' : ' FAILED'}\n`);

  return {
    'stdout': JSON.stringify(data, null, 2),
    'loadtest/reports/baseline-summary.json': JSON.stringify(data, null, 2),
  };
}
