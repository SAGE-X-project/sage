#!/bin/bash
# SAGE Database Restore Script

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-sage}"
DB_NAME="${DB_NAME:-sage}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if backup file is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Backup file not specified${NC}"
    echo "Usage: $0 <backup_file>"
    echo "Example: $0 ./backups/sage_backup_20251010_121500.sql.gz"
    exit 1
fi

BACKUP_FILE="$1"

# Check if file exists
if [ ! -f "$BACKUP_FILE" ]; then
    echo -e "${RED}Error: Backup file not found: $BACKUP_FILE${NC}"
    exit 1
fi

echo -e "${YELLOW}SAGE Database Restore${NC}"
echo "=================================="
echo "Host: $DB_HOST:$DB_PORT"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo "Backup: $BACKUP_FILE"
echo ""

# Warning
echo -e "${RED}WARNING: This will DROP all existing tables and data!${NC}"
read -p "Are you sure you want to continue? (yes/no): " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo "Restore cancelled."
    exit 0
fi

# Decompress if needed
TEMP_FILE=""
if [[ "$BACKUP_FILE" == *.gz ]]; then
    echo -e "${YELLOW}Decompressing backup...${NC}"
    TEMP_FILE="/tmp/sage_restore_$(date +%s).sql"
    gunzip -c "$BACKUP_FILE" > "$TEMP_FILE"
    RESTORE_FILE="$TEMP_FILE"
else
    RESTORE_FILE="$BACKUP_FILE"
fi

# Perform restore
echo -e "${YELLOW}Restoring database...${NC}"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" < "$RESTORE_FILE"

# Cleanup temp file
if [ -n "$TEMP_FILE" ]; then
    rm "$TEMP_FILE"
fi

echo ""
echo -e "${GREEN}Restore completed successfully!${NC}"
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
