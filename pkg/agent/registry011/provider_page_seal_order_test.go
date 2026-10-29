package registry011

import (
	"strings"
	"testing"
)

func TestProviderPageSealInventoryOrderUsesItsOwnFiniteObserver(t *testing.T) {
	t.Parallel()
	s, g := providerPageSealReadyFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := newProviderPageSealCandidateLocked(s)
	if err != nil {
		t.Fatal("normal page preparation", err)
	}
	defer o.failLocked()
	m := g.meter
	entries := make([]providerIntegrationObjectRef, 14)
	for n := range entries {
		entries[n] = providerIntegrationObjectRef{"response", "0x" + strings.Repeat(string("0123456789abcd"[13-n]), 64), 1}
	}
	before, stored, main := m.actual, s.stored, s.frontierFiles.pin
	if !providerInventoryOrderObserved(entries, m) {
		t.Fatal("finite page inventory order")
	}
	const comparison = uint64(76 + 76 + 151)
	if m.actual[10]-before[10] != (14*13/2)*comparison || m.actual[2] != before[2] {
		t.Fatal("page sort omitted the finite key-copy/comparison graph")
	}
	for n := 1; n < len(entries); n++ {
		if providerIntegrationRefKey(entries[n-1]) >= providerIntegrationRefKey(entries[n]) {
			t.Fatal("canonical page inventory order")
		}
	}
	before = m.actual
	if !providerInventoryOrderObserved(entries, m) || m.actual[10]-before[10] != 13*comparison ||
		!providerInventoryOrderObserved(nil, m) || providerInventoryOrderObserved(make([]providerIntegrationObjectRef, 257), m) {
		t.Fatal("page insertion did not keep its exact finite bound")
	}
	// Finite unit observation DATA consumes the remaining allowance; it does
	// not enlarge the bound, mint an owner, or write any original evidence.
	m.actual[10] = m.upper[10] - 1
	if providerInventoryOrderObserved(entries, m) || !m.failed || s.stored != stored || s.frontierFiles.pin != main || s.last != nil {
		t.Fatal("one-unit allowance permitted an unfunded key or changed custody")
	}
}
