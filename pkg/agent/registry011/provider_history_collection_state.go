package registry011

import "strconv"

const providerHistoryCollectionStateDomain = "sage-provider-history-collection-state-v1"

// The closed typed spine value is at most402 canonical ASCII JSON bytes.
// This ceiling changes no encoded field, original item or hash domain.
const providerHistoryCollectionSpineBytes uint64 = 512

func (c *providerHistoryCollectionCursor) initial() bool {
	if !c.valid() || c.phase != "open" || c.completed != 0 || len(c.spine) != 1 || len(c.effects) != 0 ||
		c.effectNext != 0 || c.traces != 0 || c.native != 0 || c.historyNext != 0 || c.stopVersion != 0 || c.client != "" || c.genesis != (FinalizedHeader{}) {
		return false
	}
	if c.presentStops != 0 || len(c.pending) != 0 || c.pendingApplied != 0 || len(c.traceResults) != 0 || c.stopRecord != "" {
		return false
	}
	if c.semantic.blocks != 0 || c.semantic.items != 0 || c.semantic.windows != 0 || c.semantic.window != nil || c.semantic.confirmed.state.version != 0 || len(c.semantic.confirmed.claims) != 0 || len(c.semantic.confirmed.created) != 0 {
		return false
	}
	deployment, selected := c.selection.input.journal.Deployment.Number, c.selection.selected.Number
	if selected == deployment {
		return c.mode == "content" && c.low == selected && c.high == selected && c.next == selected
	}
	high := selected - 1
	low := max(deployment, high-min(high, c.selection.input.windowBlocks-1))
	return c.mode == "ancestry" && c.low == low && c.high == high && c.next == high
}

// This is a commitment to independently reconstructed DATA, not an original
// reader or an authorization. Every invocation re-encodes the selected input
// and every retained spine/effect value with the same original observer. The
// fold avoids a whole-spine metadata allocation; it does not remove its linear
// interpretation cost, its backing memory, or repeated-prefix costs.
func (c *providerHistoryCollectionCursor) commitmentObserved(meter *providerWorkMeter) string {
	if !c.valid() || meter == nil {
		return ""
	}
	journal := providerReplayMarshal(struct {
		Format string
		Config JournalConfig
	}{"sage-replay-journal-v1", c.selection.input.journal}, 64<<10, meter)
	if len(journal) == 0 {
		return ""
	}
	manifest := providerReplayCanonical(c.selection.input.object(providerReplayString(journal, meter)), candidateLedgerMetadataLimit, meter)
	if len(manifest) == 0 || !providerReplayTextEqual(providerReplayDigest("sage-provider-history-collection-manifest-v1", manifest, meter), c.selection.manifest, meter) {
		return ""
	}
	spine := ""
	for i, header := range c.spine {
		if !validHeader(header) || uint64(i) > c.selection.selected.Number || header.Number != c.selection.selected.Number-uint64(i) {
			return ""
		}
		if i > 0 {
			child := c.spine[i-1]
			if !providerReplayTextEqual(child.ParentHash, header.Hash, meter) || child.Timestamp <= header.Timestamp {
				return ""
			}
		}
		if header.Number == c.selection.input.journal.Deployment.Number && header != c.selection.input.journal.Deployment {
			return ""
		}
		spine = providerHistoryCollectionTypedSpineDigestObserved(spine, i, header, meter)
		if spine == "" {
			return ""
		}
	}
	effects := ""
	if len(c.effects) > 0 && !c.effectFactsObserved(meter) {
		return ""
	}
	for i, effect := range c.effects {
		effects = providerHistoryCollectionValueDigestObserved("effects", map[string]any{
			"previous": effects, "ordinal": strconv.Itoa(i), "block": effect.block, "transaction": effect.transaction,
			"number": strconv.FormatUint(effect.number, 10), "index": strconv.FormatUint(effect.index, 10),
		}, meter)
		if effects == "" {
			return ""
		}
	}
	semantic := c.semantic.commitmentObserved(meter)
	pending := c.pendingRootObserved(meter)
	stopRecord := ""
	if c.stopRecord != "" {
		if c.phase != "grants" || c.stopVersion == 0 {
			return ""
		}
		stopRecord = providerHistoryCollectionBytesRootObserved(c.stopRecord, 65536, meter)
		if stopRecord == "" {
			return ""
		}
	}
	if semantic == "" || pending == "" {
		return ""
	}
	return providerHistoryCollectionValueDigestObserved("cursor", map[string]any{
		"program": c.selection.program, "manifest": c.selection.manifest, "selected": providerHistoryCollectionHeaderObject(c.selection.selected),
		"phase": c.phase, "mode": c.mode, "low": strconv.FormatUint(c.low, 10), "high": strconv.FormatUint(c.high, 10), "next": strconv.FormatUint(c.next, 10),
		"completed_groups": strconv.FormatUint(c.completed, 10), "spine": spine, "spine_count": strconv.Itoa(len(c.spine)),
		"effects": effects, "effect_count": strconv.Itoa(len(c.effects)), "effect_next": strconv.Itoa(c.effectNext),
		"traces": strconv.FormatUint(c.traces, 10), "native": strconv.FormatUint(c.native, 10),
		"history_next": strconv.FormatUint(c.historyNext, 10), "stop_version": strconv.FormatUint(c.stopVersion, 10),
		"genesis": providerHistoryCollectionHeaderObject(c.genesis), "client": c.client,
		"semantic": semantic, "pending": pending, "stop_record": stopRecord, "present_stops": strconv.FormatUint(c.presentStops, 10),
	}, meter)
}

