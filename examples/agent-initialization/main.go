//go:build ignore

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

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/sage-x-project/sage/internal/cryptoinit"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Agent represents a SAGE agent with cryptographic keys
type Agent struct {
	Name          string
	DID           string
	KeyDir        string
	ECDSAKey      *ecdsa.PrivateKey
	Ed25519Key    ed25519.PrivateKey
	X25519Key     []byte
	IsInitialized bool
}

// KeyManager handles key generation, loading, and saving
type KeyManager struct {
	keyDir string
}

// NewKeyManager creates a new key manager
func NewKeyManager(keyDir string) *KeyManager {
	return &KeyManager{
		keyDir: keyDir,
	}
}

// EnsureKeyDir creates the key directory if it doesn't exist
func (km *KeyManager) EnsureKeyDir() error {
	return os.MkdirAll(km.keyDir, 0700)
}

// KeyExists checks if a key file exists
func (km *KeyManager) KeyExists(keyType string) bool {
	keyPath := filepath.Join(km.keyDir, fmt.Sprintf("%s.key", keyType))
	_, err := os.Stat(keyPath)
	return err == nil
}

// SaveECDSAKey saves an ECDSA private key to file
func (km *KeyManager) SaveECDSAKey(key *ecdsa.PrivateKey) error {
	keyPath := filepath.Join(km.keyDir, "ecdsa.key")

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	return os.WriteFile(keyPath, keyPEM, 0600)
}

// LoadECDSAKey loads an ECDSA private key from file
func (km *KeyManager) LoadECDSAKey() (*ecdsa.PrivateKey, error) {
	keyPath := filepath.Join(km.keyDir, "ecdsa.key")

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ECDSA key file: %w", err)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ECDSA key: %w", err)
	}

	return key, nil
}

// SaveEd25519Key saves an Ed25519 private key to file
func (km *KeyManager) SaveEd25519Key(key ed25519.PrivateKey) error {
	keyPath := filepath.Join(km.keyDir, "ed25519.key")
	return os.WriteFile(keyPath, key, 0600)
}

// LoadEd25519Key loads an Ed25519 private key from file
func (km *KeyManager) LoadEd25519Key() (ed25519.PrivateKey, error) {
	keyPath := filepath.Join(km.keyDir, "ed25519.key")

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Ed25519 key file: %w", err)
	}

	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid Ed25519 key size: expected %d, got %d", ed25519.PrivateKeySize, len(keyBytes))
	}

	return ed25519.PrivateKey(keyBytes), nil
}

// SaveX25519Key saves an X25519 private key to file
func (km *KeyManager) SaveX25519Key(key []byte) error {
	keyPath := filepath.Join(km.keyDir, "x25519.key")
	return os.WriteFile(keyPath, key, 0600)
}

// LoadX25519Key loads an X25519 private key from file
func (km *KeyManager) LoadX25519Key() ([]byte, error) {
	keyPath := filepath.Join(km.keyDir, "x25519.key")

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read X25519 key file: %w", err)
	}

	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("invalid X25519 key size: expected 32, got %d", len(keyBytes))
	}

	return keyBytes, nil
}

