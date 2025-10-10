# Contributing to SAGE

Thank you for your interest in contributing to SAGE (Secure Agent Guarantee Engine)! This document provides guidelines and best practices for contributing to the project.

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [Getting Started](#getting-started)
3. [Development Environment](#development-environment)
4. [Branch Strategy](#branch-strategy)
5. [Commit Guidelines](#commit-guidelines)
6. [Pull Request Process](#pull-request-process)
7. [Code Style](#code-style)
8. [Testing Requirements](#testing-requirements)
9. [Documentation](#documentation)
10. [Review Process](#review-process)

## Code of Conduct

### Our Standards

- **Be respectful**: Treat all contributors with respect and professionalism
- **Be collaborative**: Work together to achieve the best outcomes
- **Be constructive**: Provide helpful feedback and be open to receiving it
- **Be inclusive**: Welcome contributors from all backgrounds and skill levels

### Unacceptable Behavior

- Harassment, discriminatory language, or personal attacks
- Spam, trolling, or intentionally disruptive behavior
- Sharing private information without permission
- Any conduct that violates applicable laws

## Getting Started

### Prerequisites

- **Go 1.24+**: Required for backend development
- **Node.js 18+**: Required for smart contract development
- **Git**: Version control
- **Make**: Build automation
- **Docker** (optional): For containerized development and testing

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/sage.git
   cd sage
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/sage-x-project/sage.git
   ```

### Install Dependencies

```bash
# Go dependencies
go mod download

# Smart contract dependencies
cd contracts/ethereum
npm install
cd ../..

# Verify installation
make test
```

## Development Environment

### IDE Setup

#### VS Code (Recommended)

Install extensions:
- Go (golang.go)
- Solidity (juanblanco.solidity)
- EditorConfig (editorconfig.editorconfig)

Workspace settings (`.vscode/settings.json`):
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "file",
  "go.formatTool": "gofmt",
  "editor.formatOnSave": true,
  "files.eol": "\n"
}
```

#### GoLand

- Enable Go modules integration
- Configure golangci-lint as external tool
- Set up file watchers for automatic formatting

### Environment Variables

Create `.env.local` for local development:
```bash
# Blockchain RPC endpoints
ETHEREUM_RPC_URL=http://localhost:8545
SOLANA_RPC_URL=http://localhost:8899

# Redis (optional for local testing)
REDIS_URL=redis://localhost:6379

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json
```

### Local Blockchain Networks

```bash
# Start Hardhat local Ethereum node
cd contracts/ethereum
npx hardhat node

# Start Solana test validator (separate terminal)
solana-test-validator
```

## Branch Strategy

### Branch Types

- **`main`**: Production-ready code, protected branch
- **`dev`**: Integration branch for features, default target for PRs
- **`feature/*`**: New features (e.g., `feature/add-multi-sig-support`)
- **`fix/*`**: Bug fixes (e.g., `fix/session-expiry-bug`)
- **`refactor/*`**: Code refactoring (e.g., `refactor/crypto-module`)
- **`docs/*`**: Documentation updates (e.g., `docs/api-reference`)
- **`test/*`**: Test improvements (e.g., `test/integration-coverage`)
- **`chore/*`**: Maintenance tasks (e.g., `chore/update-dependencies`)

### Workflow

1. **Create a feature branch from `dev`:**
   ```bash
   git checkout dev
   git pull upstream dev
   git checkout -b feature/my-new-feature
   ```

2. **Make your changes and commit:**
   ```bash
   git add .
   git commit -m "feat: Add new feature"
   ```

3. **Keep your branch up to date:**
   ```bash
   git fetch upstream
   git rebase upstream/dev
   ```

4. **Push to your fork:**
   ```bash
   git push origin feature/my-new-feature
   ```

5. **Open a Pull Request** to `dev` branch

### Branch Protection Rules

**`main` branch:**
- Requires pull request reviews (2 approvals)
- Requires status checks to pass
- Requires up-to-date branch before merging
- No force pushes allowed

**`dev` branch:**
- Requires pull request reviews (1 approval)
- Requires status checks to pass
- Allows squash merging

## Commit Guidelines

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: New feature for the user
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, missing semicolons, etc.)
- **refactor**: Code refactoring (no functional changes)
- **perf**: Performance improvements
- **test**: Adding or updating tests
- **chore**: Maintenance tasks (dependency updates, tooling, etc.)
- **ci**: CI/CD configuration changes
- **build**: Build system or external dependency changes

### Scopes

Common scopes:
- `crypto`: Cryptographic operations module
- `did`: Decentralized identity module
- `hpke`: HPKE implementation
- `handshake`: Handshake protocol
- `session`: Session management
- `contracts`: Smart contracts
- `cli`: Command-line tools
- `docs`: Documentation
- `tests`: Test infrastructure

### Examples

**Good commit messages:**

```
feat(hpke): Add support for X448 key exchange

Implement X448 key exchange algorithm as an alternative to X25519
for enhanced security. Includes unit tests and benchmarks.

Closes #123
```

```
fix(session): Prevent session expiry race condition

Session expiry cleanup could race with active encryption operations.
Added proper locking to ensure thread safety.

Fixes #456
```

```
docs(api): Update REST API documentation

Add examples for session creation and message encryption endpoints.
Include error response formats and rate limiting details.
```

**Bad commit messages:**

```
Update code         # Too vague
Fixed bug           # What bug? Where?
WIP                 # Work-in-progress commits should be squashed
```

### Commit Attribution

All commits should include proper attribution:

```bash
git commit --signoff -m "feat: Add new feature"
```

This adds:
```
Signed-off-by: Your Name <your.email@example.com>
```

## Pull Request Process

### Before Opening a PR

1. **Run all tests locally:**
   ```bash
   make test
   make test-integration
   ```

2. **Run linters:**
   ```bash
   make lint
   ```

3. **Format code:**
   ```bash
   make fmt
   ```

4. **Update documentation** if needed

5. **Add tests** for new functionality

### PR Title Format

Follow the same format as commit messages:

```
feat(crypto): Add Ed448 signature support
fix(session): Resolve session cleanup deadlock
docs: Update architecture documentation
```

### PR Description Template

```markdown
## Summary

Brief description of what this PR does.

## Motivation

Why is this change needed? What problem does it solve?

## Changes

- Change 1: Description
- Change 2: Description
- Change 3: Description

## Testing

Describe how you tested these changes:

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed
- [ ] Benchmark results (if performance-related)

## Checklist

- [ ] Code follows project style guidelines
- [ ] Tests pass locally
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (for significant changes)
- [ ] No sensitive information in code/comments

## Related Issues

Closes #123
Relates to #456
```

### PR Labels

Apply appropriate labels:
- `bug`: Bug fix
- `enhancement`: New feature or improvement
- `documentation`: Documentation updates
- `good first issue`: Suitable for first-time contributors
- `help wanted`: Looking for community input
- `priority: high`: High priority issue
- `breaking change`: Introduces breaking changes

## Code Style

### Go Code Style

Follow official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

**Key guidelines:**

1. **Use `gofmt`** for formatting (automated via pre-commit hooks)

2. **Error handling:**
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to process: %w", err)
   }

   // Bad
   if err != nil {
       panic(err)  // Don't panic in library code
   }
   ```

3. **Type assertions:**
   ```go
   // Good - use comma-ok idiom
   pub, ok := meta.PublicKey.(ed25519.PublicKey)
   if !ok {
       return fmt.Errorf("expected ed25519.PublicKey, got %T", meta.PublicKey)
   }

   // Bad - direct assertion can panic
   pub := meta.PublicKey.(ed25519.PublicKey)
   ```

4. **Naming conventions:**
   - Packages: lowercase, single word (e.g., `crypto`, `session`)
   - Interfaces: noun or adjective ending in `-er` (e.g., `Reader`, `Writer`)
   - Exported functions: PascalCase
   - Unexported functions: camelCase

5. **Comments:**
   - Exported functions must have doc comments
   - Comments should explain *why*, not *what*
   - Use complete sentences

### Solidity Code Style

Follow [Solidity Style Guide](https://docs.soliditylang.org/en/latest/style-guide.html).

**Key guidelines:**

1. **Use `solhint`** for linting

2. **Order of elements:**
   ```solidity
   // 1. Pragmas
   pragma solidity ^0.8.19;

   // 2. Imports
   import "./interfaces/IRegistry.sol";

   // 3. Interfaces, Libraries, Contracts

   contract SageRegistry {
       // 4. Type declarations
       using SafeMath for uint256;

       // 5. State variables
       mapping(address => Agent) public agents;

       // 6. Events
       event AgentRegistered(address indexed agent);

       // 7. Modifiers
       modifier onlyOwner() { ... }

       // 8. Constructor

       // 9. External functions

       // 10. Public functions

       // 11. Internal functions

       // 12. Private functions
   }
   ```

3. **Gas optimization:**
   - Use `immutable` for constants set in constructor
   - Pack struct variables efficiently
   - Use `calldata` instead of `memory` for read-only function parameters

### Testing Code Style

1. **Test naming:**
   ```go
   func TestSessionManager_CreateSession(t *testing.T) { ... }
   func TestSessionManager_CreateSession_InvalidKey(t *testing.T) { ... }
   ```

2. **Table-driven tests:**
   ```go
   func TestSignature(t *testing.T) {
       tests := []struct {
           name    string
           input   []byte
           want    []byte
           wantErr bool
       }{
           {name: "valid signature", input: data, want: expected, wantErr: false},
           // ...
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Test logic
           })
       }
   }
   ```

## Testing Requirements

### Test Coverage

- **Minimum coverage**: 70% for all packages
- **Critical paths**: 90%+ coverage (crypto, session, handshake)
- **New features**: Must include tests

### Test Categories

1. **Unit Tests** (`*_test.go`)
   - Test individual functions and methods
   - Use table-driven tests for multiple scenarios
   - Mock external dependencies

2. **Integration Tests** (`tests/integration/`)
   - Test component interactions
   - Use test containers for dependencies (Redis, blockchain nodes)
   - Verify end-to-end workflows

3. **Benchmark Tests** (`*_bench_test.go`)
   - Measure performance of critical operations
   - Use `testing.B` framework
   - Include in PR if performance-related changes

### Running Tests

```bash
# All tests
make test

# Specific package
go test ./pkg/agent/crypto/...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Integration tests
make test-integration

# Benchmarks
make bench
```

## Documentation

### Code Documentation

1. **Package documentation** (at top of any file in the package):
   ```go
   // Package crypto provides cryptographic operations for SAGE.
   //
   // Supported algorithms:
   //   - Ed25519 for signing
   //   - X25519 for key exchange
   //   - ChaCha20-Poly1305 for AEAD encryption
   package crypto
   ```

2. **Function documentation**:
   ```go
   // CreateSession establishes a new encrypted session with the given parameters.
   //
   // The session ID must be unique. The shared secret is used to derive session
   // keys via HKDF. Returns an error if the session ID already exists or if
   // key derivation fails.
   //
   // Example:
   //   sess, err := mgr.CreateSession("sess-123", sharedSecret)
   //   if err != nil {
   //       log.Fatal(err)
   //   }
   func (m *Manager) CreateSession(sessionID string, sharedSecret []byte) (*Session, error)
   ```

### External Documentation

Update relevant documentation files:

- `README.md`: Project overview and quick start
- `docs/ARCHITECTURE.md`: System architecture
- `docs/API.md`: API reference
- `docs/BUILD.md`: Build instructions
- `CHANGELOG.md`: Notable changes per version

## Review Process

### For Contributors

1. **Self-review** before requesting review:
   - Check for typos and formatting issues
   - Ensure tests pass and coverage is adequate
   - Verify all commits follow guidelines

2. **Respond to feedback** promptly and professionally

3. **Squash commits** if requested (to maintain clean history)

### For Reviewers

1. **Review checklist:**
   - [ ] Code follows style guidelines
   - [ ] Tests are comprehensive
   - [ ] Documentation is updated
   - [ ] No security vulnerabilities
   - [ ] Performance implications considered
   - [ ] Breaking changes documented

2. **Use GitHub review features:**
   - Comment on specific lines
   - Request changes or approve
   - Use suggestions for minor fixes

3. **Provide constructive feedback:**
   - Explain *why* changes are needed
   - Offer alternatives when possible
   - Recognize good practices

## Release Process

### Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

1. Update `CHANGELOG.md` with release notes
2. Update version in `go.mod` and relevant files
3. Create release branch: `release/v1.2.3`
4. Run full test suite and security scans
5. Merge to `main` via PR
6. Tag release: `git tag -a v1.2.3 -m "Release v1.2.3"`
7. Push tag: `git push origin v1.2.3`
8. GitHub Actions automatically builds and publishes release

## Getting Help

### Resources

- **Documentation**: [docs/](./docs/)
- **Issues**: [GitHub Issues](https://github.com/sage-x-project/sage/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sage-x-project/sage/discussions)

### Asking Questions

When asking for help:

1. **Search existing issues** first
2. **Provide context**: What are you trying to achieve?
3. **Include details**: Error messages, logs, code snippets
4. **Show what you've tried**: Demonstrate your effort to solve it

## License

By contributing to SAGE, you agree that your contributions will be licensed under:
- **Backend (Go)**: LGPL-3.0 License
- **Smart Contracts**: MIT License

See [LICENSE](LICENSE) for details.

## Acknowledgments

We appreciate all contributions, whether it's:
- Code improvements
- Bug reports
- Documentation enhancements
- Feature suggestions
- Community support

Thank you for making SAGE better!

---

**Happy Contributing! **

For questions or clarifications, please open a [GitHub Discussion](https://github.com/sage-x-project/sage/discussions).