func providerHistoryCollectionHeaderObject(h FinalizedHeader) map[string]any {
	return map[string]any{"hash": h.Hash, "parent_hash": h.ParentHash,
		"number": strconv.FormatUint(h.Number, 10), "timestamp": strconv.FormatUint(h.Timestamp, 10)}
}

// Typed DATA only. The original cursor still validates the complete chain,
// selected input and observer role. These local guards establish the size of
// this exact value before encoding; a returned digest grants no authority.
func providerHistoryCollectionSpineDigestObserved(previous string, ordinal int, header FinalizedHeader, meter *providerWorkMeter) string {
	if meter == nil || ordinal < 0 || ordinal >= 1<<20 || ordinal == 0 && previous != "" ||
		ordinal > 0 && !lowerHash(previous) || !validHeader(header) {
		return ""
	}
	value := map[string]any{"previous": previous, "ordinal": strconv.Itoa(ordinal), "header": providerHistoryCollectionHeaderObject(header)}
	if !meter.event(providerCopyEvent, uint64(len("spine"))) {
		return ""
	}
	value["format"], value["kind"] = providerHistoryCollectionStateDomain, "spine"
	raw := providerReplayCanonical(value, providerHistoryCollectionSpineBytes, meter)
	if len(raw) == 0 {
		return ""
	}
	return providerReplayDigest(providerHistoryCollectionStateDomain, raw, meter)
}

func providerHistoryCollectionValueDigestObserved(kind string, value map[string]any, meter *providerWorkMeter) string {
	if meter == nil || !meter.event(providerCopyEvent, uint64(len(kind))) {
		return ""
	}
	value["format"], value["kind"] = providerHistoryCollectionStateDomain, kind
	raw := providerReplayCanonical(value, candidateLedgerMetadataLimit, meter)
	if len(raw) == 0 {
		return ""
	}
	return providerReplayDigest(providerHistoryCollectionStateDomain, raw, meter)
}

// Arrays are separate before any response-dependent mutation. Immutable string
// backing is deliberately shared and remains part of the resident proof.
func (c *providerHistoryCollectionCursor) copyObserved(meter *providerWorkMeter) *providerHistoryCollectionCursor {
	if c.commitmentObserved(meter) == "" || !meter.event(providerCopyEvent, uint64(1+len(c.spine)+len(c.effects))) {
		return nil
	}
	next := *c
	next.owner = &next
	next.spine = make([]FinalizedHeader, len(c.spine))
	copy(next.spine, c.spine)
	next.effects = make([]providerHistoryCollectionEffect, len(c.effects))
	copy(next.effects, c.effects)
	if !c.copyPendingObserved(&next, meter) {
		return nil
	}
	semantic, ok := c.semantic.snapshotObserved(meter)
	if !ok {
		return nil
	}
	next.semantic = semantic
	return &next
}
