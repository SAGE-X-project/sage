package session

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/telemetry/metrics"
)

// The Prometheus adapter must satisfy the consumer-side interface.
var _ Metrics = metrics.PrometheusSessionMetrics{}

type recordingMetrics struct {
	mu      sync.Mutex
	created []bool
	active  int
	expired int
	ops     []string
	sizes   []string
}

func (r *recordingMetrics) SessionCreated(ok bool) {
	r.mu.Lock()
	r.created = append(r.created, ok)
	r.mu.Unlock()
}
func (r *recordingMetrics) ActiveSessions(d int) { r.mu.Lock(); r.active += d; r.mu.Unlock() }
func (r *recordingMetrics) SessionExpired()      { r.mu.Lock(); r.expired++; r.mu.Unlock() }
func (r *recordingMetrics) CryptoOperation(op, result string) {
	r.mu.Lock()
	r.ops = append(r.ops, op+"/"+result)
	r.mu.Unlock()
}
func (r *recordingMetrics) MessageSize(direction string, _ int) {
	r.mu.Lock()
	r.sizes = append(r.sizes, direction)
	r.mu.Unlock()
}

func TestManager_MetricsSinkReceivesEvents(t *testing.T) {
	mgr := NewManager()
	defer func() { _ = mgr.Close() }()
	rec := &recordingMetrics{}
	mgr.SetMetrics(rec)

	sess, err := mgr.CreateSession("metrics-1", []byte("seed-seed-seed-seed-seed-seed-32"))
	require.NoError(t, err)
	_, err = mgr.CreateSession("metrics-1", []byte("seed-seed-seed-seed-seed-seed-32"))
	require.Error(t, err, "duplicate id is a failed creation")

	ct, err := sess.Encrypt([]byte("hello"))
	require.NoError(t, err)
	_, err = sess.Decrypt(ct)
	require.NoError(t, err)
	_, err = sess.Decrypt(ct)
	require.Error(t, err, "replay")

	mgr.RemoveSession("metrics-1")

	rec.mu.Lock()
	defer rec.mu.Unlock()
	assert.Equal(t, []bool{true, false}, rec.created)
	assert.Equal(t, 0, rec.active, "+1 on create, -1 on remove")
	assert.Equal(t, []string{"encrypt/success", "decrypt/success", "decrypt/failure"}, rec.ops)
	assert.Equal(t, []string{"encrypted", "decrypted"}, rec.sizes)
}

func TestSession_DefaultsToNopMetrics(t *testing.T) {
	s, err := NewSecureSession("nop", []byte("seed-seed-seed-seed-seed-seed-32"), Config{})
	require.NoError(t, err)
	ct, err := s.Encrypt([]byte("x"))
	require.NoError(t, err)
	_, err = s.Decrypt(ct)
	require.NoError(t, err)

	// Zero-value sessions (from a pool) must not panic either.
	var zero SecureSession
	zero.metricsSink().CryptoOperation("encrypt", "success")

	mgr := NewManager()
	defer func() { _ = mgr.Close() }()
	mgr.SetMetrics(nil)
	_, err = mgr.CreateSession("nop-2", []byte("seed-seed-seed-seed-seed-seed-32"))
	require.NoError(t, err)
}
