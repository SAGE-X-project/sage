package guard010

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// Guard framing is 0.10.0, so a 0.11.0 HPKE endpoint is refused before any
// handshake: running 0.10.0 intents over a 0.11.0 session would be an
// implicit mixed profile.
func TestMCPConnectionRefuses011Endpoint(t *testing.T) {
	c := &setupRegistry{}
	j, err := registry010.OpenJournal(filepath.Join(t.TempDir(), "registry"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Close() })
	g, err := registry010.NewGate(registry010.Config{Source: "setup-test", Registry: "web:agent.example", Network: "local", Profile: "0.11.0"}, c, c, j)
	if err != nil {
		t.Fatal(err)
	}
	r, err := hpke.OpenReplayJournal010(filepath.Join(t.TempDir(), "replay"), true, c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	e, err := hpke.NewCompletionEndpoint011(setupAlice, setupAlice+"#signing-1", bytes.Repeat([]byte{1}, 32), nil, g, c, r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.Close)
	cfg := mcpConnectionConfig{endpoint: e, initiator: true, recipient: setupBob, recipientKey: setupBob + "#signing-1", ttl: 300}
	if _, err := establishMCPConnection(context.Background(), nil, cfg); !errors.Is(err, ErrInvalid) {
		t.Fatalf("0.11.0 endpoint accepted: %v", err)
	}
	// The endpoint was never used: refusal happened before the handshake.
	if err := e.BindHTTP("https://agent.example/mcp"); err != nil {
		t.Fatalf("endpoint was used before the refusal: %v", err)
	}
}
