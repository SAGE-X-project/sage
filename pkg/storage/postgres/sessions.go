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
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sage-x-project/sage/pkg/storage"
)

// SessionStore implements storage.SessionStore for PostgreSQL
type SessionStore struct {
	db *pgxpool.Pool
}

// Create creates a new session
func (s *SessionStore) Create(ctx context.Context, session *storage.Session) error {
	metadata, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO sessions (id, client_did, server_did, session_key, created_at, expires_at, last_activity, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = s.db.Exec(ctx, query,
		session.ID,
		session.ClientDID,
		session.ServerDID,
		session.SessionKey,
		session.CreatedAt,
		session.ExpiresAt,
		session.LastActivity,
		metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// Get retrieves a session by ID
func (s *SessionStore) Get(ctx context.Context, id string) (*storage.Session, error) {
	query := `
		SELECT id, client_did, server_did, session_key, created_at, expires_at, last_activity, metadata
		FROM sessions
		WHERE id = $1 AND expires_at > NOW()
	`

	var session storage.Session
	var metadataJSON []byte

	err := s.db.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.ClientDID,
		&session.ServerDID,
		&session.SessionKey,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastActivity,
		&metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &session, nil
}

// Update updates an existing session
func (s *SessionStore) Update(ctx context.Context, session *storage.Session) error {
	metadata, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE sessions
		SET session_key = $1, expires_at = $2, last_activity = $3, metadata = $4
		WHERE id = $5
	`

	result, err := s.db.Exec(ctx, query,
		session.SessionKey,
		session.ExpiresAt,
		session.LastActivity,
		metadata,
		session.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	return nil
}

// Delete deletes a session by ID
func (s *SessionStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found: %s", id)
	}

	return nil
}

// DeleteExpired deletes all expired sessions
func (s *SessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at <= NOW()`

	result, err := s.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return result.RowsAffected(), nil
}

// List lists all sessions for a client DID
func (s *SessionStore) List(ctx context.Context, clientDID string, limit, offset int) ([]*storage.Session, error) {
	query := `
		SELECT id, client_did, server_did, session_key, created_at, expires_at, last_activity, metadata
		FROM sessions
		WHERE client_did = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(ctx, query, clientDID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*storage.Session
	for rows.Next() {
		var session storage.Session
		var metadataJSON []byte

		err := rows.Scan(
			&session.ID,
			&session.ClientDID,
			&session.ServerDID,
			&session.SessionKey,
			&session.CreatedAt,
			&session.ExpiresAt,
			&session.LastActivity,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		sessions = append(sessions, &session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// UpdateActivity updates the last activity timestamp
func (s *SessionStore) UpdateActivity(ctx context.Context, id string) error {
	query := `UPDATE sessions SET last_activity = $1 WHERE id = $2`

	result, err := s.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update activity: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found: %s", id)
	}

	return nil
}

// Count returns the total number of active sessions
func (s *SessionStore) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE expires_at > NOW()`

	var count int64
	err := s.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	return count, nil
}
