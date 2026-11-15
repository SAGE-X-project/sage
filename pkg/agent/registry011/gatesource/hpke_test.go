package gatesource

import (
	"bytes"
	"context"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type acceptReplay struct{}

func (acceptReplay) Reserve(hpke.Replay010) error { return nil }

type hpkeClock struct{ *clock }

func (c hpkeClock) Now() (registry010.Stamp, error) { return c.clock.Now() }

// A Gate built on this Source carries 0.11.0 authority: 0.11.0 HPKE endpoints
// accept it, 0.10.0 endpoints refuse it, and its reads keep working.
func TestGateServes011EndpointsOnly(t *testing.T) {
	c := &clock{mono: 1000}
	g, _ := newGate(t, present(fixture(t)), c)
	if g.Profile() != "0.11.0" {
		t.Fatalf("gate profile %q", g.Profile())
	}
	seed := bytes.Repeat([]byte{7}, 32)
	if _, err := hpke.NewCompletionEndpoint011(did, did+"#sig-2", seed, nil, g, hpkeClock{c}, acceptReplay{}); err != nil {
		t.Fatalf("0.11.0 endpoint refused a 0.11.0 Gate: %v", err)
	}
	if _, err := hpke.NewCompletionEndpoint010(did, did+"#sig-2", seed, nil, g, hpkeClock{c}, acceptReplay{}); err == nil {
		t.Fatal("0.10.0 endpoint accepted a 0.11.0 Gate")
	}
	if _, err := g.Observe(context.Background(), did); err != nil {
		t.Fatalf("0.11.0 snapshot refused by its own Gate: %v", err)
	}
}
