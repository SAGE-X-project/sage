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

	"github.com/sage-x-project/sage/pkg/storage"
)

// DIDStore implements storage.DIDStore
type dIDStore struct {
	store *Store
}

func (d *dIDStore) Create(ctx context.Context, did *storage.DID) error {
	d.store.didsMu.Lock()
	defer d.store.didsMu.Unlock()

	if _, exists := d.store.dids[did.DID]; exists {
		return fmt.Errorf("%w: DID %s", storage.ErrAlreadyExists, did.DID)
	}

	// Deep copy
	didCopy := *did
	if did.PublicKey != nil {
		didCopy.PublicKey = make([]byte, len(did.PublicKey))
		copy(didCopy.PublicKey, did.PublicKey)
	}

	d.store.dids[did.DID] = &didCopy
	return nil
}

func (d *dIDStore) Get(ctx context.Context, did string) (*storage.DID, error) {
	d.store.didsMu.RLock()
	defer d.store.didsMu.RUnlock()

	didData, exists := d.store.dids[did]
	if !exists {
		return nil, fmt.Errorf("%w: DID %s", storage.ErrNotFound, did)
	}

	// Return copy
	didCopy := *didData
	return &didCopy, nil
}

func (d *dIDStore) Update(ctx context.Context, did *storage.DID) error {
	d.store.didsMu.Lock()
	defer d.store.didsMu.Unlock()

	if _, exists := d.store.dids[did.DID]; !exists {
		return fmt.Errorf("%w: DID %s", storage.ErrNotFound, did.DID)
	}

	didCopy := *did
	d.store.dids[did.DID] = &didCopy
	return nil
}

func (d *dIDStore) Delete(ctx context.Context, did string) error {
	d.store.didsMu.Lock()
	defer d.store.didsMu.Unlock()

	if _, exists := d.store.dids[did]; !exists {
		return fmt.Errorf("%w: DID %s", storage.ErrNotFound, did)
	}

	delete(d.store.dids, did)
	return nil
}

func (d *dIDStore) ListByOwner(ctx context.Context, ownerAddress string) ([]*storage.DID, error) {
	d.store.didsMu.RLock()
	defer d.store.didsMu.RUnlock()

	var dids []*storage.DID

	for _, did := range d.store.dids {
		if did.OwnerAddress == ownerAddress {
			didCopy := *did
			dids = append(dids, &didCopy)
		}
	}

	return dids, nil
}

func (d *dIDStore) Revoke(ctx context.Context, did string) error {
	d.store.didsMu.Lock()
	defer d.store.didsMu.Unlock()

	didData, exists := d.store.dids[did]
	if !exists {
		return fmt.Errorf("%w: DID %s", storage.ErrNotFound, did)
	}

	didData.Revoked = true
	return nil
}

func (d *dIDStore) IsRevoked(ctx context.Context, did string) (bool, error) {
	d.store.didsMu.RLock()
	defer d.store.didsMu.RUnlock()

	didData, exists := d.store.dids[did]
	if !exists {
		return false, fmt.Errorf("%w: DID %s", storage.ErrNotFound, did)
	}

	return didData.Revoked, nil
}
