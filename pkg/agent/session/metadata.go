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

package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Metadata contains the parameters required for message exchange before the communication channel is established.
type Metadata struct {
	ID string `json:"id"`

	Status    string `json:"status,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// MetadataBuilder constructs metadata instances with a fluent API.
type MetadataBuilder struct {
	metadata Metadata
}

// NewMetadataBuilder initializes a builder with default values.
func NewMetadataBuilder() *MetadataBuilder {
	now := time.Now().UTC()
	return &MetadataBuilder{
		metadata: Metadata{
			ID:        GeneralPrefix + uuid.NewString(),
			CreatedAt: now.Format(time.RFC3339),
			Status:    "proposed",
		},
	}
}

// WithStatus overrides the metadata status (e.g. "proposed", "active", "expired").
func (b *MetadataBuilder) WithStatus(status string) *MetadataBuilder {
	b.metadata.Status = status
	return b
}

// WithCreatedAt sets a custom creation timestamp.
func (b *MetadataBuilder) WithCreatedAt(t time.Time) *MetadataBuilder {
	b.metadata.CreatedAt = t.Format(time.RFC3339)
	return b
}

// WithExpiresAfter sets ExpiresAt to CreatedAt + duration.
func (b *MetadataBuilder) WithExpiresAfter(d time.Duration) *MetadataBuilder {
	created, err := time.Parse(time.RFC3339, b.metadata.CreatedAt)
	if err != nil {
		created = time.Now().UTC()
		b.metadata.CreatedAt = created.Format(time.RFC3339)
	}
	b.metadata.ExpiresAt = created.Add(d).Format(time.RFC3339)
	return b
}

// Build returns the constructed metadata.
func (b *MetadataBuilder) Build() *Metadata {
	return &b.metadata
}

// GenerateSalt generates a cryptographically secure 32-byte salt
func GenerateSalt() (string, error) {
	const saltSize = 32 // 256 bits
	saltBytes := make([]byte, saltSize)

	// crypto/rand.Read uses the system's CSPRNG
	if _, err := rand.Read(saltBytes); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Encode to Base64URL without padding
	return base64.RawURLEncoding.EncodeToString(saltBytes), nil
}
