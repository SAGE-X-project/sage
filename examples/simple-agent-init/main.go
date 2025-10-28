//go:build ignore

// SAGE - Simple Agent Initialization Example
// This example shows how to create an agent that automatically manages its keys:
// - Checks if keys exist on startup
// - Loads existing keys if found
// - Generates new keys if not found
// - Saves keys for future use

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

// SimpleAgent represents a basic agent with automatic key management
type SimpleAgent struct {
	Name       string
	DID        string
	KeyDir     string
	ECDSAKey   *ecdsa.PrivateKey
	Ed25519Key ed25519.PrivateKey
	X25519Key  []byte
}

// NewSimpleAgent creates a new agent with automatic key management
func NewSimpleAgent(name, keyDir string) (*SimpleAgent, error) {
	agent := &SimpleAgent{
		Name:   name,
		KeyDir: keyDir,
	}

	// Ensure key directory exists
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Initialize keys (load existing or generate new)
	if err := agent.initializeKeys(); err != nil {
		return nil, fmt.Errorf("failed to initialize keys: %w", err)
	}

	// Generate DID
	agent.DID = did.GenerateDID(did.ChainEthereum, name)

	return agent, nil
}

// initializeKeys loads existing keys or generates new ones
func (a *SimpleAgent) initializeKeys() error {
	fmt.Println("🔑 Initializing cryptographic keys...")

	// ECDSA Key
	if err := a.loadOrGenerateECDSAKey(); err != nil {
		return fmt.Errorf("ECDSA key error: %w", err)
	}

	// Ed25519 Key
	if err := a.loadOrGenerateEd25519Key(); err != nil {
		return fmt.Errorf("Ed25519 key error: %w", err)
	}

	// X25519 Key
	if err := a.loadOrGenerateX25519Key(); err != nil {
		return fmt.Errorf("X25519 key error: %w", err)
	}

	fmt.Println("✅ All keys initialized successfully")
	return nil
}

// loadOrGenerateECDSAKey loads ECDSA key or generates a new one
func (a *SimpleAgent) loadOrGenerateECDSAKey() error {
	keyPath := filepath.Join(a.KeyDir, "ecdsa.key")

	// Try to load existing key
	if keyData, err := os.ReadFile(keyPath); err == nil {
		block, _ := pem.Decode(keyData)
		if block != nil {
			if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
				a.ECDSAKey = key
				fmt.Println("  ✓ ECDSA key loaded from file")
				return nil
			}
		}
	}

	// Generate new key
	fmt.Println("  🔄 Generating new ECDSA key...")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	// Save key
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to save ECDSA key: %w", err)
	}

	a.ECDSAKey = key
	fmt.Println("  ✓ ECDSA key generated and saved")
	return nil
}

// loadOrGenerateEd25519Key loads Ed25519 key or generates a new one
func (a *SimpleAgent) loadOrGenerateEd25519Key() error {
	keyPath := filepath.Join(a.KeyDir, "ed25519.key")

	// Try to load existing key
	if keyData, err := os.ReadFile(keyPath); err == nil {
		if len(keyData) == ed25519.PrivateKeySize {
			a.Ed25519Key = ed25519.PrivateKey(keyData)
			fmt.Println("  ✓ Ed25519 key loaded from file")
			return nil
		}
	}

	// Generate new key
	fmt.Println("  🔄 Generating new Ed25519 key...")
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}

	// Save key
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return fmt.Errorf("failed to save Ed25519 key: %w", err)
	}

	a.Ed25519Key = key
	fmt.Println("  ✓ Ed25519 key generated and saved")
	return nil
}

// loadOrGenerateX25519Key loads X25519 key or generates a new one
func (a *SimpleAgent) loadOrGenerateX25519Key() error {
	keyPath := filepath.Join(a.KeyDir, "x25519.key")

	// Try to load existing key
	if keyData, err := os.ReadFile(keyPath); err == nil {
		if len(keyData) == 32 {
			a.X25519Key = keyData
			fmt.Println("  ✓ X25519 key loaded from file")
			return nil
		}
	}

	// Generate new key
	fmt.Println("  🔄 Generating new X25519 key...")
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate X25519 key: %w", err)
	}

	// Save key
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return fmt.Errorf("failed to save X25519 key: %w", err)
	}

	a.X25519Key = key
	fmt.Println("  ✓ X25519 key generated and saved")
	return nil
}

