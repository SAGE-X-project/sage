package registry011

import (
	"encoding/json"
	"testing"
)

func historyInputBudget(t *testing.T, envelope providerHistoryInputEnvelope) *providerHistoryInputBudget {
	t.Helper()
	b, err := newProviderHistoryInputBudget(admissionTestDigest("a"), 0, envelope)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func historyConsume(b *providerHistoryInputBudget, claim providerHistoryInputUses) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.consumeLocked(claim)
}

func TestProviderHistoryInputUpperPreservesEventMetric(t *testing.T) {
	for kind := providerReadEvent; kind <= providerCompareEvent; kind++ {
		var envelope providerHistoryInputEnvelope
		var expected candidateWorkVector
		bytes := uint64(3)
		if kind == providerClockEvent || kind == providerSyncEvent || kind == providerRenameEvent {
			bytes = 0
		}
		envelope[kind] = providerHistoryInputRow{bytes, 4 * bytes, 4}
		for i := 0; i < 4; i++ {
			work, ok := providerEventVector(kind, bytes)
			if !ok {
				t.Fatal("existing event metric refused finite data")
			}
			for j, n := range work {
				expected[j] += n
			}
		}
		if got, ok := envelope.upper(); !ok || got != expected {
			t.Fatalf("kind %d changed the original event metric: %v, %v", kind, got, expected)
		}
	}
	for _, kind := range []providerWorkEvent{providerCodecEvent, providerEncodeEvent} {
		var envelope providerHistoryInputEnvelope
		envelope[kind] = providerHistoryInputRow{3, 6, 4}
		upper, ok := envelope.upper()
		if !ok || upper[10] != 22 {
			t.Fatal("correlated prospective upper changed")
		}
		var actual uint64
		for _, n := range []uint64{0, 1, 3, 2} {
			work, _ := providerEventVector(kind, n)
			actual += work[10]
		}
		if actual != 18 || actual > upper[10] {
			t.Fatal("zero-input invocations or quadratic events were omitted")
		}
	}
}

func TestProviderHistoryInputUpperRejectsIncompleteOrOverflowingRows(t *testing.T) {
	for _, row := range []struct {
		name string
		kind providerWorkEvent
		row  providerHistoryInputRow
	}{
		{"unknown-kind", providerCompareEvent + 1, providerHistoryInputRow{1, 1, 1}},
		{"unbounded-width", providerReadEvent, providerHistoryInputRow{32<<20 + 1, 1, 1}},
		{"width-without-call", providerReadEvent, providerHistoryInputRow{1, 0, 0}},
		{"bytes-without-call", providerReadEvent, providerHistoryInputRow{0, 1, 0}},
		{"bytes-without-width", providerReadEvent, providerHistoryInputRow{0, 1, 1}},
		{"more-than-call-widths", providerReadEvent, providerHistoryInputRow{2, 5, 2}},
		{"clock-width", providerClockEvent, providerHistoryInputRow{1, 0, 1}},
		{"clock-bytes", providerClockEvent, providerHistoryInputRow{0, 1, 1}},
		{"sync-width", providerSyncEvent, providerHistoryInputRow{1, 0, 1}},
		{"rename-bytes", providerRenameEvent, providerHistoryInputRow{0, 1, 1}},
	} {
		t.Run(row.name, func(t *testing.T) {
			if providerHistoryInputRowValid(row.kind, row.row) {
				t.Fatal("invalid row was accepted")
			}
			if row.kind <= providerCompareEvent {
				var envelope providerHistoryInputEnvelope
				envelope[row.kind] = row.row
				if _, ok := envelope.upper(); ok {
					t.Fatal("invalid row reached a numeric upper")
				}
			}
		})
	}
	for _, row := range []struct {
		kind providerWorkEvent
		row  providerHistoryInputRow
	}{
		{providerCodecEvent, providerHistoryInputRow{32 << 20, ^uint64(0), ^uint64(0)}},
		{providerEncodeEvent, providerHistoryInputRow{2, ^uint64(0) / 2, ^uint64(0)}},
		{providerReadEvent, providerHistoryInputRow{1, ^uint64(0), ^uint64(0)}},
	} {
		var envelope providerHistoryInputEnvelope
		envelope[row.kind] = row.row
		if _, ok := envelope.upper(); ok {
			t.Fatal("overflow became a smaller upper")
		}
	}
	var empty, overflow providerHistoryInputEnvelope
	if _, ok := empty.upper(); ok {
		t.Fatal("missing work became an active envelope")
	}
	overflow[providerClockEvent] = providerHistoryInputRow{0, 0, ^uint64(0)}
	overflow[providerSyncEvent] = providerHistoryInputRow{0, 0, 1}
	if _, ok := overflow.upper(); ok {
		t.Fatal("cross-class sum overflow was accepted")
	}
	for _, row := range []providerHistoryInputRow{{0, 0, 2}, {3, 6, 2}, {3, 5, 2}, {32 << 20, ^uint64(0), ^uint64(0)}} {
		if !providerHistoryInputRowValid(providerReadEvent, row) {
			t.Fatal("valid aggregate shape refused before arithmetic")
		}
	}
}

