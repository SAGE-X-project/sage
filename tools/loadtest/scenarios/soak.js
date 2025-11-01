// SAGE Soak Test (Endurance Test)
// This test runs for an extended period to detect memory leaks, resource exhaustion, and degradation

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

const cfg = getConfig('soak');

// Soak test configuration: runs for hours
const SOAK_DURATION = __ENV.SOAK_DURATION || '2h';  // Default 2 hours, can set to 24h

export const options = {
  stages: [
    { duration: '5m', target: cfg.vus.soak },    // Ramp up to 50 users
    { duration: SOAK_DURATION, target: cfg.vus.soak }, // Hold for extended period
    { duration: '5m', target: 0 },               // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
    // Key soak test metric: performance should NOT degrade over time
    'sage_handshake_duration{scenario:soak}': ['p(95)<500'],
  },
  tags: {
    test_type: 'soak',
    environment: __ENV.SAGE_ENV || 'local',
  },
};

export function setup() {
  console.log('=== SAGE Soak Test (Endurance) ===');
  console.log(`Base URL: ${cfg.baseUrl}`);
  console.log(`VUs: ${cfg.vus.soak}`);
  console.log(`Duration: ${SOAK_DURATION}`);
  console.log('===================================\n');

  const healthRes = healthCheck(cfg.baseUrl);
  if (!healthRes || healthRes.status !== 200) {
    throw new Error('Server health check failed');
  }

  const serverDID = getServerDID(cfg.baseUrl);
  if (!serverDID) {
    throw new Error('Failed to retrieve server DID');
  }

  console.log(`Server DID: ${serverDID}\n`);
  console.log(' Starting long-running soak test...');
  console.log('   Monitor for:');
  console.log('   - Memory leaks (increasing memory usage)');
  console.log('   - Performance degradation (increasing latency)');
  console.log('   - Connection pool exhaustion');
  console.log('   - Database bloat\n');

  return {
    baseUrl: cfg.baseUrl,
    serverDID: serverDID,
    startTime: Date.now(),
  };
}

export default function (data) {
  const clientDID = generateRandomDID();
  const iteration = __ITER;

  // Periodic health check (every 50 iterations)
  if (iteration % 50 === 0) {
    const healthRes = healthCheck(data.baseUrl);
    if (!healthRes || healthRes.status !== 200) {
      console.warn(`  Health check failed at iteration ${iteration}`);
    }
  }

  // Regular soak test flow
  const registered = registerAgent(data.baseUrl, {
    did: clientDID,
    name: `Soak Test Client ${clientDID.slice(-8)}`,
  });

  if (!registered) {
    sleep(2);
    return;
  }

  sleep(1);

  const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

  if (!sessionID) {
    sleep(2);
    return;
  }

  sleep(1);

  // Send moderate number of messages (3-7)
  const messageCount = 3 + Math.floor(Math.random() * 5);
  for (let i = 0; i < messageCount; i++) {
    sendMessage(data.baseUrl, clientDID, data.serverDID, sessionID);
    sleep(0.5);
  }

  // Longer sleep for soak test (more realistic usage pattern)
  sleep(cfg.sleepDuration);
}

export function teardown(data) {
  const durationMs = Date.now() - data.startTime;
  const durationHours = (durationMs / 1000 / 60 / 60).toFixed(2);

  console.log('\n=== Soak Test Complete ===');
  console.log(`Total Duration: ${durationHours} hours`);

  // Final health check
  const healthRes = healthCheck(data.baseUrl);
  if (healthRes && healthRes.status === 200) {
    console.log(' Server is healthy after soak test');
  } else {
    console.log(' Server health degraded after soak test');
  }

  console.log('\n  Post-Test Checks:');
  console.log('   1. Check server memory usage (should be stable)');
  console.log('   2. Check database size (should not grow excessively)');
  console.log('   3. Check connection pool metrics');
  console.log('   4. Review logs for errors or warnings');
  console.log('   5. Check Grafana dashboards for trends');
  console.log('==========================\n');
}

export function handleSummary(data) {
  const metrics = data.metrics;

  console.log('\n=== Soak Test Analysis ===');

  // Analyze performance stability over time
  const avgDuration = metrics.http_req_duration.values.avg;
  const p95Duration = metrics.http_req_duration.values['p(95)'];
  const errorRate = metrics.http_req_failed ? metrics.http_req_failed.values.rate : 0;

  console.log(`Average Response Time: ${avgDuration.toFixed(2)}ms`);
  console.log(`95th Percentile: ${p95Duration.toFixed(2)}ms`);
  console.log(`Error Rate: ${(errorRate * 100).toFixed(2)}%`);

  // Check for degradation indicators
  const passed = errorRate < 0.01 && p95Duration < 500;

  console.log(`\nSoak Test Result: ${passed ? ' PASSED' : ' FAILED'}`);

  if (!passed) {
    console.log('\n  Potential Issues Detected:');
    if (errorRate >= 0.01) {
      console.log(`   - Error rate too high: ${(errorRate * 100).toFixed(2)}%`);
      console.log('     → Check logs for error patterns');
      console.log('     → Look for connection pool exhaustion');
    }
    if (p95Duration >= 500) {
      console.log(`   - Response times degraded: ${p95Duration.toFixed(2)}ms (p95)`);
      console.log('     → Check for memory leaks');
      console.log('     → Review database query performance');
      console.log('     → Check for session/nonce table bloat');
    }
  }

  console.log('\n Next Steps:');
  console.log('   - Compare metrics with baseline test');
  console.log('   - Review Prometheus metrics for trends');
  console.log('   - Check database for cleanup job effectiveness');
  console.log('   - Verify no resource leaks in application');
  console.log('==========================\n');

  return {
    'stdout': JSON.stringify(data, null, 2),
    'loadtest/reports/soak-summary.json': JSON.stringify(data, null, 2),
  };
}
