# Version Update Script

## Overview

`update-version.sh` is an automated script that synchronizes version numbers across all SAGE project files. It eliminates manual version updates and prevents version inconsistencies.

## Files Updated

The script updates version in 6 files:

1. **VERSION** - Project version file
2. **README.md** - "What's New" section with current date
3. **contracts/ethereum/package.json** - npm package version
4. **contracts/ethereum/package-lock.json** - npm lockfile version
5. **pkg/version/version.go** - Go runtime version constant
6. **lib/export.go** - C shared library version

## Usage

### Basic Usage

```bash
./tools/scripts/update-version.sh <new-version>
```

### Examples

```bash
# Update to version 1.4.0
./tools/scripts/update-version.sh 1.4.0

# Update to beta version
./tools/scripts/update-version.sh 2.0.0-beta.1

# Update to release candidate
./tools/scripts/update-version.sh 1.5.0-rc.1
```

## Version Format

The script follows **Semantic Versioning** (semver):

```
MAJOR.MINOR.PATCH[-PRERELEASE]

Examples:
  1.4.0           # Standard release
  2.0.0-beta.1    # Beta release
  1.5.0-rc.2      # Release candidate
  3.0.0-alpha.1   # Alpha release
```

**Validation Rules:**
- MAJOR, MINOR, PATCH must be numbers
- PRERELEASE is optional (alphanumeric + dots)
- Must match regex: `^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$`

## Workflow

### 1. Check Current Version

```bash
cat VERSION
# Output: 1.3.0
```

### 2. Run Update Script

```bash
./tools/scripts/update-version.sh 1.4.0
```

**Output:**
```
SAGE Version Update
================================
Current version: 1.3.0
New version:     1.4.0

Update version to 1.4.0? (y/N) y

Updating version files...

[1/6] Updating VERSION...
       VERSION updated
[2/6] Updating README.md...
       README.md updated
[3/6] Updating contracts/ethereum/package.json...
       package.json updated
[4/6] Updating contracts/ethereum/package-lock.json...
       package-lock.json updated
[5/6] Updating pkg/version/version.go...
       version.go updated
[6/6] Updating lib/export.go...
       export.go updated

Version update complete!

Verification:
  VERSION:       1.4.0
  README.md:     1.4.0
  package.json:  1.4.0
  version.go:    1.4.0
  export.go:     1.4.0

Next steps:
  1. Review changes: git diff
  2. Run tests:      make test
  3. Commit:         git add -A && git commit -m "chore: Bump version to v1.4.0"
  4. Tag release:    git tag v1.4.0
```

### 3. Review Changes

```bash
git diff
```

**Expected changes:**
- VERSION: `1.3.0` → `1.4.0`
- README.md: `v1.3.0 (2025-10-25)` → `v1.4.0 (2025-10-26)`
- package.json: `"version": "1.3.0"` → `"version": "1.4.0"`
- package-lock.json: Same as package.json
- version.go: `Version = "1.3.0"` → `Version = "1.4.0"`
- export.go: `C.CString("1.3.0")` → `C.CString("1.4.0")`

### 4. Run Tests

```bash
make test
```

Ensure all tests pass before committing.

### 5. Commit and Tag

```bash
# Commit version changes
git add -A
git commit -m "chore: Bump version to v1.4.0"

# Create git tag
git tag v1.4.0

# Push changes and tag
git push origin main
git push origin v1.4.0
```

## Release Workflow

### Patch Release (Bug Fixes)

```bash
# Current: 1.3.0
./tools/scripts/update-version.sh 1.3.1

# Changes:
# - Bug fixes only
# - No new features
# - No breaking changes
```

### Minor Release (New Features)

```bash
# Current: 1.3.1
./tools/scripts/update-version.sh 1.4.0

# Changes:
# - New features added
# - Backward compatible
# - May include bug fixes
```

### Major Release (Breaking Changes)

```bash
# Current: 1.4.0
./tools/scripts/update-version.sh 2.0.0

# Changes:
# - Breaking API changes
# - Major refactoring
# - May include new features and bug fixes
```

### Pre-release Versions

