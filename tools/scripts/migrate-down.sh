#!/bin/bash
# SAGE Database Migration Down Script

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-sage}"
DB_PASSWORD="${DB_PASSWORD:-sage}"
DB_NAME="${DB_NAME:-sage}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

# Migration configuration
MIGRATIONS_DIR="${MIGRATIONS_DIR:-./migrations}"
STEPS="${1:-1}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}SAGE Database Migration (Down)${NC}"
echo "=================================="
echo "Host: $DB_HOST:$DB_PORT"
echo "Database: $DB_NAME"
echo "Migrations: $MIGRATIONS_DIR"
echo "Steps to rollback: $STEPS"
echo ""

# Warning
echo -e "${RED}WARNING: This will rollback migrations and may delete data!${NC}"
read -p "Are you sure you want to continue? (yes/no): " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo "Rollback cancelled."
    exit 0
fi

# Construct database URL
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

# Run migrations
echo -e "${YELLOW}Rolling back migrations...${NC}"
migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" down "$STEPS"

echo ""
echo -e "${GREEN}Rollback completed successfully!${NC}"

# Show current version
echo ""
echo -e "${YELLOW}Current schema version:${NC}"
migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" version

echo ""
echo -e "${GREEN}Done!${NC}"
