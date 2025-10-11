#!/bin/bash
# SAGE Database Migration Up Script

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

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}SAGE Database Migration (Up)${NC}"
echo "=================================="
echo "Host: $DB_HOST:$DB_PORT"
echo "Database: $DB_NAME"
echo "Migrations: $MIGRATIONS_DIR"
echo ""

# Construct database URL
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

# Run migrations
echo -e "${YELLOW}Running migrations...${NC}"
migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" up

echo ""
echo -e "${GREEN}Migrations completed successfully!${NC}"

# Show current version
echo ""
echo -e "${YELLOW}Current schema version:${NC}"
migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" version

echo ""
echo -e "${GREEN}Done!${NC}"