// GenerateOrLoadKeys generates new keys or loads existing ones
func (km *KeyManager) GenerateOrLoadKeys() (*ecdsa.PrivateKey, ed25519.PrivateKey, []byte, error) {
	// Ensure key directory exists
	if err := km.EnsureKeyDir(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	var ecdsaKey *ecdsa.PrivateKey
	var ed25519Key ed25519.PrivateKey
	var x25519Key []byte
	var err error

	// ECDSA Key
	if km.KeyExists("ecdsa") {
		fmt.Println("🔑 Loading existing ECDSA key...")
		ecdsaKey, err = km.LoadECDSAKey()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load ECDSA key: %w", err)
		}
		fmt.Println("✓ ECDSA key loaded from file")
	} else {
		fmt.Println("🔑 Generating new ECDSA key...")
		ecdsaKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
		}
		if err := km.SaveECDSAKey(ecdsaKey); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to save ECDSA key: %w", err)
		}
		fmt.Println("✓ ECDSA key generated and saved")
	}

	// Ed25519 Key
	if km.KeyExists("ed25519") {
		fmt.Println("🔑 Loading existing Ed25519 key...")
		ed25519Key, err = km.LoadEd25519Key()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load Ed25519 key: %w", err)
		}
		fmt.Println("✓ Ed25519 key loaded from file")
	} else {
		fmt.Println("🔑 Generating new Ed25519 key...")
		_, ed25519Key, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
		}
		if err := km.SaveEd25519Key(ed25519Key); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to save Ed25519 key: %w", err)
		}
		fmt.Println("✓ Ed25519 key generated and saved")
	}

	// X25519 Key
	if km.KeyExists("x25519") {
		fmt.Println("🔑 Loading existing X25519 key...")
		x25519Key, err = km.LoadX25519Key()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load X25519 key: %w", err)
		}
		fmt.Println("✓ X25519 key loaded from file")
	} else {
		fmt.Println("🔑 Generating new X25519 key...")
		x25519Key = make([]byte, 32)
		if _, err := rand.Read(x25519Key); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate X25519 key: %w", err)
		}
		if err := km.SaveX25519Key(x25519Key); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to save X25519 key: %w", err)
		}
		fmt.Println("✓ X25519 key generated and saved")
	}

	return ecdsaKey, ed25519Key, x25519Key, nil
}

// NewAgent creates a new agent with key management
func NewAgent(name, keyDir string) (*Agent, error) {
	agent := &Agent{
		Name:   name,
		KeyDir: keyDir,
	}

	// Initialize key manager
	keyManager := NewKeyManager(keyDir)

	// Generate or load keys
	ecdsaKey, ed25519Key, x25519Key, err := keyManager.GenerateOrLoadKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keys: %w", err)
	}

	// Set keys
	agent.ECDSAKey = ecdsaKey
	agent.Ed25519Key = ed25519Key
	agent.X25519Key = x25519Key
	agent.IsInitialized = true

	// Generate DID
	agent.DID = did.GenerateDID(did.ChainEthereum, name)

	return agent, nil
}

// GetPublicKeys returns the public keys for registration
func (a *Agent) GetPublicKeys() ([]byte, []byte, []byte, error) {
	if !a.IsInitialized {
		return nil, nil, nil, fmt.Errorf("agent not initialized")
	}

	// ECDSA public key
	ecdsaPubKey := elliptic.Marshal(elliptic.P256(), a.ECDSAKey.PublicKey.X, a.ECDSAKey.PublicKey.Y)

	// Ed25519 public key
	ed25519PubKey := a.Ed25519Key.Public().(ed25519.PublicKey)

	// X25519 public key (derive from private key)
	x25519PubKey, err := crypto.X25519PublicKeyFromPrivate(a.X25519Key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to derive X25519 public key: %w", err)
	}

	return ecdsaPubKey, ed25519PubKey, x25519PubKey, nil
}

