package health

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/telemetry/metrics"
)

// slog satisfies the local Logger contract; so must the telemetry snapshot handler fit /metrics.
var _ Logger = slog.Default()

func TestServer_MetricsEndpointIsInjected(t *testing.T) {
	srv := NewServer(NewChecker(""), nil, 0)

	rec := httptest.NewRecorder()
	srv.handleMetrics(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code, "no handler configured")

	srv.SetMetricsHandler(metrics.SnapshotHandler())
	rec = httptest.NewRecorder()
	srv.handleMetrics(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"counters"`)
}
