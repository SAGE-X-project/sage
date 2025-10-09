package memory

import (
	"context"
	"fmt"

	"github.com/sage-x-project/sage/pkg/storage"
)

// DIDStore implements storage.DIDStore
type DIDStore struct {
	store *Store
}

func (d *DIDStore) Create(ctx context.Context, did *storage.DID) error {
	d.store.didsMu.Lock()
	defer d.store.didsMu.Unlock()

	if _, exists := d.store.dids[did.DID]; exists {
		return fmt.Errorf("DID already exists: %s", did.DID)
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

func (d *DIDStore) Get(ctx context.Context, did string) (*storage.DID, error) {
	d.store.didsMu.RLock()
	defer d.store.didsMu.RUnlock()

	didData, exists := d.store.dids[did]
	if !exists {
		return nil, fmt.Errorf("DID not found: %s", did)
	}

	// Return copy
	didCopy := *didData
	return &didCopy, nil
}

func (d *DIDStore) Update(ctx context.Context, did *storage.DID) error {
	d.store.didsMu.Lock()
	defer d.store.didsMu.Unlock()

	if _, exists := d.store.dids[did.DID]; !exists {
		return fmt.Errorf("DID not found: %s", did.DID)
	}

	didCopy := *did
	d.store.dids[did.DID] = &didCopy
	return nil
}

func (d *DIDStore) Delete(ctx context.Context, did string) error {
	d.store.didsMu.Lock()
	defer d.store.didsMu.Unlock()

	if _, exists := d.store.dids[did]; !exists {
		return fmt.Errorf("DID not found: %s", did)
	}

	delete(d.store.dids, did)
	return nil
}

func (d *DIDStore) ListByOwner(ctx context.Context, ownerAddress string) ([]*storage.DID, error) {
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

func (d *DIDStore) Revoke(ctx context.Context, did string) error {
	d.store.didsMu.Lock()
	defer d.store.didsMu.Unlock()

	didData, exists := d.store.dids[did]
	if !exists {
		return fmt.Errorf("DID not found: %s", did)
	}

	didData.Revoked = true
	return nil
}

func (d *DIDStore) IsRevoked(ctx context.Context, did string) (bool, error) {
	d.store.didsMu.RLock()
	defer d.store.didsMu.RUnlock()

	didData, exists := d.store.dids[did]
	if !exists {
		return false, fmt.Errorf("DID not found: %s", did)
	}

	return didData.Revoked, nil
}