```bash
# Alpha (internal testing)
./tools/scripts/update-version.sh 2.0.0-alpha.1

# Beta (public testing)
./tools/scripts/update-version.sh 2.0.0-beta.1

# Release Candidate (final testing)
./tools/scripts/update-version.sh 2.0.0-rc.1

# Final release
./tools/scripts/update-version.sh 2.0.0
```

## Troubleshooting

### Error: Invalid version format

```bash
./tools/scripts/update-version.sh 1.4
# Error: Invalid version format. Use semantic versioning (e.g., 1.4.0 or 2.0.0-beta.1)
```

**Solution:** Use full semver format (MAJOR.MINOR.PATCH)
```bash
./tools/scripts/update-version.sh 1.4.0
```

### Error: jq not found

The script works without `jq`, but it's recommended for JSON updates:

```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# Without jq, the script uses sed (slightly slower but functional)
```

### Verification Failed

If verification shows mismatched versions:

```bash
# Check files manually
cat VERSION
grep "What's New" README.md
grep version contracts/ethereum/package.json
grep Version pkg/version/version.go
grep CString lib/export.go

# Fix manually if needed
vim <file>

# Re-run verification
./tools/scripts/update-version.sh <version>
```

## Script Features

###  Safety Features

- **Version format validation** - Rejects invalid semver
- **Confirmation prompt** - Asks before making changes
- **Automatic verification** - Checks all files after update
- **Backup files** - Creates .bak files during sed operations (auto-deleted)

###  Smart Updates

- **README.md date** - Automatically updates to current date
- **JSON formatting** - Uses jq if available for clean JSON
- **Cross-platform** - Works on macOS and Linux

###  User-Friendly

- **Color output** - Green (success), Yellow (warning), Red (error), Blue (info)
- **Progress indicators** - Shows [1/6], [2/6], etc.
- **Clear verification** - Lists all updated versions
- **Next steps guide** - Suggests git commands

## CI/CD Integration

### GitHub Actions

```yaml
name: Release

on:
  workflow_dispatch:
    inputs:
      version:
        description: 'New version (e.g., 1.4.0)'
        required: true

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Update version
        run: |
          echo "y" | ./tools/scripts/update-version.sh ${{ github.event.inputs.version }}

      - name: Run tests
        run: make test

      - name: Commit and tag
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add -A
          git commit -m "chore: Bump version to v${{ github.event.inputs.version }}"
          git tag v${{ github.event.inputs.version }}
          git push origin main
          git push origin v${{ github.event.inputs.version }}
```

### Makefile Integration

```makefile
# Bump version
bump-version:
	@read -p "Enter new version: " version; \
	./tools/scripts/update-version.sh $$version

# Quick version update (skips confirmation for CI/CD)
bump-version-ci:
	@echo "y" | ./tools/scripts/update-version.sh $(VERSION)
```

Usage:
```bash
# Interactive
make bump-version

# CI/CD
make bump-version-ci VERSION=1.4.0
```

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.3.0 | 2025-10-25 | SageRegistryV4 multi-key support |
| 1.2.0 | 2025-10-20 | WebSocket transport added |
| 1.1.0 | 2025-10-15 | HPKE integration completed |
| 1.0.3 | 2025-10-10 | Bug fixes and performance improvements |

## Best Practices

### Before Version Update

1.  Ensure all tests pass
2.  Update CHANGELOG.md
3.  Review pending PRs
4.  Check for breaking changes

### After Version Update

1.  Run full test suite
2.  Update deployment documentation
3.  Create GitHub release
4.  Announce release (Discord, Twitter, etc.)

### Version Numbering Guidelines

**When to bump MAJOR (X.0.0):**
- Breaking API changes
- Removed features
- Major architectural changes

**When to bump MINOR (1.X.0):**
- New features (backward compatible)
- Significant improvements
- New transport implementations

**When to bump PATCH (1.3.X):**
- Bug fixes
- Security patches
- Documentation updates
- Performance improvements

## See Also

- [Semantic Versioning](https://semver.org/) - Official semver specification
- [CHANGELOG.md](../../CHANGELOG.md) - Project changelog
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines

## License

LGPL-3.0 - See LICENSE file for details.
