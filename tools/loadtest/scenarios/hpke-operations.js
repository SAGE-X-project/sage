// SAGE HPKE Operations Load Test
// This test focuses on HPKE encryption/decryption operations (handshakes)

import { sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { getConfig } from '../config.js';
import {
  healthCheck,
  getServerDID,
  getServerKemKey,
  performHandshake,
  sendMessage,
  generateRandomDID,
  registerAgent,
  customMetrics,
} from '../utils/helpers.js';

const cfg = getConfig('hpke-operations');

// Custom metrics for HPKE operations
const hpkeHandshakeDuration = new Trend('sage_hpke_handshake_duration');
const hpkeEncryptionRate = new Rate('sage_hpke_encryption_success_rate');
const hpkeOperationsTotal = new Counter('sage_hpke_operations_total');

export const options = {
  stages: [
    { duration: '30s', target: 15 },     // Ramp up to 15 users
    { duration: '2m', target: 15 },      // Hold at 15 users
    { duration: '1m', target: 40 },      // Increase to 40 users
    { duration: '3m', target: 40 },      // Hold at 40 users (crypto-intensive)
    { duration: '1m', target: 80 },      // Peak at 80 users
    { duration: '2m', target: 80 },      // Hold peak
    { duration: '30s', target: 0 },      // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1500', 'p(99)<3000'],   // HPKE is CPU-intensive
    http_req_failed: ['rate<0.03'],                     // < 3% errors
    sage_hpke_handshake_duration: ['p(95)<1200'],      // Handshake with HPKE
    sage_hpke_encryption_success_rate: ['rate>0.90'],  // > 90% success
    sage_hpke_operations_total: ['count>150'],         // At least 150 HPKE ops
  },
  tags: {
    test_type: 'hpke-operations',
    environment: __ENV.SAGE_ENV || 'local',
  },
};

// Setup: runs once at the beginning
export function setup() {
  console.log('=== SAGE HPKE Operations Load Test ===');
  console.log(`Base URL: ${cfg.baseUrl}`);
  console.log(`Target: 80 users performing HPKE operations`);
  console.log('======================================\n');

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
  if (!serverKemKey) {
    console.warn('Server KEM key not available. Test will use mock encryption.');
  }

  console.log(`Server DID: ${serverDID}`);
  console.log(`Server KEM Key: ${serverKemKey ? 'Available' : 'Not available (mock mode)'}`);
  console.log('Focus: HPKE encryption/decryption performance\n');

  return {
    baseUrl: cfg.baseUrl,
    serverDID: serverDID,
    serverKemKey: serverKemKey,
  };
}

// Main test function: Heavy HPKE operations
export default function (data) {
  const clientDID = generateRandomDID();

  // Step 1: Register agent (required for handshake)
  const registered = registerAgent(data.baseUrl, {
    did: clientDID,
    name: `HPKE Test Client ${clientDID.slice(-8)}`,
  });

  if (!registered) {
    sleep(cfg.sleepDuration);
    return;
  }

  sleep(0.2);

  // Step 2: Perform HPKE handshake (80% of operations)
  if (Math.random() < 0.8) {
    const startTime = Date.now();

    const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

    if (sessionID) {
      hpkeHandshakeDuration.add(Date.now() - startTime);
      hpkeEncryptionRate.add(1);
      hpkeOperationsTotal.add(1);

      // Follow up with encrypted messages (3-5 messages)
      const messageCount = 3 + Math.floor(Math.random() * 3);
      for (let i = 0; i < messageCount; i++) {
        sendMessage(data.baseUrl, clientDID, data.serverDID, sessionID);
        hpkeOperationsTotal.add(1);
        sleep(0.2);
      }
    } else {
      hpkeEncryptionRate.add(0);
    }
  }

  // Step 3: Burst handshakes (20% of iterations - stress crypto)
  if (Math.random() < 0.2) {
    const burstCount = 3 + Math.floor(Math.random() * 3); // 3-5 rapid handshakes

    for (let i = 0; i < burstCount; i++) {
      const burstClientDID = generateRandomDID();

      // Quick register
      registerAgent(data.baseUrl, {
        did: burstClientDID,
        name: `Burst Client ${i + 1}`,
      });

      sleep(0.1);

      // Rapid handshake
      const startTime = Date.now();
      const sessionID = performHandshake(data.baseUrl, burstClientDID, data.serverDID);

      if (sessionID) {
        hpkeHandshakeDuration.add(Date.now() - startTime);
        hpkeEncryptionRate.add(1);
        hpkeOperationsTotal.add(1);
      } else {
        hpkeEncryptionRate.add(0);
      }

      sleep(0.1);
    }
  }

  // Wait before next iteration
  sleep(cfg.sleepDuration);
}

// Teardown: runs once at the end
export function teardown(data) {
  console.log('\n=== HPKE Operations Test Complete ===');
  console.log('Check metrics for HPKE encryption performance.');
  console.log('====================================\n');
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

  console.log(`\nHPKE Operations Test Result: ${passed ? '✅ PASSED' : '❌ FAILED'}\n`);

  // Print HPKE operation statistics
  if (data.metrics.sage_hpke_operations_total) {
    console.log(`Total HPKE operations: ${data.metrics.sage_hpke_operations_total.values.count}`);
  }
  if (data.metrics.sage_hpke_handshake_duration) {
    console.log(
      `Avg HPKE handshake duration: ${data.metrics.sage_hpke_handshake_duration.values.avg.toFixed(
        2
      )}ms`
    );
    console.log(
      `P95 HPKE handshake duration: ${data.metrics.sage_hpke_handshake_duration.values.p95.toFixed(
        2
      )}ms`
    );
    console.log(
      `P99 HPKE handshake duration: ${data.metrics.sage_hpke_handshake_duration.values.p99.toFixed(
        2
      )}ms`
    );
  }
  if (data.metrics.sage_hpke_encryption_success_rate) {
    console.log(
      `HPKE encryption success rate: ${(
        data.metrics.sage_hpke_encryption_success_rate.values.rate * 100
      ).toFixed(2)}%`
    );
  }

  return {
    stdout: JSON.stringify(data, null, 2),
    'loadtest/reports/hpke-operations-summary.json': JSON.stringify(data, null, 2),
  };
}
