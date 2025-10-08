// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Test loading development config (skip validation for testing)
	cfg, err := Load(LoaderOptions{
		ConfigDir:      ".",
		Environment:    "development",
		SkipValidation: true,
	})

	if err != nil {
		t.Fatalf("Failed to load development config: %v", err)
	}

	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "development")
	}

	// Verify defaults are applied
	if cfg.Session != nil && cfg.Session.MaxIdleTime == 0 {
		t.Error("Session MaxIdleTime should have default value")
	}
}

func TestLoadForEnvironment(t *testing.T) {
	tests := []string{"development", "staging", "production", "local"}

	for _, env := range tests {
		t.Run(env, func(t *testing.T) {
			cfg, err := Load(LoaderOptions{
				ConfigDir:      ".",
				Environment:    env,
				SkipValidation: true,
			})
			if err != nil {
				t.Fatalf("Failed to load %s config: %v", env, err)
			}

			if cfg.Environment != env {
				t.Errorf("Environment = %q, want %q", cfg.Environment, env)
			}
		})
	}
}

func TestLoadWithEnvOverrides(t *testing.T) {
	// Set environment overrides
	os.Setenv("SAGE_BLOCKCHAIN_RPC", "http://override-rpc:8545")
	os.Setenv("SAGE_LOG_LEVEL", "debug")
	defer os.Unsetenv("SAGE_BLOCKCHAIN_RPC")
	defer os.Unsetenv("SAGE_LOG_LEVEL")

	cfg, err := Load(LoaderOptions{
		ConfigDir:      ".",
		Environment:    "development",
		SkipValidation: true,
	})

	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Blockchain != nil && cfg.Blockchain.NetworkRPC != "http://override-rpc:8545" {
		t.Errorf("NetworkRPC = %q, want %q", cfg.Blockchain.NetworkRPC, "http://override-rpc:8545")
	}

	if cfg.Logging != nil && cfg.Logging.Level != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.Logging.Level, "debug")
	}
}

func TestLoadWithCustomConfigDir(t *testing.T) {
	// Create temporary config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Write test config
	testConfig := `
environment: test
logging:
  level: info
  format: json
`
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Try to load (should fall back to empty config with defaults)
	cfg, err := Load(LoaderOptions{
		ConfigDir:      tmpDir,
		Environment:    "test",
		SkipValidation: true,
	})

	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Should have defaults applied even if file doesn't match environment
	if cfg == nil {
		t.Fatal("Config should not be nil")
	}
}

func TestDefaultLoaderOptions(t *testing.T) {
	opts := DefaultLoaderOptions()

	if opts.ConfigDir != "config" {
		t.Errorf("ConfigDir = %q, want %q", opts.ConfigDir, "config")
	}

	if opts.SkipEnvSubstitution {
		t.Error("SkipEnvSubstitution should be false by default")
	}

	if opts.SkipValidation {
		t.Error("SkipValidation should be false by default")
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{}
	setDefaults(cfg)

	if cfg.Environment != "development" {
		t.Errorf("Default environment = %q, want %q", cfg.Environment, "development")
	}
}

func TestSessionConfigDefaults(t *testing.T) {
	cfg := &Config{
		Session: &SessionConfig{},
	}
	setDefaults(cfg)

	if cfg.Session.MaxIdleTime != 30*time.Minute {
		t.Errorf("MaxIdleTime = %v, want %v", cfg.Session.MaxIdleTime, 30*time.Minute)
	}

	if cfg.Session.CleanupInterval != 5*time.Minute {
		t.Errorf("CleanupInterval = %v, want %v", cfg.Session.CleanupInterval, 5*time.Minute)
	}

	if cfg.Session.MaxSessions != 10000 {
		t.Errorf("MaxSessions = %d, want %d", cfg.Session.MaxSessions, 10000)
	}
}

func TestHandshakeConfigDefaults(t *testing.T) {
	cfg := &Config{
		Handshake: &HandshakeConfig{},
	}
	setDefaults(cfg)

	if cfg.Handshake.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Handshake.Timeout, 30*time.Second)
	}

	if cfg.Handshake.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want %d", cfg.Handshake.MaxRetries, 3)
	}

	if cfg.Handshake.RetryBackoff != 1*time.Second {
		t.Errorf("RetryBackoff = %v, want %v", cfg.Handshake.RetryBackoff, 1*time.Second)
	}
}
