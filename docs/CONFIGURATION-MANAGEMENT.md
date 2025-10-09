# SAGE Configuration Management

**Date:** 2025-10-08
**Status:** Implementation
**Priority:** HIGH (Production Essential)

---

## 1. Overview

### 1.1 Goals

- Yes Environment-specific configurations (dev/staging/prod)
- Yes Configuration validation with clear error messages
- Yes Support for environment variables
- Yes Default values for all settings
- Yes Hot-reload capability (optional)
- Yes Secure handling of secrets

### 1.2 Configuration Sources (Priority Order)

```
1. Command-line flags (highest priority)
2. Environment variables
3. Configuration file (.yaml/.json)
4. Default values (lowest priority)
```

---

## 2. Configuration Structure

### 2.1 File Organization

```
config/
├── default.yaml          # Default configuration
├── development.yaml      # Development overrides
├── staging.yaml          # Staging overrides
├── production.yaml       # Production overrides
└── local.yaml           # Local overrides (gitignored)
```

### 2.2 Complete Configuration Schema

```yaml
# config/default.yaml
environment: "development"

# Server Configuration
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s
  max_connections: 1000

# Blockchain Configuration
blockchain:
  ethereum:
    enabled: true
    rpc_url: "${ETHEREUM_RPC_URL:http://localhost:8545}"
    chain_id: 1337
    gas_limit: 3000000
    gas_price: 20000000000  # 20 Gwei
    max_retries: 3
    retry_delay: 1s
    request_timeout: 30s
    contract_address: ""

  solana:
    enabled: false
    rpc_url: "${SOLANA_RPC_URL:http://localhost:8899}"
    cluster: "devnet"
    commitment: "confirmed"
    max_retries: 3
    retry_delay: 1s
    request_timeout: 30s

# DID Configuration
did:
  registry_address: ""
  method: "sage"
  network: "ethereum"
  cache_size: 100
  cache_ttl: 5m
  resolver_timeout: 10s

# Key Store Configuration
keystore:
  type: "encrypted-file"  # encrypted-file, vault, aws-kms
  directory: ".sage/keys"
  passphrase_env: "SAGE_KEYSTORE_PASSPHRASE"
  rotation_enabled: false
  rotation_interval: 90d

# Session Configuration
session:
  max_age: 1h
  idle_timeout: 10m
  max_messages: 1000
  cleanup_interval: 30s
  max_sessions: 10000

# Logging Configuration
logging:
  level: "info"  # trace, debug, info, warn, error, fatal
  format: "json"  # json, text
  output: "stdout"  # stdout, stderr, file
  file_path: "/var/log/sage/app.log"
  max_size: 100  # MB
  max_backups: 3
  max_age: 7  # days
  compress: true

# Metrics Configuration
metrics:
  enabled: true
  port: 9090
  path: "/metrics"
  namespace: "sage"

# Health Check Configuration
health:
  enabled: true
  port: 8081
  path: "/health"
  checks:
    - "blockchain"
    - "did"
    - "keystore"
    - "session"

# Security Configuration
security:
  tls_enabled: false
  tls_cert_file: ""
  tls_key_file: ""
  cors_enabled: true
  cors_origins:
    - "http://localhost:3000"
  rate_limiting:
    enabled: true
    requests_per_second: 100
    burst: 200

# Feature Flags
features:
  did_resolution: true
  session_encryption: true
  replay_protection: true
  key_rotation: false
  distributed_tracing: false
```

---

## 3. Environment-Specific Configurations

### 3.1 Development

```yaml
# config/development.yaml
environment: "development"

logging:
  level: "debug"
  format: "text"

blockchain:
  ethereum:
    rpc_url: "http://localhost:8545"
    chain_id: 1337

security:
  tls_enabled: false
  cors_origins:
    - "http://localhost:3000"
    - "http://localhost:8080"
```

### 3.2 Staging

```yaml
# config/staging.yaml
environment: "staging"

server:
  host: "0.0.0.0"
  port: 8080

logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "/var/log/sage/staging.log"

blockchain:
  ethereum:
    rpc_url: "${ETHEREUM_RPC_URL}"
    chain_id: 11155111  # Sepolia

security:
  tls_enabled: true
  tls_cert_file: "/etc/sage/certs/staging.crt"
  tls_key_file: "/etc/sage/certs/staging.key"
```

### 3.3 Production

