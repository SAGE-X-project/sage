# CLAUDE.md - Guidelines for AI-Assisted Development

This document provides guidelines for using AI assistants (like Claude) when contributing to the SAGE project.

## Table of Contents

- [Overview](#overview)
- [Creating Pull Requests](#creating-pull-requests)
- [Creating Issues](#creating-issues)
- [Best Practices](#best-practices)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Documentation Requirements](#documentation-requirements)

## Overview

When using AI assistants to contribute to SAGE, always ensure that:
1. All contributions meet the project's quality standards
2. License compliance is maintained (LGPL-v3 for Go code, MIT for smart contracts)
3. Comprehensive testing is included
4. Documentation is updated appropriately

## Creating Pull Requests

### Using the PR Template

All pull requests must use the provided template located at `.github/PULL_REQUEST_TEMPLATE.md`.

#### Template Sections

1. **Description**: Provide a clear summary of your changes
   - Be specific about what was changed and why
   - Reference any related issues or discussions

2. **Type of Change**: Select all that apply
   - Bug fix
   - New feature
   - Breaking change
   - Documentation update
   - Performance improvement
   - Code refactoring
   - Dependency update
   - Configuration change
   - Test improvement

3. **Related Issues**: Link to GitHub issues
   ```markdown
   Fixes #123
   Related to #456
   ```

4. **Motivation and Context**: Explain why this change is needed
   - What problem does it solve?
   - What use case does it enable?

5. **Changes Made**: List specific changes
   ```markdown
   - Added LGPL-v3 license headers to storage package
   - Updated LICENSE file to standard format
   - Refactored session management for better performance
   ```

6. **Testing**: Describe how you tested your changes
   - Include test configuration (Go version, OS, architecture)
   - List which test suites were run
   - Provide test coverage information if applicable

7. **Documentation**: Update all relevant documentation
   - Code comments
   - README.md
   - docs/ directory
   - API documentation
   - CHANGELOG.md

### PR Checklist Example

When creating a PR, ensure all checklist items are completed:

```markdown
- [x] My code follows the project's style guidelines
- [x] I have performed a self-review of my code
- [x] I have commented my code, particularly in hard-to-understand areas
- [x] I have made corresponding changes to the documentation
- [x] My changes generate no new warnings
- [x] I have added tests that prove my fix is effective or that my feature works
- [x] New and existing unit tests pass locally with my changes
- [x] License headers are present in all new files
```

### PR Title Format

Use conventional commit format for PR titles:

```
<type>: <description>

Examples:
- fix: Add LGPL-v3 license headers to storage package
- feat: Implement multi-chain DID resolution
- docs: Update architecture documentation
- perf: Optimize HPKE encryption performance
- refactor: Simplify session management code
- test: Add integration tests for blockchain client
- chore: Update dependencies to latest versions
```

### PR Description Best Practices

**Good Example:**
```markdown
## Description
This PR adds missing LGPL-v3 license headers to 9 files in the storage package
and updates the LICENSE file to the standard LGPL-v3 format.

## Motivation and Context
During a license compliance audit, we discovered that 9 files in pkg/storage/
were missing the required LGPL-v3 license headers. Additionally, the LICENSE
file contained the full GPL v3 text, which is unnecessary for LGPL-v3 as it
only needs to reference GPL v3.

## Changes Made
- Updated LICENSE to standard LGPL-v3 format (removed full GPL v3 text)
- Added LGPL-v3 headers to 9 storage package files
- Verified 100% license compliance across all 189 Go source files

## Testing
- [x] Verified all files compile successfully
- [x] Ran full test suite: `go test ./...`
- [x] Checked license headers with grep
```

**Bad Example:**
```markdown
## Description
Fixed some stuff

## Changes Made
- Updated files
```

## Creating Issues

### Available Issue Templates

The project provides several issue templates in `.github/ISSUE_TEMPLATE/`:

1. **Bug Report** (`bug_report.yml`)
   - Use for reporting bugs or unexpected behavior
   - Include reproduction steps, environment details, and logs

2. **Feature Request** (`feature_request.yml`)
   - Use for suggesting new features or enhancements
   - Include problem statement, proposed solution, and benefits

3. **Documentation** (`documentation.yml`)
   - Use for documentation issues or improvements
   - Include location, problem description, and suggestions

4. **Performance** (`performance.yml`)
   - Use for performance issues or optimization suggestions
   - Include benchmark results and profiling data

### Issue Title Format

Use clear, descriptive titles:

```
Good:
- [Bug] HPKE encryption fails with large payloads (>1MB)
- [Feature] Add support for Cardano blockchain
- [Docs] Missing API documentation for DID resolver
- [Performance] High memory usage during session cleanup

Bad:
- Problem with encryption
- Need new feature
- Documentation issue
```

## Best Practices

### Code Quality

1. **Follow Go Best Practices**
   - Use `gofmt` for formatting
   - Run `golangci-lint run` and fix all issues
   - Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines

2. **Error Handling**
   - Always check and handle errors appropriately
   - Use wrapped errors with context: `fmt.Errorf("operation failed: %w", err)`
   - Return early on errors

3. **Code Comments**
   - Add comments for exported functions and types
   - Explain complex algorithms or non-obvious code
   - Keep comments up-to-date with code changes

### License Compliance

All new Go files must include the LGPL-v3 license header:

```go
// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package yourpackage
```

Smart contract files (Solidity) use MIT license:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;
```

### Git Commit Messages

Follow conventional commit format:

```
<type>: <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Formatting changes (no code logic change)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks, dependency updates
- `ci`: CI/CD changes

Example:
```
feat: Add support for Ed25519 signatures in DID documents

Implement Ed25519 signature algorithm for DID document signing and
verification. This adds support for the EdDSA algorithm family as
specified in RFC 8032.

- Add Ed25519 key generation and import functions
- Implement Ed25519 signing and verification
- Add comprehensive test suite with test vectors
- Update documentation with Ed25519 examples

Fixes #123
```

## Testing Requirements

### Unit Tests

- All new functions must have unit tests
- Aim for >80% code coverage
- Use table-driven tests for multiple test cases

Example:
```go
func TestSessionCreate(t *testing.T) {
    tests := []struct {
        name    string
        input   *Session
        wantErr bool
    }{
        {
            name:    "valid session",
            input:   &Session{ID: "test", ClientDID: "did:sage:123"},
            wantErr: false,
        },
        {
            name:    "empty session ID",
            input:   &Session{ClientDID: "did:sage:123"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := CreateSession(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Integration Tests

- Test interactions between components
- Use mock blockchain clients when appropriate
- Clean up resources after tests

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...

# Run specific package
go test ./pkg/agent/...

# Verbose output
go test -v ./...
```

## Documentation Requirements

### Code Documentation

Document all exported items:

```go
// Session represents a secure communication session between two agents.
// It manages encryption keys, nonces, and session state.
type Session struct {
    ID        string    // Unique session identifier
    ClientDID string    // Client's DID
    ServerDID string    // Server's DID
    CreatedAt time.Time // Session creation timestamp
}

// Create establishes a new session with the given parameters.
// It generates a unique session ID and initializes encryption keys.
//
// Returns an error if the session parameters are invalid or if
// key generation fails.
func (s *Session) Create(clientDID, serverDID string) error {
    // Implementation
}
```

### README Updates

Update relevant README files when:
- Adding new features
- Changing APIs
- Modifying configuration options
- Adding new dependencies

### Architecture Documentation

Update `docs/ARCHITECTURE.md` when:
- Adding new components
- Changing system design
- Modifying data flows
- Updating security mechanisms

### CHANGELOG

Update `CHANGELOG.md` following [Keep a Changelog](https://keepachangelog.com/) format:

```markdown
## [Unreleased]

### Added
- Support for Ed25519 signatures in DID documents
- New performance metrics for HPKE operations

### Changed
- Updated DID resolution to use caching layer
- Improved session cleanup performance

### Fixed
- Fixed memory leak in session manager
- Corrected error handling in blockchain client

### Security
- Added rate limiting for DID resolution requests
```

## Examples

### Complete PR Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feat/add-ed25519-support
   ```

2. **Make changes and commit**
   ```bash
   git add .
   git commit -m "feat: Add Ed25519 signature support

   - Implement Ed25519 key generation
   - Add signing and verification functions
   - Include comprehensive test suite
   - Update documentation"
   ```

3. **Push and create PR**
   ```bash
   git push -u origin feat/add-ed25519-support
   gh pr create --fill
   ```

4. **Fill out PR template completely**
   - Check all relevant boxes
   - Provide detailed description
   - Link to related issues
   - Include test results

5. **Address review comments**
   - Make requested changes
   - Update tests and documentation
   - Push updates to the same branch

### Complete Issue Workflow

1. **Choose appropriate template**
   - Bug report for bugs
   - Feature request for new features
   - Documentation for docs issues
   - Performance for optimization suggestions

2. **Fill out template completely**
   - Provide all required information
   - Include reproduction steps or use cases
   - Add relevant logs or examples

3. **Use appropriate labels**
   - Let maintainers apply labels if unsure
   - Use priority indicators when relevant

4. **Engage in discussion**
   - Respond to questions from maintainers
   - Provide additional information if requested
   - Help test fixes or implementations

## AI Assistant Specific Guidelines

### When Using Claude or Other AI Assistants

1. **Always review AI-generated code**
   - Verify correctness and security
   - Ensure it follows project conventions
   - Test thoroughly

2. **Maintain context**
   - Provide relevant project context to the AI
   - Reference existing code patterns
   - Mention project-specific requirements

3. **Verify license compliance**
   - Ensure all new files have proper headers
   - Check that generated code is compatible with LGPL-v3
   - Avoid incorporating GPL-incompatible code

4. **Document AI assistance**
   - Be transparent about AI-generated contributions
   - Review and understand all generated code
   - Take responsibility for the final submission

5. **Security considerations**
   - Never share sensitive information with AI
   - Review crypto code extra carefully
   - Validate security-critical functions

## Resources

- [SAGE Architecture Documentation](docs/ARCHITECTURE.md)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [License Information](LICENSE)
- [Project README](README.md)

## Questions?

- Open a [GitHub Discussion](https://github.com/sage-x-project/sage/discussions)
- Check existing [Issues](https://github.com/sage-x-project/sage/issues)
- Read the [Documentation](https://github.com/sage-x-project/sage/tree/main/docs)

---

**Note**: This document is a living guide. If you have suggestions for improvements, please open a pull request or create an issue using the Documentation template.
