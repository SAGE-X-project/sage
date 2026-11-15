package registry011

import "context"

// Only original S readers call this for a live transcript. Inert UNIT callers
// may exercise the same DATA grammar, but a successful return supplies neither
// original authentication nor permission. Every transition uses this reader's
// own original result bytes. The generic admission/factory gate stays closed
// until whole-source cost, native, residency and publication qualification.
func (c *providerHistoryCollectionCursor) advanceBoundaryObserved(ctx context.Context, table *providerHistoryGroupTable, results []providerHistoryMemberResult, methods *providerMethodBounds, meter *providerWorkMeter) (*providerHistoryCollectionCursor, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrConfiguration
	}
	if !c.valid() || table == nil || methods == nil || meter == nil || len(results) != table.count {
		return nil, ErrConfiguration
	}
	expected, err := c.nextTableObserved(table.stage, *methods, c.selection.input.capture, meter)
	if err != nil || !providerHistoryCollectionTableEqualObserved(expected, table, meter) {
		return nil, ErrConfiguration
	}
	for i, result := range results {
		if ctx.Err() != nil || !lowerHash(result.digest) || len(result.result) == 0 || len(result.result) > table.members[i].capture.BodyBytes {
			return nil, ErrConfiguration
		}
	}
	next := c.copyObserved(meter)
	if next == nil || ctx.Err() != nil {
		return nil, ErrConfiguration
	}
	switch c.phase {
	case "open":
		if !next.windowIdentityObserved(results, false, meter) {
			return nil, ErrConfiguration
		}
		if c.mode == "content" && next.semantic.beginWindowCandidate(ctx) != nil {
			return nil, ErrConfiguration
		}
		next.phase = next.mode
	case "ancestry":
		if !next.ancestryObserved(results, meter) {
			return nil, ErrConfiguration
		}
	case "close":
		if !next.windowIdentityObserved(results, true, meter) {
			return nil, ErrConfiguration
		}
		if c.mode == "content" {
			if !next.closeContentObserved(ctx, meter) {
				return nil, ErrConfiguration
			}
			break
		}
		deployment := c.selection.input.journal.Deployment.Number
		if c.low > deployment {
			next.high = c.low - 1
			next.low = max(deployment, next.high-min(next.high, c.selection.input.windowBlocks-1))
			next.next, next.phase = next.high, "open"
		} else {
			if uint64(len(next.spine)) != c.selection.selected.Number-deployment+1 {
				return nil, ErrConfiguration
			}
			next.mode, next.phase, next.low, next.next = "content", "open", deployment, deployment
			next.high = min(c.selection.selected.Number, deployment+c.selection.input.windowBlocks-1)
		}
	case "content", "effects", "state", "grants", "history":
		if !next.contentBoundaryObserved(ctx, results, meter) {
			return nil, ErrConfiguration
		}
	default:
		return nil, ErrConfiguration
	}
	next.completed++
	if next.commitmentObserved(meter) == "" || ctx.Err() != nil {
		return nil, ErrConfiguration
	}
	return next, nil
}

// Compare every request field and unused member slot, not just supplied hashes.
func providerHistoryCollectionTableEqualObserved(a, b *providerHistoryGroupTable, meter *providerWorkMeter) bool {
	if a == nil || b == nil || meter == nil || a.program != b.program || a.purpose != b.purpose || a.stage != b.stage || a.count != b.count ||
		!providerReplayTextEqual(a.raw, b.raw, meter) || !providerReplayTextEqual(a.digest, b.digest, meter) {
		return false
	}
	for i := range a.members {
		if a.members[i].class != b.members[i].class || a.members[i].capture != b.members[i].capture ||
			!providerHistoryGroupRequestEqualObserved(&a.members[i].request, &b.members[i].request, meter) {
			return false
		}
	}
	return true
}

func (c *providerHistoryCollectionCursor) headerResultObserved(result providerHistoryMemberResult, meter *providerWorkMeter) (FinalizedHeader, error) {
	return providerHistoryRPCHeaderObserved(providerReplayBytes(result.result, meter), RPCResponseLimits{c.selection.input.capture.BodyBytes, c.selection.input.capture.JSONDepth}, meter)
}

