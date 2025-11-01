#!/bin/bash
# SAGE Version Update Script
# Updates version across all project files

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Files to update
VERSION_FILE="$PROJECT_ROOT/VERSION"
README_FILE="$PROJECT_ROOT/README.md"
PACKAGE_JSON="$PROJECT_ROOT/contracts/ethereum/package.json"
PACKAGE_LOCK="$PROJECT_ROOT/contracts/ethereum/package-lock.json"
VERSION_GO="$PROJECT_ROOT/pkg/version/version.go"
EXPORT_GO="$PROJECT_ROOT/lib/export.go"

# Function to display usage
usage() {
    echo "Usage: $0 <new-version>"
    echo ""
    echo "Examples:"
    echo "  $0 1.4.0"
    echo "  $0 2.0.0-beta.1"
    echo ""
    echo "This script updates version in the following files:"
    echo "  1. VERSION"
    echo "  2. README.md"
    echo "  3. contracts/ethereum/package.json"
    echo "  4. contracts/ethereum/package-lock.json"
    echo "  5. pkg/version/version.go"
    echo "  6. lib/export.go"
    exit 1
}

# Check arguments
if [ $# -ne 1 ]; then
    usage
fi

NEW_VERSION="$1"

# Validate semantic version format
if ! [[ "$NEW_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo -e "${RED}Error: Invalid version format. Use semantic versioning (e.g., 1.4.0 or 2.0.0-beta.1)${NC}"
    exit 1
fi

# Get current version
if [ -f "$VERSION_FILE" ]; then
    CURRENT_VERSION=$(cat "$VERSION_FILE")
else
    CURRENT_VERSION="unknown"
fi

echo -e "${BLUE}SAGE Version Update${NC}"
echo "================================"
echo -e "Current version: ${YELLOW}$CURRENT_VERSION${NC}"
echo -e "New version:     ${GREEN}$NEW_VERSION${NC}"
echo ""

# Confirm update
read -p "Update version to $NEW_VERSION? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo -e "${BLUE}Updating version files...${NC}"
echo ""

# 1. Update VERSION file
echo -e "${YELLOW}[1/6]${NC} Updating VERSION..."
echo "$NEW_VERSION" > "$VERSION_FILE"
echo -e "      ${GREEN}${NC} VERSION updated"

# 2. Update README.md
echo -e "${YELLOW}[2/6]${NC} Updating README.md..."
if [ -f "$README_FILE" ]; then
    # Update version in "What's New" section
    # Pattern: ###  What's New in v1.3.0 (YYYY-MM-DD)
    TODAY=$(date +%Y-%m-%d)
    sed -i.bak -E "s/###  What's New in v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)? \([0-9]{4}-[0-9]{2}-[0-9]{2}\)/###  What's New in v${NEW_VERSION} (${TODAY})/" "$README_FILE"
    rm -f "$README_FILE.bak"
    echo -e "      ${GREEN}${NC} README.md updated"
else
    echo -e "      ${YELLOW}${NC}  README.md not found, skipping"
fi

# 3. Update contracts/ethereum/package.json
echo -e "${YELLOW}[3/6]${NC} Updating contracts/ethereum/package.json..."
if [ -f "$PACKAGE_JSON" ]; then
    # Use jq if available, otherwise use sed
    if command -v jq &> /dev/null; then
        TMP_FILE=$(mktemp)
        jq --arg version "$NEW_VERSION" '.version = $version' "$PACKAGE_JSON" > "$TMP_FILE"
        mv "$TMP_FILE" "$PACKAGE_JSON"
    else
        sed -i.bak -E "s/\"version\": \"[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?\"/\"version\": \"${NEW_VERSION}\"/" "$PACKAGE_JSON"
        rm -f "$PACKAGE_JSON.bak"
    fi
    echo -e "      ${GREEN}${NC} package.json updated"
else
    echo -e "      ${YELLOW}${NC}  package.json not found, skipping"
fi

# 4. Update contracts/ethereum/package-lock.json
echo -e "${YELLOW}[4/6]${NC} Updating contracts/ethereum/package-lock.json..."
if [ -f "$PACKAGE_LOCK" ]; then
    # Use jq if available
    if command -v jq &> /dev/null; then
        TMP_FILE=$(mktemp)
        jq --arg version "$NEW_VERSION" '.version = $version' "$PACKAGE_LOCK" > "$TMP_FILE"
        mv "$TMP_FILE" "$PACKAGE_LOCK"
    else
        sed -i.bak -E "s/\"version\": \"[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?\"/\"version\": \"${NEW_VERSION}\"/" "$PACKAGE_LOCK"
        rm -f "$PACKAGE_LOCK.bak"
    fi
    echo -e "      ${GREEN}${NC} package-lock.json updated"
else
    echo -e "      ${YELLOW}${NC}  package-lock.json not found, skipping"
fi

# 5. Update pkg/version/version.go
echo -e "${YELLOW}[5/6]${NC} Updating pkg/version/version.go..."
if [ -f "$VERSION_GO" ]; then
    sed -i.bak -E "s/Version = \"[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?\"/Version = \"${NEW_VERSION}\"/" "$VERSION_GO"
    rm -f "$VERSION_GO.bak"
    echo -e "      ${GREEN}${NC} version.go updated"
else
    echo -e "      ${YELLOW}${NC}  version.go not found, skipping"
fi

# 6. Update lib/export.go
echo -e "${YELLOW}[6/6]${NC} Updating lib/export.go..."
if [ -f "$EXPORT_GO" ]; then
    sed -i.bak -E "s/C\.CString\(\"[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?\"\)/C.CString(\"${NEW_VERSION}\")/" "$EXPORT_GO"
    rm -f "$EXPORT_GO.bak"
    echo -e "      ${GREEN}${NC} export.go updated"
else
    echo -e "      ${YELLOW}${NC}  export.go not found, skipping"
fi

echo ""
echo -e "${GREEN}Version update complete!${NC}"
echo ""

# Verify changes
echo -e "${BLUE}Verification:${NC}"
echo "  VERSION:       $(cat "$VERSION_FILE")"
echo "  README.md:     $(grep -E "What's New in v[0-9]" "$README_FILE" | sed -E 's/.*v([0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?).*/\1/' | head -1)"
if [ -f "$PACKAGE_JSON" ]; then
    if command -v jq &> /dev/null; then
        echo "  package.json:  $(jq -r .version "$PACKAGE_JSON")"
    else
        echo "  package.json:  $(grep -E '"version":' "$PACKAGE_JSON" | sed -E 's/.*"version": "([^"]+)".*/\1/')"
    fi
fi
echo "  version.go:    $(grep -E 'Version = ' "$VERSION_GO" | sed -E 's/.*Version = "([^"]+)".*/\1/')"
echo "  export.go:     $(grep -E 'C\.CString' "$EXPORT_GO" | sed -E 's/.*C\.CString\("([^"]+)"\).*/\1/')"
echo ""

# Suggest next steps
echo -e "${BLUE}Next steps:${NC}"
echo "  1. Review changes: git diff"
echo "  2. Run tests:      make test"
echo "  3. Commit:         git add -A && git commit -m \"chore: Bump version to v${NEW_VERSION}\""
echo "  4. Tag release:    git tag v${NEW_VERSION}"
echo ""
