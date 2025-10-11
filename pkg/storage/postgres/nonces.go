package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sage-x-project/sage/pkg/storage"
)

// NonceStore implements storage.NonceStore for PostgreSQL
type NonceStore struct {
	db *pgxpool.Pool
}

// CheckAndStore atomically checks if nonce is used and stores it
func (n *NonceStore) CheckAndStore(ctx context.Context, nonce string, sessionID string, expiresAt time.Time) error {
	// Use a transaction to ensure atomicity
	tx, err := n.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if nonce already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM nonces WHERE nonce = $1)`
	err = tx.QueryRow(ctx, checkQuery, nonce).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check nonce: %w", err)
	}

	if exists {
		return fmt.Errorf("nonce already used: %s", nonce)
	}

	// Store the nonce
	insertQuery := `
		INSERT INTO nonces (nonce, session_id, used_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(ctx, insertQuery, nonce, sessionID, time.Now(), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store nonce: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// IsUsed checks if a nonce has been used
func (n *NonceStore) IsUsed(ctx context.Context, nonce string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM nonces
			WHERE nonce = $1 AND expires_at > NOW()
		)
	`

	var used bool
	err := n.db.QueryRow(ctx, query, nonce).Scan(&used)
	if err != nil {
		return false, fmt.Errorf("failed to check nonce: %w", err)
	}

	return used, nil
}

// DeleteExpired deletes all expired nonces
func (n *NonceStore) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM nonces WHERE expires_at <= NOW()`

	result, err := n.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired nonces: %w", err)
	}

	return result.RowsAffected(), nil
}

// Count returns the total number of stored nonces
func (n *NonceStore) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM nonces WHERE expires_at > NOW()`

	var count int64
	err := n.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count nonces: %w", err)
	}

	return count, nil
}

// Get retrieves a nonce by its value (internal use)
func (n *NonceStore) Get(ctx context.Context, nonce string) (*storage.Nonce, error) {
	query := `
		SELECT nonce, session_id, used_at, expires_at
		FROM nonces
		WHERE nonce = $1
	`

	var result storage.Nonce
	err := n.db.QueryRow(ctx, query, nonce).Scan(
		&result.Nonce,
		&result.SessionID,
		&result.UsedAt,
		&result.ExpiresAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("nonce not found: %s", nonce)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	return &result, nil
}
