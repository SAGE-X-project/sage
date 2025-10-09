# SAGE CI/CD Pipeline

Comprehensive Continuous Integration and Continuous Deployment setup for SAGE.

## Overview

SAGE uses GitHub Actions for automated testing, security scanning, Docker builds, and releases. The CI/CD pipeline ensures code quality, security, and reliable deployments.

## Workflows

### 1. Test Workflow (`.github/workflows/test.yml`)

**Triggers:**
- Push to `main` or `dev` branches
- Pull requests to `main` or `dev`

**Jobs:**

#### Go Tests
- Runs on Ubuntu latest
- Matrix testing with Go 1.24
- Steps:
  - Install dependencies
  - Run unit tests
  - Run tests with race detector
  - Generate coverage reports
  - Upload to Codecov

#### Smart Contract Tests
- Runs on Ubuntu latest
- Node.js 20
- Steps:
  - Install Hardhat dependencies
  - Run contract tests
  - Generate coverage reports
  - Upload coverage artifacts

#### Linting
- golangci-lint for Go code
- go vet for static analysis
- gofmt for formatting checks
- solhint for Solidity code

#### Build Verification
- Matrix build on Linux, macOS, Windows
- Verifies binaries work on all platforms
- Uploads build artifacts

**Example:**
```bash
# Local equivalent
make test
go test -race ./...
go test -coverprofile=coverage.out ./...
```

### 2. Integration Test Workflow (`.github/workflows/integration-test.yml`)

**Triggers:**
- Push to `main` or `dev` branches
- Pull requests to `main` or `dev`
- Manual workflow dispatch

**Jobs:**

#### Integration Tests
- Automated test environment setup
- Services:
  - Ethereum local node (Hardhat)
  - Redis for session cache
- Test execution:
  - Integration tests with coverage
  - Handshake tests
  - HPKE tests
- Uploads integration coverage reports

#### End-to-End Tests
- Full Docker-based test environment
- Uses automated setup scripts:
  - `tools/scripts/setup_test_env.sh`
  - `tools/scripts/cleanup_test_env.sh`
- Tests complete user workflows
- Uploads logs on failure

**Example:**
```bash
# Local equivalent
./tools/scripts/setup_test_env.sh
make test-integration
make test-handshake
make test-hpke
./tools/scripts/cleanup_test_env.sh -v
```

### 3. Docker Workflow (`.github/workflows/docker.yml`)

**Triggers:**
- Push to `main` or `dev`
- Version tags (`v*`)
- Pull requests to `main`

**Jobs:**

#### Build and Push
- Multi-architecture builds (amd64, arm64)
- QEMU for cross-platform
- Docker Buildx for efficient builds
- Pushes to GitHub Container Registry (ghcr.io)
- Image tagging strategy:
  - `latest` for main branch
  - `dev` for dev branch
  - Semver tags for releases (`v1.0.0`, `v1.0`, `v1`)
  - SHA-based tags for traceability

#### Security Scan
- Trivy vulnerability scanner
- Uploads results to GitHub Security
- Fails on high/critical vulnerabilities

**Example:**
```bash
# Local equivalent
./scripts/docker-build.sh
docker scout quickview
```

### 4. Security Workflow (`.github/workflows/security.yml`)

**Triggers:**
- Push to `main` or `dev`
- Pull requests
- Weekly schedule (Monday 00:00 UTC)

**Jobs:**

#### CodeQL Analysis
- Static analysis for Go and JavaScript
- Detects security vulnerabilities
- Uploads to GitHub Security tab

#### Dependency Review
- Reviews dependencies in PRs
- Fails on high-severity vulnerabilities
- Checks for license compliance

#### Gosec
- Go security scanner
- SARIF format for GitHub integration
- Checks for common security issues:
  - SQL injection
  - Command injection
  - Weak crypto
  - etc.

#### Slither
- Smart contract static analysis
- Detects:
  - Reentrancy
  - Integer overflow/underflow
  - Access control issues
  - Gas optimization opportunities

#### GitLeaks
- Scans for secrets in code
- Prevents credential leaks
- Checks all commits

#### License Compliance
- Verifies all dependencies have compatible licenses
- Fails on GPL/AGPL (incompatible with MIT/LGPL-3.0)

**Example:**
```bash
# Local equivalent
gosec ./...
slither contracts/ethereum/contracts/
gitleaks detect
```

### 5. Release Workflow (`.github/workflows/release.yml`)

**Triggers:**
- Version tags (`v*.*.*`)

**Jobs:**

#### Build Release
- Matrix build for all platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Creates release packages (.tar.gz, .zip)
- Generates SHA256 checksums

#### Create Release
- Downloads all build artifacts
- Consolidates checksums
- Generates release notes from git log
- Creates GitHub Release
- Attaches all binaries and checksums

#### Docker Release
- Multi-arch Docker image
- Pushes to ghcr.io with version tags
- Tagged as `latest` and specific versions

**Example:**
```bash
# Create a release
git tag v1.0.0
git push origin v1.0.0

# Verify release
curl -LO https://github.com/sage-x-project/sage/releases/download/v1.0.0/sage-v1.0.0-linux-amd64.tar.gz
sha256sum -c SHA256SUMS
```

### 6. Dependabot (`.github/dependabot.yml`)

**Automated Dependency Updates:**

- **Go modules**: Weekly updates on Monday
- **GitHub Actions**: Weekly updates
- **npm packages**: Weekly updates for contracts
- **Docker base images**: Weekly updates

**Configuration:**
- Max 10 PRs for Go/npm
- Max 5 PRs for Actions/Docker
- Auto-labels: `dependencies`, `go`, `ci`, `contracts`, `docker`
- Commit message prefixes: `chore(deps)`, `chore(ci)`, `chore(docker)`

