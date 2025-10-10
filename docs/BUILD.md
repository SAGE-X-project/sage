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

# Build libraries for all platforms
make build-lib-all

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

## Building Libraries

SAGE can be built as both static (`.a`) and shared (`.so`/`.dylib`/`.dll`) libraries for use in other applications.

### Build for Current Platform

```bash
# Build both static and shared libraries
make build-lib

# Build static library only
make build-lib-static

# Build shared library only
make build-lib-shared
```

**Output (Linux):**
```
build/lib/
├── libsage.a      # Static library
├── libsage.h      # C header file
└── libsage.so     # Shared library
```

**Output (macOS):**
```
build/lib/
├── libsage.a      # Static library
├── libsage.h      # C header file
└── libsage.dylib  # Shared library
```

### Build for All Platforms

**Important:** Cross-platform library builds require platform-specific C toolchains. The `build-lib-all` target attempts to build for all platforms, but will likely only succeed for the current platform when run locally.

For reliable cross-platform library builds, use one of these approaches:
1. **Native builds:** Build on each target platform
2. **Docker:** Use Docker containers with appropriate toolchains (recommended for CI/CD)
3. **Cross-compilation toolchains:** Install platform-specific cross-compilers (advanced)

```bash
# Attempt to build all libraries (static)
# Note: May only succeed for current platform
make build-lib-all

# Build for specific platform (requires cross-compiler)
make build-lib-linux-amd64
make build-lib-darwin-arm64
make build-lib-windows-amd64
```

**Expected output (when cross-compilation is properly configured):**
```
build/lib/
├── linux-amd64/
│   ├── libsage.a
│   └── libsage.h
├── linux-arm64/
│   ├── libsage.a
│   └── libsage.h
├── darwin-amd64/
│   ├── libsage.a
│   └── libsage.h
├── darwin-arm64/
│   ├── libsage.a
│   └── libsage.h
└── windows-amd64/
    ├── libsage.a
    └── libsage.h
```

### Build Shared Libraries

```bash
# Linux x86_64
make build-lib-linux-amd64-shared

# Linux ARM64
make build-lib-linux-arm64-shared

# macOS Intel
make build-lib-darwin-amd64-shared

# macOS Apple Silicon
make build-lib-darwin-arm64-shared

# Windows x86_64 (requires MinGW)
make build-lib-windows-amd64-shared
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

## Library Usage Examples

### C/C++ Integration

**Header file (`libsage.h`):**
```c
#ifndef LIBSAGE_H
#define LIBSAGE_H

// Initialize SAGE library
int sage_init();

// Generate Ed25519 key pair
int sage_generate_keypair(char* public_key, char* private_key);

// Sign message
int sage_sign(const char* private_key, const char* message, char* signature);

// Verify signature
int sage_verify(const char* public_key, const char* message, const char* signature);

// Cleanup
void sage_cleanup();

#endif
```

**C example:**
```c
#include <stdio.h>
#include "libsage.h"

int main() {
    if (sage_init() != 0) {
        fprintf(stderr, "Failed to initialize SAGE\n");
        return 1;
    }

    char public_key[128];
    char private_key[128];

    if (sage_generate_keypair(public_key, private_key) == 0) {
        printf("Public Key: %s\n", public_key);
        printf("Private Key: %s\n", private_key);
    }

    sage_cleanup();
    return 0;
}
```

**Compile:**
```bash
# Linux
gcc -o example example.c -L build/lib/linux-amd64 -lsage -static

# macOS
clang -o example example.c build/lib/darwin-arm64/libsage.a

# Windows (MinGW)
x86_64-w64-mingw32-gcc -o example.exe example.c build/lib/windows-amd64/libsage.a
```

---

### Python Integration (ctypes)

```python
import ctypes
import os

# Load library
if os.name == 'nt':
    lib = ctypes.CDLL('build/lib/windows-amd64/libsage.dll')
elif os.uname().sysname == 'Darwin':
    lib = ctypes.CDLL('build/lib/darwin-arm64/libsage.dylib')
else:
    lib = ctypes.CDLL('build/lib/linux-amd64/libsage.so')

# Initialize
lib.sage_init()

# Generate key pair
public_key = ctypes.create_string_buffer(128)
private_key = ctypes.create_string_buffer(128)
lib.sage_generate_keypair(public_key, private_key)

print(f"Public Key: {public_key.value.decode()}")
print(f"Private Key: {private_key.value.decode()}")

# Cleanup
lib.sage_cleanup()
```

---

### Rust Integration (FFI)

```rust
// build.rs
fn main() {
    println!("cargo:rustc-link-search=native=build/lib/linux-amd64");
    println!("cargo:rustc-link-lib=static=sage");
}
```

```rust
// src/main.rs
use std::ffi::{CString, CStr};
use std::os::raw::c_char;

extern "C" {
    fn sage_init() -> i32;
    fn sage_generate_keypair(public_key: *mut c_char, private_key: *mut c_char) -> i32;
    fn sage_cleanup();
}

fn main() {
    unsafe {
        if sage_init() != 0 {
            eprintln!("Failed to initialize SAGE");
            return;
        }

        let mut public_key = vec![0u8; 128];
        let mut private_key = vec![0u8; 128];

        sage_generate_keypair(
            public_key.as_mut_ptr() as *mut c_char,
            private_key.as_mut_ptr() as *mut c_char,
        );

        println!("Public Key: {:?}", CStr::from_ptr(public_key.as_ptr() as *const c_char));

        sage_cleanup();
    }
}
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
        run: make build-lib-all

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