// GetPublicKeys returns the public keys
func (a *SimpleAgent) GetPublicKeys() ([]byte, []byte, []byte, error) {
	// ECDSA public key
	ecdsaPubKey := elliptic.Marshal(elliptic.P256(), a.ECDSAKey.PublicKey.X, a.ECDSAKey.PublicKey.Y)

	// Ed25519 public key
	ed25519PubKey := a.Ed25519Key.Public().(ed25519.PublicKey)

	// X25519 public key
	x25519PubKey, err := crypto.X25519PublicKeyFromPrivate(a.X25519Key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to derive X25519 public key: %w", err)
	}

	return ecdsaPubKey, ed25519PubKey, x25519PubKey, nil
}

// SignMessage signs a message with Ed25519
func (a *SimpleAgent) SignMessage(message []byte) []byte {
	return ed25519.Sign(a.Ed25519Key, message)
}

// VerifyMessage verifies a message signature
func (a *SimpleAgent) VerifyMessage(message, signature []byte) bool {
	return ed25519.Verify(a.Ed25519Key.Public().(ed25519.PublicKey), message, signature)
}

// PrintInfo prints agent information
func (a *SimpleAgent) PrintInfo() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Agent Information                      ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Name:           %s\n", a.Name)
	fmt.Printf("DID:            %s\n", a.DID)
	fmt.Printf("Key Directory:  %s\n", a.KeyDir)
	fmt.Println()

	// Show key info
	ecdsaPub, ed25519Pub, x25519Pub, err := a.GetPublicKeys()
	if err != nil {
		fmt.Printf("Error getting public keys: %v\n", err)
		return
	}

	fmt.Println("Keys:")
	fmt.Printf("  ECDSA:        %d bytes (public key)\n", len(ecdsaPub))
	fmt.Printf("  Ed25519:      %d bytes (private key)\n", len(a.Ed25519Key))
	fmt.Printf("  X25519:       %d bytes (private key)\n", len(a.X25519Key))
	fmt.Println()
}

// DemoSigning demonstrates message signing and verification
func (a *SimpleAgent) DemoSigning() {
	fmt.Println("🔐 Testing Message Signing")
	fmt.Println("─────────────────────────────────────────────────────────")

	message := []byte("Hello, SAGE World!")
	fmt.Printf("Message: %s\n", string(message))

	// Sign message
	signature := a.SignMessage(message)
	fmt.Printf("Signature: %x\n", signature)

	// Verify signature
	valid := a.VerifyMessage(message, signature)
	fmt.Printf("Verification: %t\n", valid)

	if valid {
		fmt.Println("✅ Message signing and verification successful!")
	} else {
		fmt.Println("❌ Message verification failed!")
	}
	fmt.Println()
}

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║     SAGE Simple Agent Initialization Example             ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Configuration
	agentName := "simple-agent"
	keyDir := "./keys"

	fmt.Println("📋 Configuration")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Printf("Agent Name:      %s\n", agentName)
	fmt.Printf("Key Directory:   %s\n", keyDir)
	fmt.Println()

	// Create agent
	fmt.Println("🤖 Creating Agent...")
	agent, err := NewSimpleAgent(agentName, keyDir)
	if err != nil {
		fmt.Printf("❌ Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	// Print agent info
	agent.PrintInfo()

	// Demo signing
	agent.DemoSigning()

	// Show key files
	fmt.Println("📁 Key Files")
	fmt.Println("─────────────────────────────────────────────────────────")
	files, err := os.ReadDir(keyDir)
	if err != nil {
		fmt.Printf("Error reading key directory: %v\n", err)
	} else {
		for _, file := range files {
			if !file.IsDir() {
				info, _ := file.Info()
				fmt.Printf("  %s (%d bytes)\n", file.Name(), info.Size())
			}
		}
	}
	fmt.Println()

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Initialization Complete!               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🎉 Your agent is ready!")
	fmt.Println()
	fmt.Println("Key management features:")
	fmt.Println("  ✓ Automatic key detection")
	fmt.Println("  ✓ Key generation when needed")
	fmt.Println("  ✓ Persistent key storage")
	fmt.Println("  ✓ Message signing capability")
	fmt.Println()
	fmt.Println("Run this program again to see it load existing keys!")
	fmt.Println()
}