```yaml
# config/production.yaml
environment: "production"

server:
  host: "0.0.0.0"
  port: 443
  max_connections: 10000

logging:
  level: "warn"
  format: "json"
  output: "file"
  file_path: "/var/log/sage/production.log"
  compress: true

blockchain:
  ethereum:
    rpc_url: "${ETHEREUM_RPC_URL}"
    chain_id: 1  # Mainnet
    gas_price: 0  # Use EIP-1559

session:
  max_sessions: 100000
  cleanup_interval: 1m

security:
  tls_enabled: true
  tls_cert_file: "/etc/sage/certs/production.crt"
  tls_key_file: "/etc/sage/certs/production.key"
  cors_enabled: false
  rate_limiting:
    requests_per_second: 1000
    burst: 2000

features:
  key_rotation: true
  distributed_tracing: true
```

---

## 4. Implementation

### 4.1 Enhanced Config Package

**File: `config/loader.go`**

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// LoadConfig loads configuration with environment-specific overrides
func LoadConfig() (*Config, error) {
    env := getEnvironment()

    // Load base config
    cfg, err := loadBase()
    if err != nil {
        return nil, err
    }

    // Load environment-specific config
    if err := loadEnvironment(cfg, env); err != nil {
        return nil, err
    }

    // Load local overrides (optional)
    loadLocal(cfg)

    // Apply environment variables
    if err := applyEnvVars(cfg); err != nil {
        return nil, err
    }

    // Validate configuration
    if err := Validate(cfg); err != nil {
        return nil, err
    }

    return cfg, nil
}

func getEnvironment() string {
    env := os.Getenv("SAGE_ENV")
    if env == "" {
        env = os.Getenv("ENV")
    }
    if env == "" {
        env = "development"
    }
    return env
}

func loadBase() (*Config, error) {
    return LoadFromFile("config/default.yaml")
}

func loadEnvironment(cfg *Config, env string) error {
    path := filepath.Join("config", env+".yaml")
    if _, err := os.Stat(path); os.IsNotExist(err) {
        // Environment config is optional
        return nil
    }

    envCfg, err := LoadFromFile(path)
    if err != nil {
        return err
    }

    // Merge environment config into base config
    return mergeConfig(cfg, envCfg)
}

func loadLocal(cfg *Config) error {
    path := "config/local.yaml"
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return nil
    }

    localCfg, err := LoadFromFile(path)
    if err != nil {
        return err
    }

    return mergeConfig(cfg, localCfg)
}
```

### 4.2 Environment Variable Substitution

**File: `config/env.go`**

```go
package config

import (
    "fmt"
    "os"
    "regexp"
    "strings"
)

var envVarPattern = regexp.MustCompile(`\$\{([^:}]+)(?::([^}]*))?\}`)

// applyEnvVars substitutes environment variables in config values
func applyEnvVars(cfg *Config) error {
    // Blockchain
    if cfg.Blockchain != nil {
        if cfg.Blockchain.Ethereum != nil {
            cfg.Blockchain.Ethereum.RPCURL = substituteEnv(cfg.Blockchain.Ethereum.RPCURL)
            cfg.Blockchain.Ethereum.ContractAddress = substituteEnv(cfg.Blockchain.Ethereum.ContractAddress)
        }
        if cfg.Blockchain.Solana != nil {
            cfg.Blockchain.Solana.RPCURL = substituteEnv(cfg.Blockchain.Solana.RPCURL)
        }
    }

    // DID
    if cfg.DID != nil {
        cfg.DID.RegistryAddress = substituteEnv(cfg.DID.RegistryAddress)
    }

    // KeyStore
    if cfg.KeyStore != nil {
        cfg.KeyStore.PassphraseEnv = substituteEnv(cfg.KeyStore.PassphraseEnv)
    }

    // Security
    if cfg.Security != nil {
        cfg.Security.TLSCertFile = substituteEnv(cfg.Security.TLSCertFile)
        cfg.Security.TLSKeyFile = substituteEnv(cfg.Security.TLSKeyFile)
    }

    return nil
}

// substituteEnv replaces ${VAR:default} with environment variable value
func substituteEnv(s string) string {
    return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
        parts := envVarPattern.FindStringSubmatch(match)
        if len(parts) < 2 {
            return match
        }

        varName := parts[1]
        defaultValue := ""
        if len(parts) > 2 {
            defaultValue = parts[2]
        }

        value := os.Getenv(varName)
        if value == "" {
            return defaultValue
        }
        return value
    })
}
```

### 4.3 Configuration Validation

**File: `config/validation.go`**

```go
package config

