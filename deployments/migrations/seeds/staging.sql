-- SAGE Staging Seed Data
-- This file contains minimal test data for staging environment

-- Insert minimal test DIDs for staging verification
INSERT INTO dids (did, public_key, owner_address, key_type, revoked) VALUES
    ('did:sage:staging:test1', E'\\x302a300506032b6570032100a1b2c3d4e5f6071829303a4b5c6d7e8f9091a2b3c4d5e6f7081920313a4b5c6d', '0x123a24Bc5634D0532925a3b844Bc9e7595f0bEb', 'Ed25519', false),
    ('did:sage:staging:test2', E'\\x302a300506032b6570032100b2c3d4e5f6071829303a4b5c6d7e8f9091a2b3c4d5e6f7081920313a4b5c6d7e', '0x234b35Cd6745E1643a36b4c955Cd0f8686f1cFec', 'Ed25519', false)
ON CONFLICT (did) DO NOTHING;

-- No sessions or nonces in staging seed (will be created by actual tests)

-- Output summary
DO $$
BEGIN
    RAISE NOTICE 'Staging seed data loaded successfully';
    RAISE NOTICE 'DIDs created: 2 (minimal test accounts)';
    RAISE NOTICE 'Sessions: None (created by tests)';
    RAISE NOTICE 'Nonces: None (created by tests)';
END $$;
