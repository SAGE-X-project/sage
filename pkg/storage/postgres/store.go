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

package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sage-x-project/sage/pkg/storage"
)

// Store implements the storage.Store interface for PostgreSQL
type Store struct {
	pool    *pgxpool.Pool
	session *SessionStore
	nonce   *NonceStore
	did     *DIDStore
}

// Config holds PostgreSQL connection configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// NewStore creates a new PostgreSQL store
func NewStore(ctx context.Context, cfg *Config) (*Store, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{
		pool: pool,
	}

	// Initialize sub-stores
	store.session = &SessionStore{db: pool}
	store.nonce = &NonceStore{db: pool}
	store.did = &DIDStore{db: pool}

	return store, nil
}

// SessionStore returns the session store
func (s *Store) SessionStore() storage.SessionStore {
	return s.session
}

// NonceStore returns the nonce store
func (s *Store) NonceStore() storage.NonceStore {
	return s.nonce
}

// DIDStore returns the DID store
func (s *Store) DIDStore() storage.DIDStore {
	return s.did
}

// Close closes the database connection pool
func (s *Store) Close() error {
	s.pool.Close()
	return nil
}

// Ping checks the database connection
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
