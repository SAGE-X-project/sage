// SAGE Stress Test
// This test pushes the system beyond normal load to identify breaking points

import { sleep } from 'k6';
import { getConfig } from '../config.js';
import {
  healthCheck,
  getServerDID,
  performHandshake,
  sendMessage,
  generateRandomDID,
  registerAgent,
  rapidFireRequests,
  customMetrics,
} from '../utils/helpers.js';

const cfg = getConfig('stress');

export const options = {
  stages: [
    { duration: '2m', target: 50 },           // Ramp up to 50 users
    { duration: '3m', target: cfg.vus.stress }, // Ramp up to 100 users
    { duration: '5m', target: cfg.vus.stress }, // Stay at 100 users
    { duration: '2m', target: 200 },          // Spike to 200 users
    { duration: '3m', target: 200 },          // Hold spike
    { duration: '2m', target: cfg.vus.stress }, // Drop back to 100
    { duration: '2m', target: 0 },            // Ramp down
  ],
  thresholds: cfg.thresholds,
  tags: {
    test_type: 'stress',
    environment: __ENV.SAGE_ENV || 'local',
  },
};

export function setup() {
  console.log('=== SAGE Stress Test ===');
  console.log(`Base URL: ${cfg.baseUrl}`);
  console.log(`Peak VUs: 200`);
  console.log(`Sustained VUs: ${cfg.vus.stress}`);
  console.log('========================\n');

  const healthRes = healthCheck(cfg.baseUrl);
  if (!healthRes || healthRes.status !== 200) {
    throw new Error('Server health check failed');
  }

  const serverDID = getServerDID(cfg.baseUrl);
  if (!serverDID) {
    throw new Error('Failed to retrieve server DID');
  }

  console.log(`Server DID: ${serverDID}\n`);
  console.log('Starting stress test...\n');

  return {
    baseUrl: cfg.baseUrl,
    serverDID: serverDID,
  };
}

export default function (data) {
  const clientDID = generateRandomDID();
  const iteration = __ITER;

  // During spike phase (every 10th iteration), do rapid-fire requests
  if (iteration % 10 === 0 && __VU > 150) {
    rapidFireRequests(data.baseUrl, 5);
    sleep(0.1);
    return;
  }

  // Regular stress test flow
  const registered = registerAgent(data.baseUrl, {
    did: clientDID,
    name: `Stress Test Client ${clientDID.slice(-8)}`,
  });

  if (!registered) {
    sleep(0.5);
    return;
  }

  sleep(0.2); // Shorter sleep for stress test

  const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

  if (!sessionID) {
    sleep(0.5);
    return;
  }

  sleep(0.2);

  // Send more messages under stress (5-10 messages)
  const messageCount = 5 + Math.floor(Math.random() * 6);
  for (let i = 0; i < messageCount; i++) {
    sendMessage(data.baseUrl, clientDID, data.serverDID, sessionID);
    sleep(0.1); // Minimal sleep between messages
  }

  sleep(0.5);
}

export function teardown(data) {
  console.log('\n=== Stress Test Complete ===');

  // Check if server is still healthy
  const healthRes = healthCheck(data.baseUrl);
  if (healthRes && healthRes.status === 200) {
    console.log(' Server survived stress test and is healthy');
  } else {
    console.log('  Server may be degraded after stress test');
  }

  console.log('============================\n');
}

export function handleSummary(data) {
  const metrics = data.metrics;

  // Check for performance degradation
  const avgDuration = metrics.http_req_duration.values.avg;
  const p95Duration = metrics.http_req_duration.values['p(95)'];
  const errorRate = metrics.http_req_failed ? metrics.http_req_failed.values.rate : 0;

  console.log('\n=== Stress Test Results ===');
  console.log(`Average Response Time: ${avgDuration.toFixed(2)}ms`);
  console.log(`95th Percentile: ${p95Duration.toFixed(2)}ms`);
  console.log(`Error Rate: ${(errorRate * 100).toFixed(2)}%`);

  const passed = errorRate < 0.05 && p95Duration < 1000;
  console.log(`\nStress Test Result: ${passed ? ' PASSED' : ' FAILED'}\n`);

  if (!passed) {
    console.log('  System showed signs of stress:');
    if (errorRate >= 0.05) {
      console.log(`   - High error rate: ${(errorRate * 100).toFixed(2)}%`);
    }
    if (p95Duration >= 1000) {
      console.log(`   - Slow response times: ${p95Duration.toFixed(2)}ms (p95)`);
    }
  }

  console.log('===========================\n');

  return {
    'stdout': JSON.stringify(data, null, 2),
    'loadtest/reports/stress-summary.json': JSON.stringify(data, null, 2),
  };
}