// RegisterOnBlockchain registers the agent on the blockchain
func (a *Agent) RegisterOnBlockchain(registryAddress, rpcURL, privateKeyHex string) error {
	if !a.IsInitialized {
		return fmt.Errorf("agent not initialized")
	}

	fmt.Println("🔗 Registering agent on blockchain...")

	// Create DID manager
	manager := did.NewManager()
	config := &did.RegistryConfig{
		ContractAddress:    registryAddress,
		RPCEndpoint:        rpcURL,
		PrivateKey:         privateKeyHex,
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	err := manager.Configure(did.ChainEthereum, config)
	if err != nil {
		return fmt.Errorf("failed to configure manager: %w", err)
	}

	// Get public keys
	ecdsaPubKey, ed25519PubKey, x25519PubKey, err := a.GetPublicKeys()
	if err != nil {
		return fmt.Errorf("failed to get public keys: %w", err)
	}

	// Marshal public keys for registration
	ecdsaPubKeyBytes, err := did.MarshalPublicKey(ecdsaPubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA public key: %w", err)
	}

	ed25519PubKeyBytes, err := did.MarshalPublicKey(ed25519PubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal Ed25519 public key: %w", err)
	}

	x25519PubKeyBytes, err := did.MarshalPublicKey(x25519PubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal X25519 public key: %w", err)
	}

	// Create registration request
	req := &did.RegistrationRequest{
		DID:         a.DID,
		Name:        a.Name,
		Description: fmt.Sprintf("Agent %s with automatic key management", a.Name),
		Endpoint:    fmt.Sprintf("https://%s.example.com", a.Name),
		Capabilities: map[string]interface{}{
			"messaging":  true,
			"encryption": true,
			"signing":    true,
		},
		KeyPair: a.ECDSAKey,
		Keys: []did.AgentKey{
			{
				Type:    did.KeyTypeEd25519,
				KeyData: ed25519PubKeyBytes,
			},
			{
				Type:    did.KeyTypeX25519,
				KeyData: x25519PubKeyBytes,
			},
		},
	}

	// Register agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := manager.RegisterAgent(ctx, did.ChainEthereum, req)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	fmt.Printf("✅ Agent registered successfully!\n")
	fmt.Printf("   Transaction Hash: %s\n", result.TransactionHash)
	fmt.Printf("   Block Number: %d\n", result.BlockNumber)
	fmt.Printf("   Gas Used: %d\n", result.GasUsed)

	return nil
}

// PrintInfo prints agent information
func (a *Agent) PrintInfo() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Agent Information                      ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Name:           %s\n", a.Name)
	fmt.Printf("DID:            %s\n", a.DID)
	fmt.Printf("Key Directory:  %s\n", a.KeyDir)
	fmt.Printf("Initialized:    %t\n", a.IsInitialized)
	fmt.Println()

	if a.IsInitialized {
		fmt.Println("Keys:")
		fmt.Printf("  ECDSA:        %d bytes\n", len(a.ECDSAKey.D.Bytes()))
		fmt.Printf("  Ed25519:      %d bytes\n", len(a.Ed25519Key))
		fmt.Printf("  X25519:       %d bytes\n", len(a.X25519Key))
		fmt.Println()
	}
}

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║     SAGE Agent Initialization with Key Management        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Configuration
	agentName := "my-agent"
	keyDir := "./keys"
	registryAddress := os.Getenv("REGISTRY_ADDRESS")
	rpcURL := os.Getenv("RPC_URL")
	privateKeyHex := os.Getenv("PRIVATE_KEY")

	// Set defaults for local development
	if registryAddress == "" {
		registryAddress = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
	}
	if rpcURL == "" {
		rpcURL = "http://localhost:8545"
	}
	if privateKeyHex == "" {
		privateKeyHex = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	}

	fmt.Println("📋 Configuration")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Printf("Agent Name:      %s\n", agentName)
	fmt.Printf("Key Directory:   %s\n", keyDir)
	fmt.Printf("Registry:        %s\n", registryAddress)
	fmt.Printf("RPC URL:         %s\n", rpcURL)
	fmt.Println()

	// Step 1: Create agent with key management
	fmt.Println("🤖 Step 1: Initializing Agent")
	fmt.Println("─────────────────────────────────────────────────────────")

	agent, err := NewAgent(agentName, keyDir)
	if err != nil {
		fmt.Printf("❌ Failed to initialize agent: %v\n", err)
		os.Exit(1)
	}

	agent.PrintInfo()

	// Step 2: Register on blockchain (optional)
	fmt.Println("🚀 Step 2: Blockchain Registration (Optional)")
	fmt.Println("─────────────────────────────────────────────────────────")

	registerOnChain := os.Getenv("REGISTER_ON_CHAIN")
	if registerOnChain == "true" {
		err = agent.RegisterOnBlockchain(registryAddress, rpcURL, privateKeyHex)
		if err != nil {
			fmt.Printf("⚠️  Registration failed: %v\n", err)
			fmt.Println("   Agent keys are still available for local use")
		}
	} else {
		fmt.Println("Skipping blockchain registration (set REGISTER_ON_CHAIN=true to enable)")
		fmt.Println("Agent is ready for local operations")
	}

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Initialization Complete!               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🎉 Your agent is ready to use!")
	fmt.Println()
	fmt.Println("Key files saved in:", keyDir)
	fmt.Println("  - ecdsa.key    (ECDSA private key)")
	fmt.Println("  - ed25519.key  (Ed25519 private key)")
	fmt.Println("  - x25519.key   (X25519 private key)")
	fmt.Println()
	fmt.Println("Next time you run this program, it will automatically load")
	fmt.Println("the existing keys instead of generating new ones.")
	fmt.Println()
}
