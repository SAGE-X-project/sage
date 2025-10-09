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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMetadataBuilder(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		b := NewMetadataBuilder()
		s := b.Build()

		require.NotEmpty(t, s.ID)
		require.Contains(t, s.ID, "-", "ID should contain UUID hyphens")

		// CreatedAt should be valid RFC3339 timestamp
		_, err := time.Parse(time.RFC3339, s.CreatedAt)
		require.NoError(t, err)

		require.Equal(t, "proposed", s.Status, "default status should be 'proposed'")
		require.Empty(t, s.ExpiresAt)
	})

	t.Run("WithStatus", func(t *testing.T) {
		s := NewMetadataBuilder().WithStatus("active").Build()
		require.Equal(t, "active", s.Status)
	})

	t.Run("WithCreatedAt", func(t *testing.T) {
		custom := time.Date(2025, 7, 30, 12, 34, 56, 0, time.UTC)
		s := NewMetadataBuilder().WithCreatedAt(custom).Build()
		require.Equal(t, custom.Format(time.RFC3339), s.CreatedAt)
	})

	t.Run("WithExpiresAfter", func(t *testing.T) {
		d := 2 * time.Hour
		// start with specific CreatedAt to avoid parsing errors
		builder := NewMetadataBuilder().WithCreatedAt(time.Now().UTC())
		s := builder.WithExpiresAfter(d).Build()

		// parse CreatedAt and ExpiresAt
		created, err := time.Parse(time.RFC3339, s.CreatedAt)
		require.NoError(t, err)

		expires, err := time.Parse(time.RFC3339, s.ExpiresAt)
		require.NoError(t, err)
		require.True(t, expires.Sub(created) == d, "ExpiresAt should be CreatedAt + duration")
	})

	t.Run("ChainingAll", func(t *testing.T) {
		custom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		s := NewMetadataBuilder().
			WithCreatedAt(custom).
			WithStatus("active").
			WithExpiresAfter(30*time.Minute).
			Build()

		require.Equal(t, custom.Format(time.RFC3339), s.CreatedAt)
		require.Equal(t, "active", s.Status)

		created, _ := time.Parse(time.RFC3339, s.CreatedAt)
		expires, _ := time.Parse(time.RFC3339, s.ExpiresAt)
		require.Equal(t, created.Add(30*time.Minute), expires)
	})
}
