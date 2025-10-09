-- SAGE Development Seed Data
-- This file contains test data for development environment

-- Insert test DIDs
INSERT INTO dids (did, public_key, owner_address, key_type, revoked) VALUES
    ('did:sage:dev:alice', E'\\x302a300506032b6570032100c6ba7d8a9ed1e2e87a4832c0c5f6e4c24a9c6a8e5c8f9d4c3b2a1f0e8d7c6b5a4', '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb', 'Ed25519', false),
    ('did:sage:dev:bob', E'\\x302a300506032b6570032100d7cb8e9b9fe2f3f98b5943d1d6g7f5d35b0d7b9f6d9g0e5d4c3b2a1f0e9e8d7c', '0x853f46Dd7714Ab2533a4a844Cd0e8696e1bFec', 'Ed25519', false),
    ('did:sage:dev:charlie', E'\\x302a300506032b6570032100e8dc9f0c0gf3g0g09c6a54e2e7h8g6e46c1e8c0g7e0h1f6e5d4c3b2a1f0f9f8e', '0x964g57Ee8825Bc3644b5b955De1f9707f2cGfd', 'Ed25519', false),
    ('did:sage:dev:revoked', E'\\x302a300506032b6570032100f9ed0g1d1hg4h1h10d7b65f3f8i9h7f57d2f9d1h8f1i2g7f6e5d4c3b2a1g0g9g', '0xa75h68Ff9936Cd4755c6c066Ef2g0818g3dHge', 'Ed25519', true)
ON CONFLICT (did) DO NOTHING;

-- Insert test sessions (expires in 1 hour from now)
INSERT INTO sessions (id, client_did, server_did, session_key, created_at, expires_at, last_activity, metadata) VALUES
    ('550e8400-e29b-41d4-a716-446655440000',
     'did:sage:dev:alice',
     'did:sage:dev:bob',
     E'\\x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
     NOW(),
     NOW() + INTERVAL '1 hour',
     NOW(),
     '{"purpose": "test", "environment": "development"}'::jsonb),

    ('660f9511-f30c-52e5-b827-557766551111',
     'did:sage:dev:bob',
     'did:sage:dev:alice',
     E'\\xfedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210',
     NOW(),
     NOW() + INTERVAL '2 hours',
     NOW(),
     '{"purpose": "bidirectional", "environment": "development"}'::jsonb),

    ('770fa622-g41d-63f6-c938-668877662222',
     'did:sage:dev:charlie',
     'did:sage:dev:alice',
     E'\\x123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef1',
     NOW(),
     NOW() + INTERVAL '30 minutes',
     NOW(),
     '{"purpose": "short-lived", "environment": "development"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- Insert test nonces (for replay attack prevention)
INSERT INTO nonces (nonce, session_id, used_at, expires_at) VALUES
    ('nonce_alice_bob_001', '550e8400-e29b-41d4-a716-446655440000', NOW(), NOW() + INTERVAL '5 minutes'),
    ('nonce_bob_alice_001', '660f9511-f30c-52e5-b827-557766551111', NOW(), NOW() + INTERVAL '5 minutes'),
    ('nonce_charlie_alice_001', '770fa622-g41d-63f6-c938-668877662222', NOW(), NOW() + INTERVAL '5 minutes')
ON CONFLICT (nonce) DO NOTHING;

-- Output summary
DO $$
BEGIN
    RAISE NOTICE 'Development seed data loaded successfully';
    RAISE NOTICE 'DIDs created: 4 (3 active, 1 revoked)';
    RAISE NOTICE 'Sessions created: 3';
    RAISE NOTICE 'Nonces created: 3';
END $$;
