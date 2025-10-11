#!/bin/bash
# SAGE Database Backup Script

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-sage}"
DB_NAME="${DB_NAME:-sage}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/sage_backup_${TIMESTAMP}.sql"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}SAGE Database Backup${NC}"
echo "=================================="
echo "Host: $DB_HOST:$DB_PORT"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo ""

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Perform backup
echo -e "${YELLOW}Creating backup...${NC}"
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    --format=plain \
    --no-owner \
    --no-acl \
    --clean \
    --if-exists \
    > "$BACKUP_FILE"

# Compress backup
echo -e "${YELLOW}Compressing backup...${NC}"
gzip "$BACKUP_FILE"
BACKUP_FILE="${BACKUP_FILE}.gz"

# Get file size
BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)

echo ""
echo -e "${GREEN}Backup completed successfully!${NC}"
echo "File: $BACKUP_FILE"
echo "Size: $BACKUP_SIZE"
echo ""

# Keep only last 7 backups
echo -e "${YELLOW}Cleaning old backups (keeping last 7)...${NC}"
cd "$BACKUP_DIR"
ls -t sage_backup_*.sql.gz 2>/dev/null | tail -n +8 | xargs -r rm --
BACKUP_COUNT=$(ls -1 sage_backup_*.sql.gz 2>/dev/null | wc -l)
echo "Total backups: $BACKUP_COUNT"

echo ""
echo -e "${GREEN}Done!${NC}"
