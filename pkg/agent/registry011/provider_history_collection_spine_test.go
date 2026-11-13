package registry011

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

func historyCollectionSpineTestHeader() FinalizedHeader {
	return FinalizedHeader{Hash: "0x" + strings.Repeat("f", 64), ParentHash: "0x" + strings.Repeat("a", 64), Number: ^uint64(0), Timestamp: 1<<53 - 1}
}

// Independent old map shape. The generic oracle adds format/kind to its own
// map, so callers must never share that mutable map with another oracle.
func historyCollectionSpineTestValue(previous string, ordinal int, header FinalizedHeader) map[string]any {
	return map[string]any{"previous": previous, "ordinal": strconv.Itoa(ordinal), "header": map[string]any{
		"hash": header.Hash, "parent_hash": header.ParentHash,
		"number": strconv.FormatUint(header.Number, 10), "timestamp": strconv.FormatUint(header.Timestamp, 10)}}
}

func TestProviderHistoryCollectionSpineMatchesOriginalBytesAndCumulativeEvents(t *testing.T) {
	for _, ordinal := range []int{0, 1, (1 << 20) - 1} {
		previous := ""
		if ordinal != 0 {
			previous = "0x" + strings.Repeat("d", 64)
		}
		header := historyCollectionSpineTestHeader()
		before := header
		value := historyCollectionSpineTestValue(previous, ordinal, header)
		value["format"], value["kind"] = providerHistoryCollectionStateDomain, "spine"
		raw := providerIntegrationJSON(value)
		if len(raw) == 0 || len(raw) > 402 || ordinal == (1<<20)-1 && len(raw) != 402 {
			t.Fatal("independent full-uint64/ASCII closed schema maximum", ordinal, len(raw))
		}
		want := providerCandidateHash(providerHistoryCollectionStateDomain, raw)
		// Fund the OLD generic8192 preflight before either call. The same
		// cumulative observer is never reset or replaced between the two.
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: 1 << 60})
		old := providerHistoryCollectionValueDigestObserved("spine", historyCollectionSpineTestValue(previous, ordinal, header), m)
		first := m.actual
		got := providerHistoryCollectionSpineDigestObserved(previous, ordinal, header, m)
		n := uint64(len(raw))
		actual := candidateWorkVector{10: 6 + n*n + 1 + n + uint64(len(providerHistoryCollectionStateDomain)) + 2}
		if old != want || got != want || m.failed || first != actual || header != before {
			t.Fatal("typed spine changed old canonical bytes, digest or actual events", ordinal, old, got, first, actual)
		}
		for i, units := range actual {
			if m.actual[i] != 2*units {
				t.Fatal("typed spine borrowed a cache or renewed the observer", i, m.actual, actual)
			}
		}
	}
}

func TestProviderHistoryCollectionSpineRefusesInvalidTypedDataBeforeWork(t *testing.T) {
	for _, mode := range []string{"nil", "negative", "large", "first-previous", "later-empty", "later-invalid", "hash", "parent", "timestamp"} {
		header, previous, ordinal := historyCollectionSpineTestHeader(), "", 0
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: 1 << 60})
		switch mode {
		case "nil":
			m = nil
		case "negative":
			ordinal = -1
		case "large":
			ordinal = 1 << 20
		case "first-previous":
			previous = "0x" + strings.Repeat("d", 64)
		case "later-empty":
			ordinal = 1
		case "later-invalid":
			ordinal, previous = 1, "0x"+strings.Repeat("D", 64)
		case "hash":
			header.Hash = "0x" + strings.Repeat("F", 64)
		case "parent":
			header.ParentHash = "short"
		case "timestamp":
			header.Timestamp = 1 << 53
		}
		if got := providerHistoryCollectionSpineDigestObserved(previous, ordinal, header, m); got != "" ||
			m != nil && (m.actual != (candidateWorkVector{}) || m.failed) {
			t.Fatal("detached invalid typed DATA reached an event", mode, got)
		}
		if m != nil && providerHistoryCollectionSpineDigestObserved("", 0, historyCollectionSpineTestHeader(), m) == "" {
			t.Fatal("early DATA refusal spent an unentered observer", mode)
		}
	}
}

