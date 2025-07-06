package rotation

import (
	"fmt"
	"sync"
	"time"

	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
)

// keyRotator implements the KeyRotator interface
type keyRotator struct {
	storage  sagecrypto.KeyStorage
	config   sagecrypto.KeyRotationConfig
	history  map[string][]sagecrypto.KeyRotationEvent
	mu       sync.RWMutex
	rotating map[string]bool // Track keys currently being rotated
}

// NewKeyRotator creates a new key rotator
func NewKeyRotator(storage sagecrypto.KeyStorage) sagecrypto.KeyRotator {
	return &keyRotator{
		storage: storage,
		config: sagecrypto.KeyRotationConfig{
			// Only KeepOldKeys is currently used
			// RotationInterval and MaxKeyAge are reserved for future auto-rotation feature
			KeepOldKeys: false,
		},
		history:  make(map[string][]sagecrypto.KeyRotationEvent),
		rotating: make(map[string]bool),
	}
}

// Rotate rotates the key for the given ID
func (r *keyRotator) Rotate(id string) (sagecrypto.KeyPair, error) {
	r.mu.Lock()
	
	// Check if key is already being rotated
	if r.rotating[id] {
		r.mu.Unlock()
		return nil, fmt.Errorf("key %s is already being rotated", id)
	}
	r.rotating[id] = true
	r.mu.Unlock()

	// Ensure we clear the rotating flag when done
	defer func() {
		r.mu.Lock()
		delete(r.rotating, id)
		r.mu.Unlock()
	}()

	// Load existing key
	oldKeyPair, err := r.storage.Load(id)
	if err != nil {
		return nil, err
	}

	// Generate new key of the same type
	var newKeyPair sagecrypto.KeyPair
	switch oldKeyPair.Type() {
	case sagecrypto.KeyTypeEd25519:
		newKeyPair, err = keys.GenerateEd25519KeyPair()
	case sagecrypto.KeyTypeSecp256k1:
		newKeyPair, err = keys.GenerateSecp256k1KeyPair()
	default:
		return nil, fmt.Errorf("unsupported key type for rotation: %s", oldKeyPair.Type())
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}

	// Store old key if configured
	if r.config.KeepOldKeys {
		oldKeyID := fmt.Sprintf("%s.old.%s", id, oldKeyPair.ID())
		if err := r.storage.Store(oldKeyID, oldKeyPair); err != nil {
			return nil, fmt.Errorf("failed to store old key: %w", err)
		}
	}

	// Store new key
	if err := r.storage.Store(id, newKeyPair); err != nil {
		return nil, fmt.Errorf("failed to store new key: %w", err)
	}

	// Record rotation event
	r.mu.Lock()
	event := sagecrypto.KeyRotationEvent{
		Timestamp: time.Now(),
		OldKeyID:  oldKeyPair.ID(),
		NewKeyID:  newKeyPair.ID(),
		Reason:    "Manual rotation",
	}
	// Append to the end for better performance
	r.history[id] = append(r.history[id], event)
	r.mu.Unlock()

	return newKeyPair, nil
}

// SetRotationConfig sets the rotation configuration
func (r *keyRotator) SetRotationConfig(config sagecrypto.KeyRotationConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = config
}

// GetRotationHistory returns the rotation history for a key
func (r *keyRotator) GetRotationHistory(id string) ([]sagecrypto.KeyRotationEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	history, exists := r.history[id]
	if !exists {
		return []sagecrypto.KeyRotationEvent{}, nil
	}
	
	// Return a copy in reverse order (newest first)
	result := make([]sagecrypto.KeyRotationEvent, len(history))
	for i, event := range history {
		result[len(history)-1-i] = event
	}
	
	return result, nil
}