# SAGE Installation and Build Instructions

This document provides detailed instructions for building, installing, and modifying SAGE (Secure Agent Guarantee Engine) in compliance with LGPL-3.0 license requirements.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Building from Source](#building-from-source)
- [Installation](#installation)
- [Modification and Relinking](#modification-and-relinking)
- [LGPL-3.0 Compliance](#lgpl-30-compliance)

## Prerequisites

### Required Tools

- **Go**: Version 1.21 or higher
  - Download: https://golang.org/dl/
  - Verify installation: `go version`

- **Git**: For cloning the repository
  - Download: https://git-scm.com/downloads
  - Verify installation: `git --version`

- **Node.js**: Version 18+ (for smart contract development)
  - Download: https://nodejs.org/
  - Verify installation: `node --version`

- **Make** (optional but recommended)
  - Linux/macOS: Usually pre-installed
  - Windows: Install via MinGW or use WSL

### System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Memory**: Minimum 4GB RAM
- **Disk Space**: At least 2GB free space
- **Network**: Internet connection for downloading dependencies

## Building from Source

### 1. Clone the Repository

```bash
git clone https://github.com/sage-x-project/sage.git
cd sage
```

### 2. Download Dependencies

```bash
# Download Go dependencies
go mod download

# Verify dependencies
go mod verify
```

### 3. Build Go Binaries

#### Build All CLI Tools

```bash
# Create build directory
mkdir -p build/bin

# Build sage-crypto CLI
go build -o build/bin/sage-crypto ./cmd/sage-crypto

# Build sage-did CLI
go build -o build/bin/sage-did ./cmd/sage-did
```

#### Build with Specific Options

```bash
# Build with optimization and without debug symbols
go build -ldflags="-s -w" -o build/bin/sage-crypto ./cmd/sage-crypto

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o build/bin/sage-crypto-linux ./cmd/sage-crypto
```

#### Build Information

```bash
# Include version information
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
  -o build/bin/sage-crypto ./cmd/sage-crypto
```

### 4. Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 5. Build Smart Contracts (Optional)

```bash
cd contracts/ethereum

# Install Node.js dependencies
npm install

# Compile contracts
npm run compile

# Run contract tests
npm test
```

## Installation

### Install Binaries

#### Option 1: Install to System Path

```bash
# Linux/macOS - Install to /usr/local/bin
sudo cp build/bin/sage-crypto /usr/local/bin/
sudo cp build/bin/sage-did /usr/local/bin/

# Verify installation
sage-crypto --version
sage-did --version
```

#### Option 2: Install to User Directory

```bash
# Create user bin directory
mkdir -p ~/bin

# Copy binaries
cp build/bin/sage-crypto ~/bin/
cp build/bin/sage-did ~/bin/

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/bin:$PATH"
```

#### Option 3: Use Go Install

```bash
# Install directly with Go
go install ./cmd/sage-crypto
go install ./cmd/sage-did

# Binaries will be in $GOPATH/bin
```

### Verify Installation

```bash
# Check versions
sage-crypto --version
sage-did --version

# Run help commands
sage-crypto --help
sage-did --help
```

## Modification and Relinking

This section describes how to modify SAGE and relink it with your application, as required by LGPL-3.0.

### Modifying SAGE Source Code

#### 1. Fork and Clone

```bash
# Fork the repository on GitHub
# Then clone your fork
git clone https://github.com/YOUR_USERNAME/sage.git
cd sage
```

#### 2. Create a Feature Branch

```bash
git checkout -b my-feature
```

#### 3. Make Your Changes

Edit the source files as needed. All `.go` files must retain the LGPL-3.0 header:

```go
// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// SPDX-License-Identifier: LGPL-3.0-or-later
```

#### 4. Test Your Changes

```bash
# Run tests
go test ./...

# Build to verify
go build ./...
```

#### 5. Document Modifications

Create a `MODIFICATIONS.md` file documenting your changes:

```markdown
# Modifications to SAGE

## Date: 2025-XX-XX
## Modified by: Your Name/Organization

### Changes Made:
- Description of modification 1
- Description of modification 2

### Files Modified:
- path/to/file1.go
- path/to/file2.go
```

### Relinking Modified SAGE

#### Using SAGE as a Library

If you're using SAGE as a Go library in your application:

```bash
# In your application's directory

# Use your modified version
go mod edit -replace github.com/sage-x-project/sage=../path/to/your/modified/sage

# Download dependencies
go mod download

# Build your application
go build ./...
```

#### Example go.mod with Modified SAGE

```go
module yourapp

go 1.21

require (
    github.com/sage-x-project/sage v0.1.0
)

// Point to your modified version
replace github.com/sage-x-project/sage => ../your-modified-sage
```

#### Rebuild Process

```bash
# Clean build cache
go clean -cache

# Download dependencies
go mod download

# Verify modules
go mod verify

# Build your application with modified SAGE
go build -o myapp ./...
```

## LGPL-3.0 Compliance

### Source Code Provision

When distributing SAGE (modified or unmodified), you must:

1. **Provide Complete Source Code**
   - Include all `.go` source files
   - Include `go.mod` and `go.sum`
   - Include build scripts and this `INSTALL.md`

2. **Provide Build Instructions**
   - This `INSTALL.md` file satisfies this requirement
   - Users must be able to rebuild SAGE from source

3. **Maintain License Notices**
   - Keep all LGPL-3.0 headers in source files
   - Include `LICENSE`, `NOTICE`, and this `INSTALL.md`

### Installation Information

The following information enables users to install modified versions:

#### Tool Chain Information

- **Compiler**: Go compiler version 1.21+
- **Build System**: Go native build system (`go build`)
- **Package Manager**: Go modules (`go mod`)

#### Build Environment

```bash
# Minimal environment variables (if needed)
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

# Build flags (example)
export CGO_ENABLED=0  # For static linking (optional)
```

#### Complete Build Commands

```bash
# From scratch
git clone https://github.com/sage-x-project/sage.git
cd sage
go mod download
go build -o build/bin/sage-crypto ./cmd/sage-crypto
go build -o build/bin/sage-did ./cmd/sage-did
```

### For Application Developers Using SAGE

If you use SAGE as a library in your application:

-  **You CAN**: Use SAGE in proprietary applications
-  **You CAN**: Distribute your application under any license
-  **You MUST**: Provide SAGE source code (if modified)
-  **You MUST**: Allow users to replace SAGE library
-  **You MUST**: Include SAGE's LICENSE and NOTICE files

#### Recommended Distribution Method

```
your-application/
├── your-app-binary
├── README.md
├── YOUR_LICENSE          # Your application license
├── third-party/
│   └── sage/
│       ├── LICENSE       # SAGE's LGPL-3.0 license
│       ├── NOTICE        # SAGE's third-party notices
│       ├── INSTALL.md    # This file
│       └── source/       # SAGE source code (if modified)
```

## Obtaining Source Code

### Official Releases

- **GitHub Releases**: https://github.com/sage-x-project/sage/releases
- **Source Code**: Download source tarball from releases

### Clone Repository

```bash
# Clone specific version
git clone --branch v0.1.0 https://github.com/sage-x-project/sage.git

# Clone latest
git clone https://github.com/sage-x-project/sage.git
```

## Support

- **Issues**: https://github.com/sage-x-project/sage/issues
- **Discussions**: https://github.com/sage-x-project/sage/discussions
- **Documentation**: See README.md and project documentation

## License

SAGE is licensed under LGPL-3.0-or-later. See LICENSE file for details.

For questions about licensing and compliance, please open an issue on GitHub.