func (c *providerHistoryCollectionCursor) windowIdentityObserved(results []providerHistoryMemberResult, closing bool, meter *providerWorkMeter) bool {
	chain, genesis, registry, client, finalized := 0, 1, 2, 3, 4
	if closing {
		if len(results) != 7 {
			return false
		}
		chain, genesis, registry, client, finalized = 3, 4, 5, 6, 2
		stopNumber := c.high
		if c.mode == "ancestry" {
			stopNumber = c.low
		}
		stop, ok := c.spineHeader(stopNumber, meter)
		got, err := c.headerResultObserved(results[0], meter)
		selected, selectedErr := c.headerResultObserved(results[1], meter)
		if !ok || err != nil || selectedErr != nil || got != stop || selected != c.selection.selected {
			return false
		}
	} else if len(results) != 5+len(c.selection.input.codes) {
		return false
	}
	var chainID, version string
	if providerReplayUnmarshal(providerReplayBytes(results[chain].result, meter), &chainID, meter) != nil ||
		!providerReplayTextEqual(chainID, "0xaa36a7", meter) ||
		providerReplayUnmarshal(providerReplayBytes(results[client].result, meter), &version, meter) != nil || len(version) == 0 || len(version) > 256 {
		return false
	}
	g, err := c.headerResultObserved(results[genesis], meter)
	f, finalizedErr := c.headerResultObserved(results[finalized], meter)
	if err != nil || finalizedErr != nil || g.Number != 0 || !providerReplayTextEqual(g.Hash, sepoliaGenesis, meter) ||
		f.Number < c.selection.selected.Number || f.Number == c.selection.selected.Number && f != c.selection.selected {
		return false
	}
	values, _, err := rpcABIObserved(providerReplayBytes(results[registry].result, meter), false, meter, 128)
	if err != nil || len(values) != 1 || !providerReplayTextEqual(providerReplayString(values[0], meter), c.selection.input.journal.Registry, meter) {
		return false
	}
	if c.client == "" {
		if closing || c.completed != 0 {
			return false
		}
		c.genesis, c.client = g, version
	} else if c.genesis != g || !providerReplayTextEqual(c.client, version, meter) {
		return false
	}
	if !closing {
		for i, expected := range c.selection.input.codes {
			code, codeErr := rpcDataObserved(providerReplayBytes(results[5+i].result, meter), meter)
			digest := providerReplayDigest("", code, meter)
			if codeErr != nil || len(code) == 0 || len(expected.SHA256) != 64 || len(digest) != 66 || !providerReplayTextEqual(digest[2:], expected.SHA256, meter) {
				return false
			}
		}
	}
	return true
}

func (c *providerHistoryCollectionCursor) ancestryObserved(results []providerHistoryMemberResult, meter *providerWorkMeter) bool {
	if c.mode != "ancestry" || c.next < c.low || c.next > c.high || len(results) != int(min(uint64(providerHistoryGroupMembers), c.next-c.low+1)) ||
		!candidateCapacity(0, uint64(len(c.spine)), uint64(len(results)), c.selection.input.blocks) || !meter.event(providerCopyEvent, uint64(len(c.spine)+len(results))) {
		return false
	}
	// Allocate exact capacity and charge every copied old/new slot. An append
	// growth could silently copy the old prefix again or retain excess capacity.
	spine := make([]FinalizedHeader, len(c.spine)+len(results))
	copy(spine, c.spine)
	for i, result := range results {
		header, err := c.headerResultObserved(result, meter)
		child := spine[len(c.spine)+i-1]
		if err != nil || header.Number != c.next-uint64(i) || header.Number+1 != child.Number ||
			!providerReplayTextEqual(child.ParentHash, header.Hash, meter) || child.Timestamp <= header.Timestamp ||
			header.Number == c.selection.input.journal.Deployment.Number && header != c.selection.input.journal.Deployment {
			return false
		}
		spine[len(c.spine)+i] = header
	}
	c.spine = spine
	if uint64(len(results)) == c.next-c.low+1 {
		c.next, c.phase = c.low, "close"
	} else {
		c.next -= uint64(len(results))
	}
	return true
}
