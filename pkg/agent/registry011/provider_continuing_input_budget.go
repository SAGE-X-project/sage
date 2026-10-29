package registry011

import "sync"

// A disjoint byte-event envelope for a future whole-program owner. It is
// prospective DATA, not a retained hold, an ACK, a measurement or permission.
// Each invocation is charged again, including repeated original reads. The
// enclosing program must derive these rows and fund ALL other work before H0.
type providerHistoryInputRow struct{ width, bytes, calls uint64 }
type providerHistoryInputEnvelope [providerCompareEvent + 1]providerHistoryInputRow
type providerHistoryInputUse struct{ bytes, calls uint64 }
type providerHistoryInputUses [providerCompareEvent + 1]providerHistoryInputUse

func (*providerHistoryInputEnvelope) UnmarshalJSON([]byte) error { return ErrConfiguration }

// This is the existing logical event metric with a correlated input upper:
// sum(n*n+1) <= width*sum(n)+calls. It does not discount any old child quote.
// Width is fixed for the lifetime of this instance, never a reported actual.
func (e providerHistoryInputEnvelope) upper() (candidateWorkVector, bool) {
	var total candidateWorkVector
	active := false
	for kind, row := range e {
		if !providerHistoryInputRowValid(providerWorkEvent(kind), row) {
			return candidateWorkVector{}, false
		}
		if row.calls == 0 {
			continue
		}
		active = true
		work, ok := providerHistoryInputRowUpper(providerWorkEvent(kind), row)
		if !ok {
			return candidateWorkVector{}, false
		}
		for i, n := range work {
			next, fits := providerUpperSum(total[i], n)
			if !fits {
				return candidateWorkVector{}, false
			}
			total[i] = next
		}
	}
	return total, active
}

func providerHistoryInputRowValid(kind providerWorkEvent, row providerHistoryInputRow) bool {
	if kind > providerCompareEvent || row.width > 32<<20 {
		return false
	}
	if row.calls == 0 {
		return row.bytes == 0 && row.width == 0
	}
	if kind == providerClockEvent || kind == providerSyncEvent || kind == providerRenameEvent {
		return row.width == 0 && row.bytes == 0
	}
	if row.width == 0 {
		return row.bytes == 0
	}
	// ceil(bytes/width) <= calls avoids an overflowing width*calls.
	return row.bytes/row.width <= row.calls && (row.bytes%row.width == 0 || row.bytes/row.width < row.calls)
}

func providerHistoryInputRowUpper(kind providerWorkEvent, row providerHistoryInputRow) (candidateWorkVector, bool) {
	var work candidateWorkVector
	units := row.bytes
	if kind == providerCodecEvent || kind == providerEncodeEvent {
		var ok bool
		units, ok = providerUpperProduct(row.width, row.bytes)
		if !ok {
			return work, false
		}
	}
	units, ok := providerUpperSum(units, row.calls)
	if !ok {
		return work, false
	}
	if kind == providerCodecEvent {
		work[2] = row.bytes
	}
	if kind == providerWriteEvent {
		work[9] = row.bytes
	}
	work[10] = units
	return work, true
}

// The instance serializes the guard and the input effect. There is no refund,
// resize, restore or reset. Recreating it does NOT recreate operational funds:
// no current owner, meter, Source or transport accepts this candidate object.
// The future program must own one original instance per independent role.
type providerHistoryInputBudget struct {
	mu       sync.Mutex
	owner    *providerHistoryInputBudget
	program  string
	role     uint8
	envelope providerHistoryInputEnvelope
	upper    candidateWorkVector
	used     providerHistoryInputUses
	failed   bool
}

func (*providerHistoryInputBudget) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistoryInputBudget(program string, role uint8, envelope providerHistoryInputEnvelope) (*providerHistoryInputBudget, error) {
	upper, ok := envelope.upper()
	if !ok || !lowerHash(program) || role > 1 {
		return nil, ErrConfiguration
	}
	b := &providerHistoryInputBudget{program: program, role: role, envelope: envelope, upper: upper}
	b.owner = b
	return b, nil
}

// Called only with the guard mutex held. A claim is a closed source upper
// consumed BEFORE the corresponding input. It is never reported actual work.
// A multi-invocation claim must separately prove every input <= fixed width;
// an aggregate byte total alone cannot establish that property.
func (b *providerHistoryInputBudget) consumeLocked(claim providerHistoryInputUses) bool {
	if b == nil || b.owner != b || b.failed {
		return false
	}
	derived, ok := b.envelope.upper()
	if !ok || derived != b.upper || !lowerHash(b.program) || b.role > 1 {
		b.failed = true
		return false
	}
	active := false
	next := b.used
	for kind, use := range claim {
		row := b.envelope[kind]
		if use.calls == 0 && use.bytes == 0 {
			continue
		}
		active = true
		if use.calls == 0 || !providerHistoryInputRowValid(providerWorkEvent(kind), providerHistoryInputRow{row.width, use.bytes, use.calls}) ||
			!candidateCapacity(0, b.used[kind].bytes, use.bytes, row.bytes) ||
			!candidateCapacity(0, b.used[kind].calls, use.calls, row.calls) {
			b.failed = true
			return false
		}
		next[kind].bytes += use.bytes
		next[kind].calls += use.calls
	}
	if !active {
		b.failed = true
		return false
	}
	b.used = next
	return true
}