func TestProviderHistoryInputBudgetConsumesEveryInvocationWithoutRefund(t *testing.T) {
	var envelope providerHistoryInputEnvelope
	envelope[providerReadEvent] = providerHistoryInputRow{8, 20, 4}
	b := historyInputBudget(t, envelope)
	// The constructor copies its fixed-width array, never a caller slice/map.
	envelope[providerReadEvent].bytes = 100
	for _, use := range []providerHistoryInputUse{{8, 1}, {12, 2}, {0, 1}} {
		var claim providerHistoryInputUses
		claim[providerReadEvent] = use
		if !historyConsume(b, claim) {
			t.Fatal("prepaid finite input refused")
		}
	}
	if b.used[providerReadEvent] != (providerHistoryInputUse{20, 4}) || b.envelope[providerReadEvent].bytes != 20 {
		t.Fatal("an input or empty invocation was refunded")
	}
	prior := b.used
	var extra providerHistoryInputUses
	extra[providerReadEvent].calls = 1
	if historyConsume(b, extra) || !b.failed || b.used != prior || historyConsume(b, extra) {
		t.Fatal("exhaustion reset the instance or earlier charges")
	}
}

func TestProviderHistoryInputBudgetRejectsInvalidClaimsAtomically(t *testing.T) {
	var envelope providerHistoryInputEnvelope
	envelope[providerReadEvent] = providerHistoryInputRow{8, 8, 2}
	for _, row := range []struct {
		name  string
		kind  providerWorkEvent
		claim providerHistoryInputUse
	}{
		{"no-invocations", providerReadEvent, providerHistoryInputUse{}},
		{"bytes-without-call", providerReadEvent, providerHistoryInputUse{1, 0}},
		{"per-input-width", providerReadEvent, providerHistoryInputUse{9, 1}},
		{"aggregate-bytes", providerReadEvent, providerHistoryInputUse{9, 2}},
		{"invocation-count", providerReadEvent, providerHistoryInputUse{1, 3}},
		{"inactive-kind", providerHashEvent, providerHistoryInputUse{0, 1}},
	} {
		t.Run(row.name, func(t *testing.T) {
			b := historyInputBudget(t, envelope)
			var claim providerHistoryInputUses
			claim[row.kind] = row.claim
			if historyConsume(b, claim) || !b.failed || b.used != (providerHistoryInputUses{}) {
				t.Fatal("invalid claim partially consumed or preserved a usable instance")
			}
		})
	}
	// A later invalid class cannot leave an earlier row partly consumed.
	b := historyInputBudget(t, envelope)
	var mixed providerHistoryInputUses
	mixed[providerReadEvent] = providerHistoryInputUse{1, 1}
	mixed[providerHashEvent] = providerHistoryInputUse{0, 1}
	if historyConsume(b, mixed) || b.used != (providerHistoryInputUses{}) {
		t.Fatal("partial multi-class debit escaped")
	}
}

func TestProviderHistoryInputBudgetRejectsMissingOrCopiedState(t *testing.T) {
	var envelope providerHistoryInputEnvelope
	envelope[providerReadEvent] = providerHistoryInputRow{8, 8, 1}
	for _, row := range []struct {
		program string
		role    uint8
		e       providerHistoryInputEnvelope
	}{
		{"", 0, envelope},
		{"0X" + admissionTestDigest("a")[2:], 0, envelope},
		{admissionTestDigest("a"), 2, envelope},
		{admissionTestDigest("a"), 0, providerHistoryInputEnvelope{}},
	} {
		if b, err := newProviderHistoryInputBudget(row.program, row.role, row.e); err == nil || b != nil {
			t.Fatal("unbound or incomplete input state accepted")
		}
	}
	b := historyInputBudget(t, envelope)
	var claim providerHistoryInputUses
	claim[providerReadEvent] = providerHistoryInputUse{1, 1}
	for _, invalid := range []*providerHistoryInputBudget{nil, {}, {owner: b}} {
		if invalid.consumeLocked(claim) {
			t.Fatal("missing or copied identity spent another instance")
		}
	}
	for _, target := range []any{&envelope, b} {
		if json.Unmarshal([]byte(`{}`), target) == nil {
			t.Fatal("JSON reconstructed a guard")
		}
	}
	// Fixed scalar snapshots, not live host mutation or replacement callbacks.
	for _, state := range []string{"row", "upper", "program", "role", "used-bytes", "used-calls"} {
		x := historyInputBudget(t, envelope)
		switch state {
		case "row":
			x.envelope[providerReadEvent].width = 32<<20 + 1
		case "upper":
			x.upper[10]++
		case "program":
			x.program = ""
		case "role":
			x.role = 2
		case "used-bytes":
			x.used[providerReadEvent].bytes = 9
		case "used-calls":
			x.used[providerReadEvent].calls = 2
		}
		if historyConsume(x, claim) || !x.failed {
			t.Fatal("invalid finite state remained usable")
		}
	}
}