## Secrets Configuration

Required GitHub secrets:

### GitHub Container Registry
- `GITHUB_TOKEN`: Auto-provided by GitHub Actions (no setup needed)

### Codecov (Optional)
- `CODECOV_TOKEN`: For uploading coverage reports
  - Get from https://codecov.io
  - Add to repository secrets

### External Registries (Optional)
If using Docker Hub or other registries:
- `DOCKER_USERNAME`
- `DOCKER_PASSWORD`

## Status Badges

Add to README.md:

```markdown
[![Test](https://github.com/sage-x-project/sage/actions/workflows/test.yml/badge.svg)](https://github.com/sage-x-project/sage/actions/workflows/test.yml)
[![Docker](https://github.com/sage-x-project/sage/actions/workflows/docker.yml/badge.svg)](https://github.com/sage-x-project/sage/actions/workflows/docker.yml)
[![Security](https://github.com/sage-x-project/sage/actions/workflows/security.yml/badge.svg)](https://github.com/sage-x-project/sage/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/sage-x-project/sage/branch/main/graph/badge.svg)](https://codecov.io/gh/sage-x-project/sage)
```

## Local Development

### Running Tests Locally

```bash
# Go tests
make test

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# With race detector
go test -race ./...

# Contract tests
cd contracts/ethereum
npm test
npm run coverage
```

### Security Scanning Locally

```bash
# Install tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
npm install -g @crytic/slither
brew install gitleaks  # or apt/yum

# Run scans
gosec ./...
slither contracts/ethereum/contracts/
gitleaks detect
```

### Docker Build Locally

```bash
# Build single platform
docker build -t sage-backend:local .

# Build multi-platform (requires buildx)
./scripts/docker-build.sh

# Test image
docker run --rm sage-backend:local sage-crypto help
```

## Workflow Best Practices

### Pull Request Process

1. Create feature branch from `dev`
2. Make changes and commit
3. Push to remote
4. Open PR to `dev`
5. CI runs automatically:
   - Tests must pass
   - Security scans must pass
   - Code coverage should not decrease
   - Docker build must succeed
6. Request review
7. Merge after approval

### Release Process

1. Ensure `dev` branch is stable
2. Merge `dev` into `main`
3. Create and push version tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
4. GitHub Actions automatically:
   - Builds binaries for all platforms
   - Creates GitHub Release
   - Builds and pushes Docker images
5. Verify release on GitHub Releases page

### Hotfix Process

1. Create branch from `main`: `hotfix/critical-bug`
2. Fix the issue
3. Open PR to `main`
4. After merge, create patch version tag: `v1.0.1`
5. Cherry-pick fix to `dev` branch

## Troubleshooting

### Test Failures

**Go test failures:**
```bash
# Run specific test
go test -v -run TestFunctionName ./path/to/package

# Enable verbose logging
go test -v ./...

# Check for race conditions
go test -race ./...
```

**Contract test failures:**
```bash
cd contracts/ethereum
npm test -- --grep "test name"
npx hardhat test --verbose
```

### Docker Build Failures

**Check Docker buildx:**
```bash
docker buildx ls
docker buildx create --use
```

**Clear Docker cache:**
```bash
docker builder prune
docker buildx prune
```

**Build with verbose output:**
```bash
docker build --progress=plain -t sage-backend:debug .
```

### Security Scan Failures

**Gosec false positives:**
Add `#nosec` comment with justification:
```go
password := "hardcoded" // #nosec G101 -- test credential only
```

**Slither false positives:**
Add to `slither.config.json`:
```json
{
  "filter_paths": "test|mock",
  "exclude_dependencies": true,
  "exclude_informational": false
}
```

**Dependency vulnerabilities:**
```bash
# Update Go dependencies
go get -u ./...
go mod tidy

# Update npm dependencies
cd contracts/ethereum
npm audit fix
```

## Optimization Tips

### Faster CI Runs

1. **Use caching:**
   - Go modules cached automatically
   - npm packages cached
   - Docker layers cached with GitHub Actions cache

2. **Parallel jobs:**
   - Tests run in parallel by default
   - Use matrix strategy for multi-platform builds

3. **Conditional workflows:**
   - Skip Docker builds on documentation changes:
     ```yaml
     on:
       push:
         paths-ignore:
           - 'docs/**'
           - '**.md'
     ```

### Cost Optimization

1. **Artifact retention:**
   - Build artifacts: 7 days
   - Coverage reports: 30 days
   - Release artifacts: Permanent

2. **Limit concurrent jobs:**
   ```yaml
   concurrency:
     group: ${{ github.workflow }}-${{ github.ref }}
     cancel-in-progress: true
   ```

## Monitoring

### GitHub Actions Dashboard

1. Go to repository → Actions tab
2. View workflow runs
3. Check logs for failures
4. Download artifacts

### Security Dashboard

1. Go to repository → Security tab
2. View CodeQL alerts
3. Check Dependabot alerts
4. Review secret scanning results

### Docker Registry

1. Go to repository → Packages
2. View published Docker images
3. Check vulnerability scans
4. Manage package versions

## Future Enhancements

- [ ] Automated performance benchmarking
- [ ] Canary deployments
- [ ] Kubernetes deployment automation
- [ ] Integration with cloud platforms (AWS, GCP, Azure)
- [ ] Automated changelog generation
- [ ] Slack/Discord notifications for releases
- [ ] Nightly builds from `dev` branch
- [ ] Cross-repository dependency updates

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [Codecov Documentation](https://docs.codecov.com/)
- [Dependabot Documentation](https://docs.github.com/en/code-security/dependabot)
- [Main Documentation](../README.md)
- [Docker Deployment](../docker/README.md)