import (
    "fmt"
    "net/url"
    "os"
    "strings"
)

// Validate validates the configuration
func Validate(cfg *Config) error {
    var errors []string

    // Validate environment
    if !isValidEnvironment(cfg.Environment) {
        errors = append(errors, fmt.Sprintf("invalid environment: %s (must be development, staging, or production)", cfg.Environment))
    }

    // Validate server config
    if cfg.Server != nil {
        if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
            errors = append(errors, fmt.Sprintf("invalid server port: %d", cfg.Server.Port))
        }
    }

    // Validate blockchain config
    if cfg.Blockchain != nil {
        if cfg.Blockchain.Ethereum != nil && cfg.Blockchain.Ethereum.Enabled {
            if err := validateURL(cfg.Blockchain.Ethereum.RPCURL); err != nil {
                errors = append(errors, fmt.Sprintf("invalid Ethereum RPC URL: %v", err))
            }
            if cfg.Blockchain.Ethereum.ChainID == 0 {
                errors = append(errors, "Ethereum chain ID cannot be 0")
            }
        }

        if cfg.Blockchain.Solana != nil && cfg.Blockchain.Solana.Enabled {
            if err := validateURL(cfg.Blockchain.Solana.RPCURL); err != nil {
                errors = append(errors, fmt.Sprintf("invalid Solana RPC URL: %v", err))
            }
        }
    }

    // Validate keystore config
    if cfg.KeyStore != nil {
        if cfg.KeyStore.Type == "" {
            errors = append(errors, "keystore type is required")
        }
        if cfg.KeyStore.Directory == "" {
            errors = append(errors, "keystore directory is required")
        }
    }

    // Validate logging config
    if cfg.Logging != nil {
        validLevels := map[string]bool{"trace": true, "debug": true, "info": true, "warn": true, "error": true, "fatal": true}
        if !validLevels[cfg.Logging.Level] {
            errors = append(errors, fmt.Sprintf("invalid logging level: %s", cfg.Logging.Level))
        }

        if cfg.Logging.Output == "file" && cfg.Logging.FilePath == "" {
            errors = append(errors, "logging file path is required when output is 'file'")
        }
    }

    // Validate TLS config
    if cfg.Security != nil && cfg.Security.TLSEnabled {
        if cfg.Security.TLSCertFile == "" {
            errors = append(errors, "TLS cert file is required when TLS is enabled")
        } else if _, err := os.Stat(cfg.Security.TLSCertFile); os.IsNotExist(err) {
            errors = append(errors, fmt.Sprintf("TLS cert file not found: %s", cfg.Security.TLSCertFile))
        }

        if cfg.Security.TLSKeyFile == "" {
            errors = append(errors, "TLS key file is required when TLS is enabled")
        } else if _, err := os.Stat(cfg.Security.TLSKeyFile); os.IsNotExist(err) {
            errors = append(errors, fmt.Sprintf("TLS key file not found: %s", cfg.Security.TLSKeyFile))
        }
    }

    if len(errors) > 0 {
        return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
    }

    return nil
}

func isValidEnvironment(env string) bool {
    return env == "development" || env == "staging" || env == "production"
}

func validateURL(rawURL string) error {
    if rawURL == "" {
        return fmt.Errorf("URL is empty")
    }

    u, err := url.Parse(rawURL)
    if err != nil {
        return err
    }

    if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ws" && u.Scheme != "wss" {
        return fmt.Errorf("invalid URL scheme: %s", u.Scheme)
    }

    return nil
}
```

---

## 5. Usage Examples

### 5.1 Loading Configuration

```go
package main

