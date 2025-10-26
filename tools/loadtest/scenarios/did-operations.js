// SAGE DID Operations Load Test
// This test focuses on DID registration and resolution operations

import { sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { getConfig } from '../config.js';
import {
  healthCheck,
  getServerDID,
  generateRandomDID,
  registerAgent,
  customMetrics,
} from '../utils/helpers.js';

const cfg = getConfig('did-operations');

// Custom metrics for DID operations
const didRegisterDuration = new Trend('sage_did_register_duration');
const didResolveDuration = new Trend('sage_did_resolve_duration');
const didOperationsTotal = new Counter('sage_did_operations_total');

export const options = {
  stages: [
    { duration: '30s', target: 20 },     // Ramp up to 20 users
    { duration: '2m', target: 20 },      // Hold at 20 users
    { duration: '1m', target: 50 },      // Increase to 50 users
    { duration: '2m', target: 50 },      // Hold at 50 users
    { duration: '1m', target: 100 },     // Peak at 100 users
    { duration: '2m', target: 100 },     // Hold peak
    { duration: '30s', target: 0 },      // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<800', 'p(99)<1500'],
    http_req_failed: ['rate<0.02'],                    // < 2% errors
    sage_did_register_duration: ['p(95)<600'],         // DID registration performance
    sage_did_operations_total: ['count>200'],          // At least 200 DID operations
  },
  tags: {
    test_type: 'did-operations',
    environment: __ENV.SAGE_ENV || 'local',
  },
};

// Setup: runs once at the beginning
export function setup() {
  console.log('=== SAGE DID Operations Load Test ===');
  console.log(`Base URL: ${cfg.baseUrl}`);
  console.log(`Target: 100 users performing DID operations`);
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

  console.log(`Server DID: ${serverDID}`);
  console.log('Focus: DID registration and resolution performance\n');

  return {
    baseUrl: cfg.baseUrl,
    serverDID: serverDID,
  };
}

// Main test function: Heavy DID operations
export default function (data) {
  // Operation 1: Register new DID (70% of iterations)
  if (Math.random() < 0.7) {
    const startTime = Date.now();
    const clientDID = generateRandomDID();

    const registered = registerAgent(data.baseUrl, {
      did: clientDID,
      name: `DID Test Agent ${clientDID.slice(-8)}`,
      isActive: true,
    });

    if (registered) {
      didRegisterDuration.add(Date.now() - startTime);
      didOperationsTotal.add(1);
    }

    sleep(0.3);
  }

  // Operation 2: Resolve server DID (20% of iterations)
  if (Math.random() < 0.2) {
    const startTime = Date.now();
    const resolvedDID = getServerDID(data.baseUrl);

    if (resolvedDID) {
      didResolveDuration.add(Date.now() - startTime);
      didOperationsTotal.add(1);
    }

    sleep(0.2);
  }

  // Operation 3: Batch register multiple DIDs (10% of iterations)
  if (Math.random() < 0.1) {
    const batchSize = 5 + Math.floor(Math.random() * 5); // 5-10 DIDs
    let successCount = 0;

    for (let i = 0; i < batchSize; i++) {
      const startTime = Date.now();
      const clientDID = generateRandomDID();

      const registered = registerAgent(data.baseUrl, {
        did: clientDID,
        name: `Batch DID ${i + 1}`,
        isActive: true,
      });

      if (registered) {
        successCount++;
        didRegisterDuration.add(Date.now() - startTime);
      }

      sleep(0.1);
    }

    didOperationsTotal.add(successCount);
  }

  // Wait before next iteration
  sleep(cfg.sleepDuration);
}

// Teardown: runs once at the end
export function teardown(data) {
  console.log('\n=== DID Operations Test Complete ===');
  console.log('Check metrics for DID operation performance.');
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

  console.log(`\nDID Operations Test Result: ${passed ? '✅ PASSED' : '❌ FAILED'}\n`);

  // Print DID operation statistics
  if (data.metrics.sage_did_operations_total) {
    console.log(`Total DID operations: ${data.metrics.sage_did_operations_total.values.count}`);
  }
  if (data.metrics.sage_did_register_duration) {
    console.log(
      `Avg DID registration duration: ${data.metrics.sage_did_register_duration.values.avg.toFixed(
        2
      )}ms`
    );
    console.log(
      `P95 DID registration duration: ${data.metrics.sage_did_register_duration.values.p95.toFixed(
        2
      )}ms`
    );
  }
  if (data.metrics.sage_did_resolve_duration) {
    console.log(
      `Avg DID resolution duration: ${data.metrics.sage_did_resolve_duration.values.avg.toFixed(
        2
      )}ms`
    );
  }

  return {
    stdout: JSON.stringify(data, null, 2),
    'loadtest/reports/did-operations-summary.json': JSON.stringify(data, null, 2),
  };
}
