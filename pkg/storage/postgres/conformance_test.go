package postgres

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/storage"
	"github.com/sage-x-project/sage/pkg/storage/storagetest"
)

// TestPostgresStore_Conformance runs the shared storage contract against a
// PostgreSQL instance described by SAGE_TEST_PG_HOST / _PORT / _USER /
// _PASSWORD / _DATABASE. It is skipped when SAGE_TEST_PG_HOST is unset. The
// database must be empty; each sub-test truncates the tables it uses.
func TestPostgresStore_Conformance(t *testing.T) {
	host := os.Getenv("SAGE_TEST_PG_HOST")
	if host == "" {
		t.Skip("SAGE_TEST_PG_HOST not set; skipping PostgreSQL conformance test")
	}
	port, _ := strconv.Atoi(os.Getenv("SAGE_TEST_PG_PORT"))
	if port == 0 {
		port = 5432
	}
	cfg := &Config{
		Host: host, Port: port,
		User: os.Getenv("SAGE_TEST_PG_USER"), Password: os.Getenv("SAGE_TEST_PG_PASSWORD"),
		Database: os.Getenv("SAGE_TEST_PG_DATABASE"), SSLMode: "disable",
	}
	storagetest.RunConformance(t, func(t *testing.T) storage.Store {
		s, err := NewStore(context.Background(), cfg)
		require.NoError(t, err)
		for _, table := range []string{"sessions", "nonces", "dids"} {
			_, err := s.pool.Exec(context.Background(), "TRUNCATE TABLE "+table)
			require.NoError(t, err)
		}
		return s
	})
}