func TestProviderHistoryCollectionSpineKeepsExactPreflightAndGenericLimits(t *testing.T) {
	for _, shortage := range []uint64{0, 1} {
		// CopyEvent(5) spends6 before Encode preflight(512) needs262145.
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: 262151 - shortage})
		previous := "0x" + strings.Repeat("d", 64)
		got := providerHistoryCollectionSpineDigestObserved(previous, (1<<20)-1, historyCollectionSpineTestHeader(), m)
		if shortage == 0 {
			value := historyCollectionSpineTestValue(previous, (1<<20)-1, historyCollectionSpineTestHeader())
			value["format"], value["kind"] = providerHistoryCollectionStateDomain, "spine"
			raw := providerIntegrationJSON(value)
			if len(raw) != 402 || got != providerCandidateHash(providerHistoryCollectionStateDomain, raw) || m.failed || m.actual != (candidateWorkVector{10: 162056}) {
				t.Fatal("exact typed preflight was confused with actual output cost", got, m.actual)
			}
		} else {
			if got != "" || !m.failed || m.actual != (candidateWorkVector{10: 6}) {
				t.Fatal("one-short preflight must retain only the original copy prefix", got, m.actual)
			}
			if retry := providerHistoryCollectionSpineDigestObserved(previous, (1<<20)-1, historyCollectionSpineTestHeader(), m); retry != "" || m.actual != (candidateWorkVector{10: 6}) {
				t.Fatal("failed typed preflight revived its original observer", retry, m.actual)
			}
		}
	}
	for _, kind := range []string{"effects", "cursor"} {
		copyUnits := uint64(len(kind)) + 1
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: copyUnits + 262145})
		if got := providerHistoryCollectionValueDigestObserved(kind, map[string]any{"previous": ""}, m); got != "" || !m.failed || m.actual != (candidateWorkVector{10: copyUnits}) {
			t.Fatal("a generic non-spine value silently adopted the smaller ceiling", kind, got, m.actual)
		}
	}
}

func TestProviderHistoryCollectionSpineKeepsOriginalReaderRoles(t *testing.T) {
	t.Run("collection", func(t *testing.T) {
		o := historyNativeInputFixture(t, true, 0)
		m, c := o.meter, o.coordinator.collection
		if c == nil || !c.initial() || c == o.custodian.collection || len(c.spine) != 1 {
			t.Fatal("genuine independent initial original cursor")
		}
		before, held, upper, reads := m.actual, o.store.stored, m.upper, o.store.clock.read
		root := c.commitmentObserved(m)
		first := historyEventQueueDelta(t, m, before)
		if !lowerHash(root) || root != c.commitmentObserved(m) || m.failed || o.store.stored != held || m.upper != upper || o.store.clock.read != reads {
			t.Fatal("original full cursor root changed or renewed its hold", root)
		}
		for i, units := range first {
			if m.actual[i] != before[i]+2*units {
				t.Fatal("full original cursor bypassed or cached a repeated interpretation", i)
			}
		}
		// This original UNIT reader retains its full chain/input/semantic
		// path. It is not a complete collection S or actual Registry run.
	})
	t.Run("terminal-inspection", func(t *testing.T) {
		o, _ := historyMemberDispatchFixture(t, 1)
		s, r, producer := o.store, o.parent.recovery, o.meter
		s.mu.Lock()
		o.failLocked()
		s.mu.Unlock()
		if err := r.beginInspectionCandidate(context.Background(), o); err != nil {
			t.Fatal("genuine original read-only recovery lane", err)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		defer r.stopLocked(true)
		start, old, reads := r.meter.actual, producer.actual, s.clock.read
		header := historyCollectionSpineTestHeader()
		want := providerHistoryCollectionValueDigestObserved("spine", historyCollectionSpineTestValue("", 0, header), r.meter)
		first := historyEventQueueDelta(t, r.meter, start)
		got := providerHistoryCollectionSpineDigestObserved("", 0, header, r.meter)
		if !lowerHash(want) || got != want || r.meter.failed || !o.parent.failed || producer.actual != old || s.clock.read != reads {
			t.Fatal("typed read-only DATA required or revived a healthy producer", want, got)
		}
		for i, units := range first {
			if r.meter.actual[i] != start[i]+2*units {
				t.Fatal("inspection changed original actual work", i)
			}
		}
	})
}
