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
