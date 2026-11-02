package registry011

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"testing"
)

// One genuine normal, package-owned candidate. Only finite in-memory header
// bytes are inspected after preparation; these UNITs neither dial an external
// endpoint nor construct a production cost witness.
func historyBufferedHeadFixture(t *testing.T) *providerHistoryGroupOwner {
	t.Helper()
	o, d := historyMemberDispatchFixture(t, 1)
	o.store.mu.Lock()
	attempt, _, cancel, err := o.prepareMemberCandidateLocked(context.Background(), d, 0)
	o.store.mu.Unlock()
	if err != nil || attempt == nil || cancel == nil {
		t.Fatal("normal owned member preparation", err)
	}
	t.Cleanup(func() {
		cancel()
		o.store.mu.Lock()
		defer o.store.mu.Unlock()
		o.failLocked()
	})
	return o
}

func TestProviderBufferedHeadKeepsBytesEventsAndReaderPosition(t *testing.T) {
	o := historyBufferedHeadFixture(t)
	for _, row := range []struct {
		name, wire, head, rest string
		maximum               int
		units                 uint64
		failed                bool
	}{
		{"terminator", "\r\n\r\nTAIL", "\r\n\r\n", "TAIL", 64, 25, false},
		{"one-prefix", "a\r\n\r\nTAIL", "a\r\n\r\n", "TAIL", 64, 38, false},
		{"three-prefix", "abc\r\n\r\nTAIL", "abc\r\n\r\n", "TAIL", 64, 64, false},
		{"buffer-boundary", "HTTP/1.1 200 OK\r\n\r\nTAIL", "HTTP/1.1 200 OK\r\n\r\n", "TAIL", 64, 220, false},
		{"empty-eof", "", "", "", 64, 1, true},
		{"short-eof", "abc", "", "", 64, 13, true},
		{"long-eof", "abcd", "", "", 64, 26, true},
		{"over-limit", "abcdef", "", "ef", 3, 25, true},
		{"terminator-over-limit", "abc\r\n\r\nTAIL", "", "TAIL", 6, 64, true},
		{"terminator-at-limit", "abc\r\n\r\nTAIL", "abc\r\n\r\n", "TAIL", 7, 64, false},
		{"zero-limit", "abc", "", "bc", 0, 4, true},
		{"negative-limit", "abc", "", "abc", -1, 0, true},
	} {
		t.Run(row.name, func(t *testing.T) {
			o.store.mu.Lock()
			defer o.store.mu.Unlock()
			m := o.meter
			before, clocks, reads := m.actual, m.clocks, o.store.clock.read
			r := bufio.NewReaderSize(bytes.NewReader([]byte(row.wire)), 16)
			_, _ = r.Peek(1) // Fixture provisioning only; no byte is consumed.
			head, err := readProviderHeadObserved(r, row.maximum, m)
			rest, restErr := io.ReadAll(r)
			expected := before
			expected[10] += row.units
			if (err != nil) != row.failed || string(head) != row.head || string(rest) != row.rest || restErr != nil || m.actual != expected || m.failed || m.closed || m.clocks != clocks || o.store.clock.read != reads {
				t.Fatal("header bytes, literal logical work or reader position changed", err, string(head), string(rest), m.actual, expected)
			}
		})
	}
}

func TestProviderBufferedHeadKeepsExactInsufficientWorkPrefix(t *testing.T) {
	for _, row := range []struct {
		name     string
		room     uint64
		units    uint64
		consumed int
		failed   bool
	}{
		{"zero", 0, 0, 0, true},
		{"one", 1, 0, 0, true},
		{"read-only", 2, 2, 1, true},
		{"copy-short", 3, 2, 1, true},
		{"one-byte", 4, 4, 1, true},
		{"next-read-short", 5, 4, 1, true},
		{"three-bytes", 12, 12, 3, true},
		{"compare-short", 16, 16, 4, true},
		{"compare-one-short", 24, 16, 4, true},
		{"exact", 25, 25, 4, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			o := historyBufferedHeadFixture(t)
			o.store.mu.Lock()
			defer o.store.mu.Unlock()
			m := o.meter
			// Finite UNIT counter state, not a real cost claim or a reset.
			// Keep the exact selected owner and both original upper bounds.
			limit := min(m.upper[10], o.windowStart[10]+o.windowUpper[10])
			if limit < row.room || limit-row.room < m.actual[10] || o.firstPReading {
				t.Fatal("finite counter state cannot fit the original scope")
			}
			m.actual[10] = limit - row.room
			if !m.hasOwner() || !m.fitsOwnedWindow(candidateWorkVector{}) {
				t.Fatal("otherwise valid finite owned counter state")
			}
			before, clocks, reads := m.actual, m.clocks, o.store.clock.read
			wire := "\r\n\r\nTAIL"
			r := bufio.NewReaderSize(bytes.NewReader([]byte(wire)), 16)
			_, _ = r.Peek(1)
			if _, handled, complete, err := tryReadProviderBufferedHeadObserved(r, 64, nil, m); handled || complete || err != nil || m.actual != before || m.failed || r.Buffered() != len(wire) {
				t.Fatal("speculative whole-prefix check changed a refused subject", err)
			}
			head, err := readProviderHeadObserved(r, 64, m)
			rest, restErr := io.ReadAll(r)
			expected := before
			expected[10] += row.units
			if (err != nil) != row.failed || m.failed != row.failed || m.closed || m.actual != expected || string(rest) != wire[row.consumed:] || restErr != nil || m.clocks != clocks || o.store.clock.read != reads || !row.failed && string(head) != "\r\n\r\n" || row.failed && head != nil {
				t.Fatal("refusal moved its exact original byte/event prefix", err, m.actual, expected, string(rest))
			}
		})
	}
}

func TestProviderBufferedHeadRechecksOwnerOnEveryCall(t *testing.T) {
	o := historyBufferedHeadFixture(t)
	o.store.mu.Lock()
	defer o.store.mu.Unlock()
	m := o.meter
	newReader := func() *bufio.Reader {
		r := bufio.NewReaderSize(bytes.NewReader([]byte("\r\n\r\nTAIL")), 16)
		_, _ = r.Peek(1)
		return r
	}
	if head, err := readProviderHeadObserved(newReader(), 64, m); err != nil || string(head) != "\r\n\r\n" || m.failed {
		t.Fatal("normal first header", err)
	}
	before, clocks, reads := m.actual, m.clocks, o.store.clock.read
	o.parent.recovery.used = true // Inert UNIT state change between calls.
	r := newReader()
	if head, err := readProviderHeadObserved(r, 64, m); err == nil || head != nil || !m.failed || m.actual != before || m.clocks != clocks || o.store.clock.read != reads || r.Buffered() != 8 {
		t.Fatal("earlier positive result survived changed original ownership", err)
	}
}
