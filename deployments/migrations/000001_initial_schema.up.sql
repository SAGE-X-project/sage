-- SAGE Database Initial Schema
-- Sessions, Nonces, and DIDs tables

-- Sessions table: Stores active secure sessions
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    client_did TEXT NOT NULL,
    server_did TEXT NOT NULL,
    session_key BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    last_activity TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB,
    CONSTRAINT sessions_expires_at_check CHECK (expires_at > created_at)
);

-- Indexes for sessions
CREATE INDEX idx_sessions_client_did ON sessions(client_did);
CREATE INDEX idx_sessions_server_did ON sessions(server_did);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_created_at ON sessions(created_at);

-- Nonces table: Replay attack prevention
CREATE TABLE IF NOT EXISTS nonces (
    nonce TEXT PRIMARY KEY,
    session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
    used_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    CONSTRAINT nonces_expires_at_check CHECK (expires_at > used_at)
);

-- Indexes for nonces
CREATE INDEX idx_nonces_session_id ON nonces(session_id);
CREATE INDEX idx_nonces_expires_at ON nonces(expires_at);
CREATE INDEX idx_nonces_used_at ON nonces(used_at);

-- DIDs table: Cache for blockchain DIDs
CREATE TABLE IF NOT EXISTS dids (
    did TEXT PRIMARY KEY,
    public_key BYTEA NOT NULL,
    owner_address TEXT NOT NULL,
    key_type TEXT NOT NULL DEFAULT 'Ed25519',
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for dids
CREATE INDEX idx_dids_owner_address ON dids(owner_address);
CREATE INDEX idx_dids_revoked ON dids(revoked) WHERE revoked = TRUE;
CREATE INDEX idx_dids_key_type ON dids(key_type);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to update updated_at on dids table
CREATE TRIGGER update_dids_updated_at
    BEFORE UPDATE ON dids
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
