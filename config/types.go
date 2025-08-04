// Package config provides configuration management for SAGE
package config

import (
	"time"
)

// Config represents the main configuration structure
type Config struct {
	Version  string                     `yaml:"version" json:"version"`
	Networks map[string]NetworkConfig   `yaml:"networks" json:"networks"`
	KeyMgmt  KeyManagementConfig       `yaml:"key_management" json:"key_management"`
	Proxy    ProxyConfig               `yaml:"proxy" json:"proxy"`
	Registry RegistrationConfig        `yaml:"registration" json:"registration"`
	Security SecurityConfig            `yaml:"security" json:"security"`
	Logging  LoggingConfig            `yaml:"logging" json:"logging"`
}

// NetworkConfig contains network-specific configuration
type NetworkConfig struct {
	Chains map[string]ChainConfig `yaml:"chains" json:"chains"`
}

// ChainConfig contains chain-specific configuration
type ChainConfig struct {
	RPC       string `yaml:"rpc" json:"rpc"`
	Contract  string `yaml:"contract,omitempty" json:"contract,omitempty"`
	ProgramID string `yaml:"program_id,omitempty" json:"program_id,omitempty"`
	ChainID   uint64 `yaml:"chain_id,omitempty" json:"chain_id,omitempty"`
}

// KeyManagementConfig contains key management configuration
type KeyManagementConfig struct {
	DefaultAlgorithm string                     `yaml:"default_algorithm" json:"default_algorithm"`
	Storage          StorageConfig             `yaml:"storage" json:"storage"`
	Algorithms       map[string]AlgorithmConfig `yaml:"algorithms" json:"algorithms"`
}

// StorageConfig contains key storage configuration
type StorageConfig struct {
	Type       string `yaml:"type" json:"type"` // file, memory, hsm
	Path       string `yaml:"path,omitempty" json:"path,omitempty"`
	Encryption bool   `yaml:"encryption" json:"encryption"`
}

// AlgorithmConfig contains algorithm-specific configuration
type AlgorithmConfig struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	AddressFormat string `yaml:"address_format,omitempty" json:"address_format,omitempty"`
	KeySize       int    `yaml:"key_size,omitempty" json:"key_size,omitempty"`
}

// ProxyConfig contains proxy server configuration
type ProxyConfig struct {
	Enabled    bool                `yaml:"enabled" json:"enabled"`
	Endpoints  []ProxyEndpoint     `yaml:"endpoints" json:"endpoints"`
	GasPolicy  GasPolicyConfig     `yaml:"gas_policy" json:"gas_policy"`
}

// ProxyEndpoint represents a proxy server endpoint
type ProxyEndpoint struct {
	URL    string `yaml:"url" json:"url"`
	APIKey string `yaml:"api_key" json:"api_key"`
}

// GasPolicyConfig contains gas policy configuration
type GasPolicyConfig struct {
	MaxGasPrice     string `yaml:"max_gas_price" json:"max_gas_price"`
	SponsorAddress  string `yaml:"sponsor_address" json:"sponsor_address"`
}

// RegistrationConfig contains agent registration configuration
type RegistrationConfig struct {
	AutoRegister  bool              `yaml:"auto_register" json:"auto_register"`
	CheckInterval time.Duration     `yaml:"check_interval" json:"check_interval"`
	RetryPolicy   RetryPolicyConfig `yaml:"retry_policy" json:"retry_policy"`
}

// RetryPolicyConfig contains retry policy configuration
type RetryPolicyConfig struct {
	MaxAttempts  int           `yaml:"max_attempts" json:"max_attempts"`
	Backoff      string        `yaml:"backoff" json:"backoff"` // linear, exponential
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay     time.Duration `yaml:"max_delay" json:"max_delay"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	SignatureAlgorithms []string             `yaml:"signature_algorithms" json:"signature_algorithms"`
	Verification        VerificationConfig   `yaml:"verification" json:"verification"`
}

// VerificationConfig contains verification configuration
type VerificationConfig struct {
	RequireActiveAgent bool          `yaml:"require_active_agent" json:"require_active_agent"`
	MaxClockSkew       time.Duration `yaml:"max_clock_skew" json:"max_clock_skew"`
	CacheTTL           time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`   // debug, info, warn, error
	Format string `yaml:"format" json:"format"` // json, text
	Output string `yaml:"output" json:"output"` // stdout, stderr, file path
}