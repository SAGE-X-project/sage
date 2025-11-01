// SAGE Mixed Workload Load Test
// This test simulates realistic production traffic with mixed operations

import { sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { getConfig } from '../config.js';
import {
  healthCheck,
  getServerDID,
  getServerKemKey,
  performHandshake,
  sendMessage,
  generateRandomDID,
  registerAgent,
  fullSessionFlow,
  customMetrics,
} from '../utils/helpers.js';

const cfg = getConfig('mixed-workload');

// Custom metrics for mixed workload
const workloadDistribution = new Counter('sage_workload_distribution');
const operationDuration = new Trend('sage_operation_duration');

export const options = {
  stages: [
    { duration: '1m', target: 25 },      // Ramp up to 25 users
    { duration: '3m', target: 25 },      // Hold at 25 users
    { duration: '1m', target: 50 },      // Increase to 50 users
    { duration: '5m', target: 50 },      // Hold at 50 users (main test)
    { duration: '1m', target: 75 },      // Peak at 75 users
    { duration: '3m', target: 75 },      // Hold peak
    { duration: '1m', target: 0 },       // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000', 'p(99)<2000'],
    http_req_failed: ['rate<0.02'],                    // < 2% errors
    sage_success_rate: ['rate>0.92'],                  // > 92% success overall
    sage_sessions_created: ['count>50'],               // At least 50 sessions
    sage_operation_duration: ['p(95)<900'],            // Mixed operations
  },
  tags: {
    test_type: 'mixed-workload',
    environment: __ENV.SAGE_ENV || 'local',
  },
};

// Setup: runs once at the beginning
export function setup() {
  console.log('=== SAGE Mixed Workload Load Test ===');
  console.log(`Base URL: ${cfg.baseUrl}`);
  console.log(`Target: 75 users with realistic mixed operations`);
  console.log('=====================================\n');

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

  // Get server KEM public key
  const serverKemKey = getServerKemKey(cfg.baseUrl);

  console.log(`Server DID: ${serverDID}`);
  console.log(`Server KEM Key: ${serverKemKey ? 'Available' : 'Not available'}`);
  console.log('Focus: Realistic mixed production workload\n');
  console.log('Workload distribution:');
  console.log('  - 40% Full session flows (register + handshake + messages)');
  console.log('  - 25% DID operations (register only)');
  console.log('  - 20% Message sending (existing sessions)');
  console.log('  - 10% Health checks');
  console.log('  - 5% Burst operations\n');

  return {
    baseUrl: cfg.baseUrl,
    serverDID: serverDID,
    serverKemKey: serverKemKey,
  };
}

// Main test function: Mixed realistic operations
export default function (data) {
  const rand = Math.random();
  const startTime = Date.now();

  // Workload 1: Full session flow (40% - most common)
  if (rand < 0.40) {
    workloadDistribution.add(1, { workload: 'full_session_flow' });

    const clientDID = generateRandomDID();

    // Register
    const registered = registerAgent(data.baseUrl, {
      did: clientDID,
      name: `Mixed Workload Client ${clientDID.slice(-8)}`,
    });

    if (registered) {
      sleep(0.3);

      // Handshake
      const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

      if (sessionID) {
        sleep(0.2);

        // Send messages (2-4 messages)
        const messageCount = 2 + Math.floor(Math.random() * 3);
        for (let i = 0; i < messageCount; i++) {
          sendMessage(data.baseUrl, clientDID, data.serverDID, sessionID);
          sleep(0.3);
        }
      }
    }

    operationDuration.add(Date.now() - startTime);
  }
  // Workload 2: DID registration only (25%)
  else if (rand < 0.65) {
    workloadDistribution.add(1, { workload: 'did_operations' });

    const clientDID = generateRandomDID();

    registerAgent(data.baseUrl, {
      did: clientDID,
      name: `DID Registration ${clientDID.slice(-8)}`,
      isActive: true,
    });

    operationDuration.add(Date.now() - startTime);
    sleep(0.5);
  }
  // Workload 3: Message sending (20% - simulate active sessions)
  else if (rand < 0.85) {
    workloadDistribution.add(1, { workload: 'message_sending' });

    const clientDID = generateRandomDID();

    // Register and get session
    const registered = registerAgent(data.baseUrl, {
      did: clientDID,
      name: `Active Session Client ${clientDID.slice(-8)}`,
    });

    if (registered) {
      sleep(0.2);

      const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

      if (sessionID) {
        // Rapid message sending (simulating active conversation)
        const burstMessages = 5 + Math.floor(Math.random() * 5); // 5-10 messages
        for (let i = 0; i < burstMessages; i++) {
          sendMessage(data.baseUrl, clientDID, data.serverDID, sessionID);
          sleep(0.15); // Faster messaging
        }
      }
    }

    operationDuration.add(Date.now() - startTime);
  }
  // Workload 4: Health checks (10%)
  else if (rand < 0.95) {
    workloadDistribution.add(1, { workload: 'health_checks' });

    healthCheck(data.baseUrl);
    sleep(0.2);

    // Sometimes check KEM key too
    if (Math.random() < 0.5) {
      getServerKemKey(data.baseUrl);
    }

    operationDuration.add(Date.now() - startTime);
    sleep(0.5);
  }
  // Workload 5: Burst operations (5% - simulating traffic spikes)
  else {
    workloadDistribution.add(1, { workload: 'burst_operations' });

    // Rapid-fire multiple operations
    for (let i = 0; i < 3; i++) {
      const burstClientDID = generateRandomDID();

      registerAgent(data.baseUrl, {
        did: burstClientDID,
        name: `Burst ${i + 1}`,
      });

      sleep(0.1);

      performHandshake(data.baseUrl, burstClientDID, data.serverDID);
      sleep(0.1);
    }

    operationDuration.add(Date.now() - startTime);
  }

  // Random think time (simulating user behavior)
  const thinkTime = 0.5 + Math.random() * 2; // 0.5-2.5s
  sleep(thinkTime);
}

// Teardown: runs once at the end
export function teardown(data) {
  console.log('\n=== Mixed Workload Test Complete ===');
  console.log('Check metrics for realistic production performance.');
  console.log('===================================\n');
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

  console.log(`\nMixed Workload Test Result: ${passed ? ' PASSED' : ' FAILED'}\n`);

  // Print workload statistics
  console.log('=== Workload Distribution ===');
  if (data.metrics.sage_workload_distribution) {
    const distribution = data.metrics.sage_workload_distribution.values;
    console.log(`Full session flows: ${distribution.full_session_flow || 0}`);
    console.log(`DID operations: ${distribution.did_operations || 0}`);
    console.log(`Message sending: ${distribution.message_sending || 0}`);
    console.log(`Health checks: ${distribution.health_checks || 0}`);
    console.log(`Burst operations: ${distribution.burst_operations || 0}`);
  }

  console.log('\n=== Performance ===');
  if (data.metrics.sage_operation_duration) {
    console.log(
      `Avg operation duration: ${data.metrics.sage_operation_duration.values.avg.toFixed(2)}ms`
    );
    console.log(
      `P95 operation duration: ${data.metrics.sage_operation_duration.values.p95.toFixed(2)}ms`
    );
  }
  if (data.metrics.sage_sessions_created) {
    console.log(`Sessions created: ${data.metrics.sage_sessions_created.values.count}`);
  }
  if (data.metrics.sage_success_rate) {
    console.log(
      `Overall success rate: ${(data.metrics.sage_success_rate.values.rate * 100).toFixed(2)}%`
    );
  }

  return {
    stdout: JSON.stringify(data, null, 2),
    'loadtest/reports/mixed-workload-summary.json': JSON.stringify(data, null, 2),
  };
}
