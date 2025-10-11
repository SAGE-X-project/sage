-- SAGE Database Schema Rollback
-- Drop all tables and functions

-- Drop triggers first
DROP TRIGGER IF EXISTS update_dids_updated_at ON dids;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (order matters due to foreign keys)
DROP TABLE IF EXISTS nonces;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS dids;
