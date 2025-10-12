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

package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/sage-x-project/sage/pkg/storage"
)

// NonceStore implements storage.NonceStore
type NonceStore struct {
	store *Store
}

func (n *NonceStore) CheckAndStore(ctx context.Context, nonce string, sessionID string, expiresAt time.Time) error {
	n.store.noncesMu.Lock()
	defer n.store.noncesMu.Unlock()

	// Check if nonce already exists
	if _, exists := n.store.nonces[nonce]; exists {
		return fmt.Errorf("nonce already used: %s", nonce)
	}

	// Store the nonce
	n.store.nonces[nonce] = &storage.Nonce{
		Nonce:     nonce,
		SessionID: sessionID,
		UsedAt:    time.Now(),
		ExpiresAt: expiresAt,
	}

	return nil
}

func (n *NonceStore) IsUsed(ctx context.Context, nonce string) (bool, error) {
	n.store.noncesMu.RLock()
	defer n.store.noncesMu.RUnlock()

	nonceData, exists := n.store.nonces[nonce]
	if !exists {
		return false, nil
	}

	// Check if nonce is expired
	if time.Now().After(nonceData.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

func (n *NonceStore) DeleteExpired(ctx context.Context) (int64, error) {
	n.store.noncesMu.Lock()
	defer n.store.noncesMu.Unlock()

	now := time.Now()
	var count int64

	for nonce, data := range n.store.nonces {
		if now.After(data.ExpiresAt) {
			delete(n.store.nonces, nonce)
			count++
		}
	}

	return count, nil
}

func (n *NonceStore) Count(ctx context.Context) (int64, error) {
	n.store.noncesMu.RLock()
	defer n.store.noncesMu.RUnlock()

	now := time.Now()
	var count int64

	for _, data := range n.store.nonces {
		if now.Before(data.ExpiresAt) {
			count++
		}
	}

	return count, nil
}
