// SAGE Load Testing Configuration

export const config = {
  // Base URL for API endpoints
  baseUrl: __ENV.SAGE_BASE_URL || 'http://localhost:8080',

  // Test duration settings
  durations: {
    rampUp: '30s',
    steady: '1m',
    rampDown: '30s',
  },

  // Virtual user settings
  vus: {
    baseline: 10,
    stress: 100,
    spike: 500,
    soak: 50,
    'concurrent-sessions': 60,
    'did-operations': 100,
    'hpke-operations': 80,
    'mixed-workload': 75,
  },

  // Performance thresholds
  thresholds: {
    baseline: {
      http_req_duration: ['p(95)<500', 'p(99)<1000'],  // 95% < 500ms, 99% < 1s
      http_req_failed: ['rate<0.01'],                   // < 1% errors
      http_reqs: ['rate>10'],                            // > 10 req/s
    },
    stress: {
      http_req_duration: ['p(95)<1000', 'p(99)<2000'], // 95% < 1s, 99% < 2s
      http_req_failed: ['rate<0.05'],                   // < 5% errors
      http_reqs: ['rate>50'],                            // > 50 req/s
    },
    soak: {
      http_req_duration: ['p(95)<500', 'p(99)<1000'],
      http_req_failed: ['rate<0.01'],
      // No memory leaks: should be stable over time
    },
    spike: {
      http_req_duration: ['p(95)<2000', 'p(99)<5000'], // More lenient during spikes
      http_req_failed: ['rate<0.10'],                   // < 10% errors during spike
    },
    'concurrent-sessions': {
      http_req_duration: ['p(95)<1000', 'p(99)<2000'],
      http_req_failed: ['rate<0.02'],                   // < 2% errors
      http_reqs: ['rate>20'],                            // > 20 req/s
    },
    'did-operations': {
      http_req_duration: ['p(95)<800', 'p(99)<1500'],
      http_req_failed: ['rate<0.02'],                   // < 2% errors
      http_reqs: ['rate>30'],                            // > 30 req/s
    },
    'hpke-operations': {
      http_req_duration: ['p(95)<1500', 'p(99)<3000'], // HPKE is CPU-intensive
      http_req_failed: ['rate<0.03'],                   // < 3% errors
      http_reqs: ['rate>15'],                            // > 15 req/s
    },
    'mixed-workload': {
      http_req_duration: ['p(95)<1000', 'p(99)<2000'],
      http_req_failed: ['rate<0.02'],                   // < 2% errors
      http_reqs: ['rate>25'],                            // > 25 req/s
    },
  },

  // API endpoints
  endpoints: {
    health: '/debug/health',
    kemPub: '/debug/kem-pub',
    serverDid: '/debug/server-did',
    registerAgent: '/debug/register-agent',
    sendMessage: '/v1/a2a:sendMessage',
    protected: '/protected',
  },

  // Test data
  testAgent: {
    did: 'did:sage:ethereum:0xLoadTest123',
    name: 'Load Test Agent',
    isActive: true,
  },

  // Sleep duration between iterations
  sleepDuration: 1,
};

// Export environment-specific config
export function getConfig(scenario) {
  const env = __ENV.SAGE_ENV || 'local';

  // Environment-specific overrides
  const envConfigs = {
    local: {
      baseUrl: 'http://localhost:8080',
    },
    staging: {
      baseUrl: 'https://staging-api.sage.example.com',
      sleepDuration: 2,
    },
    production: {
      baseUrl: 'https://api.sage.example.com',
      sleepDuration: 3,
      // More conservative thresholds
      thresholds: {
        http_req_duration: ['p(95)<1000'],
        http_req_failed: ['rate<0.001'],
      },
    },
  };

  return {
    ...config,
    ...envConfigs[env],
    thresholds: scenario ? config.thresholds[scenario] : config.thresholds.baseline,
  };
}
