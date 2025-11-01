// SAGE Spike Test
// This test simulates sudden, dramatic increases in traffic to test system resilience

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

const cfg = getConfig('spike');

export const options = {
  stages: [
    { duration: '1m', target: 20 },             // Baseline: 20 users
    { duration: '30s', target: cfg.vus.spike }, // SPIKE: 500 users in 30 seconds
    { duration: '2m', target: cfg.vus.spike },  // Hold spike: 500 users for 2 minutes
    { duration: '30s', target: 20 },            // Drop back to baseline
    { duration: '1m', target: 20 },             // Recovery period
    { duration: '30s', target: 0 },             // Ramp down
  ],
  thresholds: cfg.thresholds,
  tags: {
    test_type: 'spike',
    environment: __ENV.SAGE_ENV || 'local',
  },
};

export function setup() {
  console.log('=== SAGE Spike Test ===');
  console.log(`Base URL: ${cfg.baseUrl}`);
  console.log(`Baseline VUs: 20`);
  console.log(`Spike VUs: ${cfg.vus.spike}`);
  console.log('=======================\n');

  const healthRes = healthCheck(cfg.baseUrl);
  if (!healthRes || healthRes.status !== 200) {
    throw new Error('Server health check failed');
  }

  const serverDID = getServerDID(cfg.baseUrl);
  if (!serverDID) {
    throw new Error('Failed to retrieve server DID');
  }

  console.log(`Server DID: ${serverDID}\n`);
  console.log(' Preparing spike test...');
  console.log('   This will simulate sudden traffic surge');
  console.log('   Watch for:');
  console.log('   - Response time spikes');
  console.log('   - Error rate increases');
  console.log('   - Connection timeouts');
  console.log('   - Queue buildup\n');

  return {
    baseUrl: cfg.baseUrl,
    serverDID: serverDID,
    spikeStart: null,
    spikeEnd: null,
  };
}

export default function (data) {
  const currentVU = __VU;
  const clientDID = generateRandomDID();

  // Track spike phase
  if (currentVU > 100 && !data.spikeStart) {
    data.spikeStart = Date.now();
    console.log(' SPIKE STARTED');
  }

  if (currentVU <= 100 && data.spikeStart && !data.spikeEnd) {
    data.spikeEnd = Date.now();
    const spikeDuration = ((data.spikeEnd - data.spikeStart) / 1000).toFixed(2);
    console.log(` SPIKE ENDED (duration: ${spikeDuration}s)`);
  }

  // During spike: aggressive behavior
  if (currentVU > 100) {
    // Minimal delays during spike
    const registered = registerAgent(data.baseUrl, {
      did: clientDID,
      name: `Spike Test Client ${clientDID.slice(-8)}`,
    });

    if (!registered) {
      sleep(0.1);
      return;
    }

    const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

    if (sessionID) {
      // Send burst of messages
      const messageCount = 5 + Math.floor(Math.random() * 5);
      for (let i = 0; i < messageCount; i++) {
        sendMessage(data.baseUrl, clientDID, data.serverDID, sessionID);
      }
    }

    sleep(0.1); // Minimal sleep during spike

  } else {
    // Baseline behavior: normal flow
    const registered = registerAgent(data.baseUrl, {
      did: clientDID,
      name: `Baseline Client ${clientDID.slice(-8)}`,
    });

    if (!registered) {
      sleep(1);
      return;
    }

    sleep(0.5);

    const sessionID = performHandshake(data.baseUrl, clientDID, data.serverDID);

    if (!sessionID) {
      sleep(1);
      return;
    }

    sleep(0.5);

    // Normal message flow
    const messageCount = 3 + Math.floor(Math.random() * 3);
    for (let i = 0; i < messageCount; i++) {
      sendMessage(data.baseUrl, clientDID, data.serverDID, sessionID);
      sleep(0.3);
    }

    sleep(1);
  }
}

export function teardown(data) {
  console.log('\n=== Spike Test Complete ===');

  // Check system recovery
  console.log('Checking system recovery...');
  sleep(5); // Wait for system to stabilize

  const healthRes = healthCheck(data.baseUrl);
  if (healthRes && healthRes.status === 200) {
    console.log(' System recovered from spike');
  } else {
    console.log('  System may need time to recover');
  }

  console.log('===========================\n');
}

export function handleSummary(data) {
  const metrics = data.metrics;

  console.log('\n=== Spike Test Analysis ===');

  const avgDuration = metrics.http_req_duration.values.avg;
  const p95Duration = metrics.http_req_duration.values['p(95)'];
  const p99Duration = metrics.http_req_duration.values['p(99)'];
  const maxDuration = metrics.http_req_duration.values.max;
  const errorRate = metrics.http_req_failed ? metrics.http_req_failed.values.rate : 0;

  console.log('\n Performance During Spike:');
  console.log(`   Average: ${avgDuration.toFixed(2)}ms`);
  console.log(`   95th Percentile: ${p95Duration.toFixed(2)}ms`);
  console.log(`   99th Percentile: ${p99Duration.toFixed(2)}ms`);
  console.log(`   Max: ${maxDuration.toFixed(2)}ms`);
  console.log(`   Error Rate: ${(errorRate * 100).toFixed(2)}%`);

  // Spike resilience criteria
  const passed = errorRate < 0.10 && p95Duration < 2000;

  console.log(`\nSpike Test Result: ${passed ? ' PASSED' : ' FAILED'}`);

  if (!passed) {
    console.log('\n  System Struggled with Spike:');
    if (errorRate >= 0.10) {
      console.log(`   - High error rate: ${(errorRate * 100).toFixed(2)}%`);
      console.log('     → May need rate limiting');
      console.log('     → Consider request queue/throttling');
      console.log('     → Check connection pool size');
    }
    if (p95Duration >= 2000) {
      console.log(`   - Slow response times: ${p95Duration.toFixed(2)}ms (p95)`);
      console.log('     → May need horizontal scaling');
      console.log('     → Check for bottlenecks (DB, CPU)');
      console.log('     → Consider caching strategy');
    }
  } else {
    console.log('\n System Handled Spike Well:');
    console.log('   - Error rate acceptable during extreme load');
    console.log('   - Response times remained reasonable');
    console.log('   - System appears resilient to traffic spikes');
  }

  console.log('\n Recommendations:');
  console.log('   - Monitor auto-scaling triggers');
  console.log('   - Review connection pool settings');
  console.log('   - Consider implementing rate limiting');
  console.log('   - Set up traffic spike alerts');
  console.log('==========================\n');

  return {
    'stdout': JSON.stringify(data, null, 2),
    'loadtest/reports/spike-summary.json': JSON.stringify(data, null, 2),
  };
}
