# SAGE Build Guide

This guide covers building SAGE binaries and libraries for multiple platforms and architectures.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Prerequisites](#prerequisites)
3. [Building Binaries](#building-binaries)
4. [Building Libraries](#building-libraries)
5. [Cross-Platform Builds](#cross-platform-builds)
6. [Release Builds](#release-builds)
7. [Platform-Specific Notes](#platform-specific-notes)
8. [Troubleshooting](#troubleshooting)

---

## Quick Start

```bash
# Build for current platform
make build

# Build for all platforms
make build-all-platforms


# Create full release (binaries + libraries + packages)
make release
```

---

## Prerequisites

### Required

- **Go 1.23.0+** - [Download](https://golang.org/dl/)
- **Make** - Standard on Linux/macOS, install via [Chocolatey](https://chocolatey.org/) on Windows
- **Git** - For version information

### Optional (for cross-compilation)

**Note:** Cross-platform library compilation requires platform-specific C toolchains and is complex to set up. For production use, we recommend:
1. Building libraries natively on each target platform, or
2. Using Docker containers for cross-platform builds (see CI/CD section)

For binary cross-compilation (which works without additional toolchains):
- **MinGW-w64** - For Windows binary compilation on Linux/macOS
  ```bash
  # Ubuntu/Debian
  sudo apt-get install mingw-w64

  # macOS
  brew install mingw-w64
  ```

---

## Building Binaries

### Build for Current Platform

```bash
# Build all binaries
make build

# Build specific binary
make build-crypto      # sage-crypto CLI
make build-did         # sage-did CLI
make build-verify      # sage-verify CLI
```

**Output:**
- Binaries: `build/bin/`
- Examples: `build/bin/`

### Build Specific Example

```bash
make build-example-basic-demo
make build-example-basic-tool
make build-example-client
```

---

## Cross-Platform Builds

### Build for All Platforms

```bash
# Build binaries for all platforms
make build-all-platforms
```

**Output:**
```
build/dist/
├── linux-amd64/
│   ├── sage-crypto
│   ├── sage-did
│   └── sage-verify
├── linux-arm64/
│   ├── sage-crypto
│   ├── sage-did
│   └── sage-verify
├── darwin-amd64/
│   ├── sage-crypto
│   ├── sage-did
│   └── sage-verify
├── darwin-arm64/
│   ├── sage-crypto
│   ├── sage-did
│   └── sage-verify
├── windows-amd64/
│   ├── sage-crypto.exe
│   ├── sage-did.exe
│   └── sage-verify.exe
└── windows-arm64/
    ├── sage-crypto.exe
    ├── sage-did.exe
    └── sage-verify.exe
```

### Build for Specific Platform

```bash
# Build for specific OS and architecture
make build-platform GOOS=linux GOARCH=amd64
make build-platform GOOS=darwin GOARCH=arm64
make build-platform GOOS=windows GOARCH=amd64
```

**Supported Platforms:**
| OS | Architecture | GOOS | GOARCH |
|----|-------------|------|--------|
| Linux | x86_64 | linux | amd64 |
| Linux | ARM64 | linux | arm64 |
| macOS | Intel | darwin | amd64 |
| macOS | Apple Silicon | darwin | arm64 |
| Windows | x86_64 | windows | amd64 |
| Windows | ARM64 | windows | arm64 |

---

## Release Builds

### Full Release Build

Creates binaries, libraries, and packages for all platforms with checksums.

```bash
make release
```

**What it does:**
1. Cleans all build artifacts
2. Builds binaries for all platforms
3. Builds libraries for all platforms
4. Creates `.tar.gz` packages for each platform
5. Generates SHA256 checksums

**Output:**
```
build/
├── dist/
│   ├── linux-amd64/
│   ├── linux-arm64/
│   ├── darwin-amd64/
│   ├── darwin-arm64/
│   ├── windows-amd64/
│   └── packages/
│       ├── sage-linux-amd64.tar.gz
│       ├── sage-linux-arm64.tar.gz
│       ├── sage-darwin-amd64.tar.gz
│       ├── sage-darwin-arm64.tar.gz
│       ├── sage-windows-amd64.tar.gz
│       └── SHA256SUMS
└── lib/
    ├── linux-amd64/
    ├── linux-arm64/
    ├── darwin-amd64/
    ├── darwin-arm64/
    └── windows-amd64/
```

### Create Packages Only

```bash
# Build and package (no clean)
make package
```

### Generate Checksums

```bash
make checksums
```

**Output:**
```
build/dist/packages/SHA256SUMS
```

---

## Platform-Specific Notes

### Linux

**Static Library:**
```bash
make build-lib-linux-amd64
```

**Shared Library:**
```bash
make build-lib-linux-amd64-shared
```

**Usage:**
```c
// Compile with static library
gcc -o myapp myapp.c build/lib/linux-amd64/libsage.a

// Compile with shared library
gcc -o myapp myapp.c -L build/lib/linux-amd64 -lsage
export LD_LIBRARY_PATH=build/lib/linux-amd64:$LD_LIBRARY_PATH
./myapp
```

---

### macOS

**Universal Binary (Intel + Apple Silicon):**
```bash
# Build for both architectures
make build-lib-darwin-amd64
make build-lib-darwin-arm64

# Create universal binary with lipo
lipo -create \
  build/lib/darwin-amd64/libsage.a \
  build/lib/darwin-arm64/libsage.a \
  -output build/lib/libsage-universal.a
```

**Shared Library (.dylib):**
```bash
make build-lib-darwin-arm64-shared
```

**Usage:**
```c
// Compile with static library
clang -o myapp myapp.c build/lib/darwin-arm64/libsage.a

// Compile with shared library
clang -o myapp myapp.c -L build/lib/darwin-arm64 -lsage
export DYLD_LIBRARY_PATH=build/lib/darwin-arm64:$DYLD_LIBRARY_PATH
./myapp
```

**Code Signing (macOS):**
```bash
# Sign the binary
codesign -s "Developer ID Application" build/bin/sage-crypto

# Verify signature
codesign -v build/bin/sage-crypto

# Check entitlements
codesign -d --entitlements - build/bin/sage-crypto
```

---

### Windows

**Static Library:**
```bash
make build-lib-windows-amd64
```

**DLL (requires MinGW):**
```bash
# On Linux/macOS with MinGW installed
make build-lib-windows-amd64-shared
```

**Usage (MSVC):**
```cmd
# Compile with static library
cl.exe /I build\lib\windows-amd64 myapp.c build\lib\windows-amd64\libsage.a

# Compile with DLL
cl.exe /I build\lib\windows-amd64 myapp.c /link build\lib\windows-amd64\libsage.lib
copy build\lib\windows-amd64\libsage.dll .
myapp.exe
```

**Usage (MinGW):**
```bash
# Compile with static library
x86_64-w64-mingw32-gcc -o myapp.exe myapp.c build/lib/windows-amd64/libsage.a

# Compile with DLL
x86_64-w64-mingw32-gcc -o myapp.exe myapp.c -L build/lib/windows-amd64 -lsage
```

---

## Build Options

### Version Information

Version, commit hash, and build time are automatically embedded:

```bash
# Uses git tags
make build

# Or set manually
VERSION=1.0.0 COMMIT=abc123 make build
```

**Check version:**
```bash
./build/bin/sage-crypto --version
# Output: sage-crypto v1.0.0 (abc123) built at 2025-10-08_12:34:56
```

### Build Flags

```bash
# Custom LDFLAGS
LDFLAGS="-w -s -X main.CustomVar=value" make build

# Disable optimizations (debugging)
LDFLAGS="" GOFLAGS="" make build

# Enable race detector
GOFLAGS="-race" make build
```

### CGO

```bash
# Enable CGO (required for Windows DLL)
CGO_ENABLED=1 make build-lib-windows-amd64-shared

# Disable CGO (static builds)
CGO_ENABLED=0 make build
```

---

## Troubleshooting

### Error: "go: cannot find main module"

**Solution:**
```bash
# Ensure you're in the project root
cd /path/to/sage

# Verify go.mod exists
ls go.mod
```

---

### Error: "undefined reference to \_\_stack\_chk\_fail"

**Solution (Linux):**
```bash
# Use musl for static builds
CGO_ENABLED=0 make build
```

---

### Error: "x86\_64-w64-mingw32-gcc: command not found"

**Solution:**
```bash
# Install MinGW
# Ubuntu/Debian
sudo apt-get install mingw-w64

# macOS
brew install mingw-w64
```

---

### Windows: "cannot execute binary file"

**Cause:** Trying to run Windows binary on Linux/macOS

**Solution:**
```bash
# Use Wine
sudo apt-get install wine64
wine64 build/dist/windows-amd64/sage-crypto.exe --version
```

---

### macOS: "cannot be opened because the developer cannot be verified"

**Solution:**
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine build/bin/sage-crypto

# Or sign the binary
codesign -s - build/bin/sage-crypto
```

---

### Build is slow

**Solutions:**
```bash
# Use build cache
export GOCACHE=$(go env GOCACHE)

# Parallel builds
make -j$(nproc) build

# Disable verbose output
GOFLAGS="" make build
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23.0'

      - name: Build all platforms
        run: make build-all-platforms

      - name: Build libraries
        run: 
      - name: Create release
        if: startsWith(github.ref, 'refs/tags/')
        run: make release

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: sage-binaries
          path: build/dist/packages/
```

---

## Advanced Usage

### Custom Build Script

```bash
#!/bin/bash
# custom-build.sh

VERSION=$(git describe --tags --always)
PLATFORMS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"

for platform in $PLATFORMS; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}

    OUTPUT="sage-$GOOS-$GOARCH"
    [[ "$GOOS" == "windows" ]] && OUTPUT="$OUTPUT.exe"

    echo "Building $OUTPUT..."
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-w -s -X main.Version=$VERSION" \
        -o "build/$OUTPUT" \
        ./cmd/sage-crypto
done
```

---

### Docker Multi-Stage Build

```dockerfile
# Build stage
FROM golang:1.23.0-alpine AS builder
WORKDIR /build
COPY . .
RUN make build

# Runtime stage
FROM alpine:3.19
COPY --from=builder /build/build/bin/sage-crypto /usr/local/bin/
CMD ["sage-crypto"]
```

---

## Build Performance

**Typical build times (on Apple M1 Pro):**
| Target | Time |
|--------|------|
| Current platform | ~5s |
| All platforms (6) | ~30s |
| Libraries (5 platforms) | ~25s |
| Full release | ~60s |

**Optimization tips:**
```bash
# Use cached builds
go build -i

# Parallel compilation
GOMAXPROCS=8 make build

# Skip tests
make build SKIP_TESTS=1
```

---

## Version Information

**Embedded at build time:**
- `main.Version` - Git tag or "dev"
- `main.Commit` - Git commit hash
- `main.BuildTime` - UTC timestamp

**Access in code:**
```go
package main

var (
    Version   string
    Commit    string
    BuildTime string
)

func printVersion() {
    fmt.Printf("Version: %s\n", Version)
    fmt.Printf("Commit: %s\n", Commit)
    fmt.Printf("Built: %s\n", BuildTime)
}
```

---

## Support Matrix

| Platform | Binary | Static Lib | Shared Lib | Tested |
|----------|--------|------------|------------|--------|
| Linux x86_64 | Yes | Yes | Yes | Yes |
| Linux ARM64 | Yes | Yes | Yes | Yes |
| macOS Intel | Yes | Yes | Yes | Yes |
| macOS Apple Silicon | Yes | Yes | Yes | Yes |
| Windows x86_64 | Yes | Yes | Requires MinGW | Partial |
| Windows ARM64 | Yes | Yes | No | No |

Note: "Requires MinGW" indicates additional tools are needed; "Partial" indicates limited testing coverage.

---

## Getting Help

- **Documentation:** See `docs/` directory
- **Issues:** https://github.com/sage-x-project/sage/issues
- **Discussions:** https://github.com/sage-x-project/sage/discussions

---

**Last Updated:** 2025-10-08
**SAGE Version:** 1.0.0
