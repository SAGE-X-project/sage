package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sage-x-project/sage/pkg/storage"
)

// DIDStore implements storage.DIDStore for PostgreSQL
type DIDStore struct {
	db *pgxpool.Pool
}

// Create creates a new DID entry
func (d *DIDStore) Create(ctx context.Context, did *storage.DID) error {
	query := `
		INSERT INTO dids (did, public_key, owner_address, key_type, revoked, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := d.db.Exec(ctx, query,
		did.DID,
		did.PublicKey,
		did.OwnerAddress,
		did.KeyType,
		did.Revoked,
		did.CreatedAt,
		did.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create DID: %w", err)
	}

	return nil
}

// Get retrieves a DID by its identifier
func (d *DIDStore) Get(ctx context.Context, did string) (*storage.DID, error) {
	query := `
		SELECT did, public_key, owner_address, key_type, revoked, created_at, updated_at
		FROM dids
		WHERE did = $1
	`

	var result storage.DID
	err := d.db.QueryRow(ctx, query, did).Scan(
		&result.DID,
		&result.PublicKey,
		&result.OwnerAddress,
		&result.KeyType,
		&result.Revoked,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("DID not found: %s", did)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get DID: %w", err)
	}

	return &result, nil
}

// Update updates an existing DID
func (d *DIDStore) Update(ctx context.Context, did *storage.DID) error {
	query := `
		UPDATE dids
		SET public_key = $1, owner_address = $2, key_type = $3, revoked = $4
		WHERE did = $5
	`

	result, err := d.db.Exec(ctx, query,
		did.PublicKey,
		did.OwnerAddress,
		did.KeyType,
		did.Revoked,
		did.DID,
	)

	if err != nil {
		return fmt.Errorf("failed to update DID: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("DID not found: %s", did.DID)
	}

	return nil
}

// Delete deletes a DID
func (d *DIDStore) Delete(ctx context.Context, did string) error {
	query := `DELETE FROM dids WHERE did = $1`

	result, err := d.db.Exec(ctx, query, did)
	if err != nil {
		return fmt.Errorf("failed to delete DID: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("DID not found: %s", did)
	}

	return nil
}

// ListByOwner lists all DIDs owned by an address
func (d *DIDStore) ListByOwner(ctx context.Context, ownerAddress string) ([]*storage.DID, error) {
	query := `
		SELECT did, public_key, owner_address, key_type, revoked, created_at, updated_at
		FROM dids
		WHERE owner_address = $1
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(ctx, query, ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to list DIDs: %w", err)
	}
	defer rows.Close()

	var dids []*storage.DID
	for rows.Next() {
		var did storage.DID
		err := rows.Scan(
			&did.DID,
			&did.PublicKey,
			&did.OwnerAddress,
			&did.KeyType,
			&did.Revoked,
			&did.CreatedAt,
			&did.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan DID: %w", err)
		}

		dids = append(dids, &did)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating DIDs: %w", err)
	}

	return dids, nil
}

// Revoke marks a DID as revoked
func (d *DIDStore) Revoke(ctx context.Context, did string) error {
	query := `UPDATE dids SET revoked = true WHERE did = $1`

	result, err := d.db.Exec(ctx, query, did)
	if err != nil {
		return fmt.Errorf("failed to revoke DID: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("DID not found: %s", did)
	}

	return nil
}

// IsRevoked checks if a DID is revoked
func (d *DIDStore) IsRevoked(ctx context.Context, did string) (bool, error) {
	query := `SELECT revoked FROM dids WHERE did = $1`

	var revoked bool
	err := d.db.QueryRow(ctx, query, did).Scan(&revoked)
	if err == pgx.ErrNoRows {
		return false, fmt.Errorf("DID not found: %s", did)
	}
	if err != nil {
		return false, fmt.Errorf("failed to check DID revocation: %w", err)
	}

	return revoked, nil
}
