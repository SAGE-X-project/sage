package registry011

// The optional observer belongs to the original reader, not to a decoded
// block or a producer's reported work vector. Legacy DATA callers leave it
// nil. An owned collector must bind it before parsing any retained content.
// These are logical primitive observations; they are not a CPU, allocator,
// resident-memory or complete acquisition-cost certificate.
func (b *replayBudget) decodeObserved(raw []byte) bool {
	return b != nil && (b.observer == nil || b.observer.event(providerCodecEvent, uint64(len(raw))))
}

func (b *replayBudget) encodePreflight(maximum int) bool {
	return b != nil && maximum > 0 && maximum <= b.limits.MaxEvidenceBytes &&
		(b.observer == nil || b.observer.preflight(providerEncodeEvent, uint64(maximum)))
}

// The pinned canonical binary decoder cannot expand its successful binary
// re-encoding beyond the original input. Check that bound again before
// recording the actual output. It is not a caller-selected execution upper.
func (b *replayBudget) encodedObserved(raw []byte, maximum int) bool {
	if b == nil || len(raw) > maximum {
		if b != nil && b.observer != nil {
			b.observer.refuse()
		}
		return false
	}
	return b.observer == nil || b.observer.event(providerEncodeEvent, uint64(len(raw)))
}

func (b *replayBudget) equalObserved(a, c []byte) bool {
	return b != nil && providerReplayEqual(a, c, b.observer)
}

func (b *replayBudget) blockObserved() bool {
	return b != nil && (b.observer == nil || b.observer.add(candidateWorkVector{3: 1}))
}

func (b *replayBudget) transactionObserved() bool {
	return b != nil && (b.observer == nil || b.observer.add(candidateWorkVector{4: 1}))
}

func (b *replayBudget) frameObserved() bool {
	return b != nil && (b.observer == nil || b.observer.add(candidateWorkVector{5: 1}))
}

func (b *replayBudget) logsObserved(count int) bool {
	return b != nil && count >= 0 && (b.observer == nil || b.observer.add(candidateWorkVector{6: uint64(count)}))
}