import (
    "github.com/sage-x-project/sage/config"
    "log"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    log.Printf("Environment: %s", cfg.Environment)
    log.Printf("Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
}
```

### 5.2 Environment Variable Override

```bash
# Set environment
export SAGE_ENV=production

# Override blockchain RPC URL
export ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_API_KEY

# Run application
./sage-backend
```

### 5.3 Configuration File Priority

```bash
# Will load in order:
# 1. config/default.yaml
# 2. config/production.yaml (if SAGE_ENV=production)
# 3. config/local.yaml (if exists, gitignored)
# 4. Environment variables
```

---

## 6. Security Best Practices

### 6.1 Secrets Management

**DO:**
- Yes Use environment variables for secrets
- Yes Store secrets in secret management systems (Vault, AWS Secrets Manager)
- Yes Use `.gitignore` for `local.yaml` and `.env` files
- Yes Rotate secrets regularly

**DON'T:**
- No Commit secrets to version control
- No Use plain text secrets in production config files
- No Share production config files

### 6.2 Example `.gitignore`

```gitignore
# Local configuration overrides
config/local.yaml
config/local.json

# Environment files
.env
.env.local
.env.*.local

# Secrets
*.key
*.pem
*.crt
secrets/

# Keystore
.sage/keys/
```

---

## 7. Testing

### 7.1 Configuration Test

**File: `config/config_test.go`**

```go
package config_test

import (
    "os"
    "testing"

    "github.com/sage-x-project/sage/config"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
    // Set test environment
    os.Setenv("SAGE_ENV", "development")
    defer os.Unsetenv("SAGE_ENV")

    cfg, err := config.LoadConfig()
    require.NoError(t, err)
    assert.NotNil(t, cfg)
    assert.Equal(t, "development", cfg.Environment)
}

func TestEnvVarSubstitution(t *testing.T) {
    os.Setenv("TEST_RPC_URL", "http://test:8545")
    defer os.Unsetenv("TEST_RPC_URL")

    result := config.SubstituteEnv("${TEST_RPC_URL:http://default:8545}")
    assert.Equal(t, "http://test:8545", result)
}

func TestEnvVarDefault(t *testing.T) {
    result := config.SubstituteEnv("${NONEXISTENT_VAR:http://default:8545}")
    assert.Equal(t, "http://default:8545", result)
}

func TestValidation(t *testing.T) {
    cfg := &config.Config{
        Environment: "production",
        Server: &config.ServerConfig{
            Port: 443,
        },
    }

    err := config.Validate(cfg)
    assert.NoError(t, err)
}

func TestValidationFails(t *testing.T) {
    cfg := &config.Config{
        Environment: "invalid",
    }

    err := config.Validate(cfg)
    assert.Error(t, err)
}
```

---

## 8. Deployment

### 8.1 Docker

**Dockerfile:**
```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o sage-backend ./cmd/sage

FROM alpine:latest
WORKDIR /app

# Copy binary
COPY --from=builder /app/sage-backend .

# Copy default config
COPY config/default.yaml config/
COPY config/production.yaml config/

# Environment can be overridden at runtime
ENV SAGE_ENV=production

CMD ["./sage-backend"]
```

### 8.2 Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: sage-config
data:
  production.yaml: |
    environment: "production"
    server:
      port: 8080
    logging:
      level: "warn"
      format: "json"
---
apiVersion: v1
kind: Secret
metadata:
  name: sage-secrets
type: Opaque
stringData:
  ETHEREUM_RPC_URL: "https://mainnet.infura.io/v3/YOUR_KEY"
  SAGE_KEYSTORE_PASSPHRASE: "your-secret-passphrase"
```

---

## 9. Migration Guide

### 9.1 Migrating Existing Configuration

```bash
# Old way
./sage-backend --ethereum-rpc=http://localhost:8545 --log-level=debug

# New way (config file)
cat > config/local.yaml <<EOF
blockchain:
  ethereum:
    rpc_url: "http://localhost:8545"
logging:
  level: "debug"
EOF

./sage-backend

# Or with environment variables
export ETHEREUM_RPC_URL=http://localhost:8545
export SAGE_LOG_LEVEL=debug
./sage-backend
```

---

## 10. Checklist

### Implementation
- [ ] Create config structure files
- [ ] Implement environment-specific loading
- [ ] Add environment variable substitution
- [ ] Implement validation
- [ ] Add default configurations
- [ ] Create example configs for each environment

### Testing
- [ ] Unit tests for config loading
- [ ] Integration tests with different environments
- [ ] Validation tests

### Documentation
- [ ] Document all configuration options
- [ ] Create migration guide
- [ ] Add deployment examples

### Security
- [ ] Review secret handling
- [ ] Update .gitignore
- [ ] Document security best practices

---

**Status:** List READY TO IMPLEMENT
**Estimated Time:** 2-3 hours
**Priority:** HIGH
**Next Step:** Implement enhanced configuration package
