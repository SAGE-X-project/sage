#!/bin/bash
# SAGE Database Seed Script

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-sage}"
DB_NAME="${DB_NAME:-sage}"
ENV="${SAGE_ENV:-dev}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}SAGE Database Seed${NC}"
echo "=================================="
echo "Host: $DB_HOST:$DB_PORT"
echo "Database: $DB_NAME"
echo "Environment: $ENV"
echo ""

# Determine seed file
SEED_FILE="./migrations/seeds/${ENV}.sql"

if [ ! -f "$SEED_FILE" ]; then
    echo "Error: Seed file not found: $SEED_FILE"
    echo "Available environments: dev, staging"
    exit 1
fi

# Run seed
echo -e "${YELLOW}Loading seed data...${NC}"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SEED_FILE"

echo ""
echo -e "${GREEN}Seed data loaded successfully!${NC}"
echo ""

# Show statistics
echo -e "${YELLOW}Database statistics:${NC}"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\
SELECT
    'sessions' as table_name, COUNT(*) as row_count FROM sessions
UNION ALL
SELECT 'nonces', COUNT(*) FROM nonces
UNION ALL
SELECT 'dids', COUNT(*) FROM dids;"

echo ""
echo -e "${GREEN}Done!${NC}"
