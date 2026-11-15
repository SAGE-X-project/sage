package registry011

import "context"

// At most eight pending objects, seven pending controls (including the
// temporary HEAD), and the two files of the unfinished member attempt.
const providerHistoryRecoveryTailInputs = 8 + 7 + 2

type providerHistoryRecoveryTailInput struct {
	kind                         providerHistoryRecoveryFileKind
	index                        int
	state                        providerHistoryRecoveryInputState
	bytes                        uint64
	digest                       string
	whole, optional, absenceOnly bool
}

type providerHistoryRecoveryTailData struct {
	entries                       [providerHistoryRecoveryTailInputs]providerHistoryRecoveryTailInput
	count                         int
	incomplete, unverifiedPartial bool
	unacknowledged                bool
	attemptResult                 string
}

func (*providerHistoryRecoveryTailData) UnmarshalJSON([]byte) error { return errProviderIntegration }

// This plan is constructed only from the original pending checkpoint or
// unfinished capture seal. It supplies no caller filename/root/resolver.
type providerHistoryRecoveryTailPlan struct {
	checkpoint                      *providerHistoryRecoveryCheckpoint
	kind                            providerHistoryRecoveryFileKind
	index                           int
	raw, domain, digest             string
	bytes                           uint64
	optional, mustWhole, mustAbsent bool
}

func (m *providerHistoryRecoveryTailPayloads) read(ctx context.Context, p providerHistoryRecoveryTailPlan) (*providerHistoryRecoveryInput, bool, error) {
	r := m.recovery
	if m.data.count < 0 || m.data.count >= len(m.data.entries) || !p.mustAbsent && (p.bytes == 0 || p.bytes > providerHistoryOriginalPackBytes ||
		p.raw != "" && (uint64(len(p.raw)) != p.bytes || p.domain != "" || p.digest != "") || p.raw == "" && !lowerHash(p.digest)) {
		return nil, false, errProviderIntegration
	}
	input, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: p.checkpoint, kind: p.kind, index: p.index})
	if err != nil || input == nil || input.state == providerHistoryRecoveryChangedInput || p.mustAbsent && input.state != providerHistoryRecoveryMissingInput {
		return nil, false, errProviderIntegration
	}
	entry := providerHistoryRecoveryTailInput{kind: p.kind, index: p.index, state: input.state, bytes: uint64(len(input.raw)), optional: p.optional, absenceOnly: p.mustAbsent}
	switch input.state {
	case providerHistoryRecoveryMissingInput:
		if p.mustWhole {
			return nil, false, errProviderIntegration
		}
		if !p.optional && !p.mustAbsent {
			m.data.incomplete = true
		}
	case providerHistoryRecoveryWholeInput, providerHistoryRecoveryPartialInput:
		if p.mustAbsent || entry.bytes > p.bytes {
			return nil, false, errProviderIntegration
		}
		entry.digest = providerReplayDigest(p.domain, input.raw, r.meter)
		if entry.digest == "" {
			return nil, false, errProviderIntegration
		}
		if entry.bytes < p.bytes {
			if p.mustWhole {
				return nil, false, errProviderIntegration
			}
			entry.state, m.data.incomplete = providerHistoryRecoveryPartialInput, true
			if p.raw != "" {
				if !providerReplayEqual(input.raw, providerReplayBytes(p.raw[:len(input.raw)], r.meter), r.meter) {
					return nil, false, errProviderIntegration
				}
			} else if entry.bytes != 0 {
				// A whole-object hash cannot authenticate a short prefix.
				// Preserve the observed hash as explicitly unverified DATA.
				m.data.unverifiedPartial = true
			}
		} else {
			if input.state != providerHistoryRecoveryWholeInput || p.raw != "" && !providerReplayEqual(input.raw, providerReplayBytes(p.raw, r.meter), r.meter) ||
				p.raw == "" && !providerReplayTextEqual(entry.digest, p.digest, r.meter) {
				return nil, false, errProviderIntegration
			}
			entry.whole = true
		}
	default:
		return nil, false, errProviderIntegration
	}
	if !r.meter.event(providerCopyEvent, 1) || !r.inputReadyLocked(ctx) {
		return nil, false, errProviderIntegration
	}
	m.data.entries[m.data.count] = entry
	m.data.count++
	return input, entry.whole, nil
}

func (r *providerHistoryRecoveryOwner) tailDataEqualLocked(a, b *providerHistoryRecoveryTailData) bool {
	if a == nil || b == nil || !r.meter.event(providerCompareEvent, 0) || a.count != b.count || a.incomplete != b.incomplete || a.unverifiedPartial != b.unverifiedPartial ||
		a.unacknowledged != b.unacknowledged || !providerReplayTextEqual(a.attemptResult, b.attemptResult, r.meter) {
		return false
	}
	for i := range a.entries {
		x, y := a.entries[i], b.entries[i]
		if !r.meter.event(providerCompareEvent, 0) || x.kind != y.kind || x.index != y.index || x.state != y.state || x.bytes != y.bytes ||
			x.whole != y.whole || x.optional != y.optional || x.absenceOnly != y.absenceOnly || !providerReplayTextEqual(x.digest, y.digest, r.meter) {
			return false
		}
	}
	return true
}
