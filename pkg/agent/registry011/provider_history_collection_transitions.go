package registry011

import (
	"context"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

func (c *providerHistoryCollectionCursor) contentBoundaryObserved(ctx context.Context, results []providerHistoryMemberResult, meter *providerWorkMeter) bool {
	if !c.valid() || ctx == nil || ctx.Err() != nil || meter == nil || meter != c.semantic.observer || c.mode != "content" ||
		c.semantic.window == nil || !c.semantic.window.selected(c.semantic.window.run) {
		return false
	}
	switch c.phase {
	case "content":
		return c.readContentObserved(ctx, results, meter)
	case "effects":
		if len(c.pending) == 0 || len(c.traceResults) != len(c.effects) || c.effectNext < 0 || c.effectNext >= len(c.effects) ||
			len(results) != min(providerHistoryGroupMembers, len(c.effects)-c.effectNext) ||
			!candidateCapacity(0, c.traces, uint64(len(results)), c.selection.input.effects) {
			return false
		}
		for i, result := range results {
			if ctx.Err() != nil || len(result.result) == 0 || len(result.result) > c.selection.input.capture.BodyBytes ||
				c.traceResults[c.effectNext+i] != "" || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[string]().Size())) {
				return false
			}
			// Results are immutable strings owned by this original replay. Parse
			// complete frames only when their block can be replayed below.
			c.traceResults[c.effectNext+i] = result.result
		}
		c.effectNext += len(results)
		c.traces += uint64(len(results))
		return c.applyReadyBlocksObserved(ctx, meter)
	case "state":
		if len(results) != 1 || len(c.pending) != 0 || c.semantic.last.Number != c.high || c.semantic.window.comparing || c.stopRecord != "" {
			return false
		}
		values, present, err := rpcABIObserved(providerReplayBytes(results[0].result, meter), true, meter, 65536)
		if err != nil || len(values) != 1 || present != (c.semantic.window.run.state.version != 0) {
			return false
		}
		c.stopVersion = c.semantic.window.run.state.version
		if !present {
			if len(values[0]) != 0 || c.semantic.compareStateCandidate(ctx, false, nil, nil) != nil {
				return false
			}
			return c.selectHistoryObserved(meter)
		}
		record, _, encoded := (replayStream{state: c.semantic.window.run.state}).stateBytesObserved(meter)
		if !encoded || !providerReplayEqual(values[0], record, meter) || !candidateCapacity(0, c.presentStops, 1, c.selection.input.presentStops) {
			return false
		}
		c.stopRecord = providerReplayString(values[0], meter)
		if c.stopRecord == "" || ctx.Err() != nil {
			return false
		}
		c.presentStops++
		c.phase = "grants"
		return true
	case "grants":
		if len(results) != 1 || c.stopRecord == "" || c.stopVersion == 0 || c.stopVersion != c.semantic.window.run.state.version {
			return false
		}
		values, _, err := rpcABIObserved(providerReplayBytes(results[0].result, meter), false, meter, 262144)
		if err != nil || len(values) != 1 || c.semantic.compareStateCandidate(ctx, true, providerReplayBytes(c.stopRecord, meter), values[0]) != nil {
			return false
		}
		c.stopRecord = ""
		return c.selectHistoryObserved(meter)
	case "history":
		if c.historyNext == 0 || c.historyNext > c.stopVersion || len(results) != int(min(uint64(providerHistoryGroupMembers), c.stopVersion-c.historyNext+1)) {
			return false
		}
		for i, result := range results {
			values, _, err := rpcABIObserved(providerReplayBytes(result.result, meter), false, meter, 262144, 262144)
			if ctx.Err() != nil || err != nil || len(values) != 2 || c.semantic.compareHistoryItemCandidate(ctx, c.historyNext+uint64(i), values[0], values[1]) != nil {
				return false
			}
		}
		if uint64(len(results)) == c.stopVersion-c.historyNext+1 {
			c.historyNext, c.phase = 0, "close"
		} else {
			c.historyNext += uint64(len(results))
		}
		return ctx.Err() == nil
	default:
		return false
	}
}

func (c *providerHistoryCollectionCursor) selectHistoryObserved(meter *providerWorkMeter) bool {
	w := c.semantic.window
	if meter == nil || w == nil || !w.comparing || w.baseVersion > c.stopVersion || c.stopVersion > c.selection.input.versions ||
		c.stopVersion-w.baseVersion != uint64(len(w.queue)) || !meter.event(providerCopyEvent, 16) {
		return false
	}
	if w.baseVersion == c.stopVersion {
		c.historyNext, c.phase = 0, "close"
	} else {
		c.historyNext, c.phase = w.baseVersion+1, "history"
	}
	return true
}

func (c *providerHistoryCollectionCursor) closeContentObserved(ctx context.Context, meter *providerWorkMeter) bool {
	if !c.valid() || ctx == nil || ctx.Err() != nil || meter == nil || meter != c.semantic.observer || c.mode != "content" ||
		len(c.pending) != 0 || len(c.effects) != 0 || len(c.traceResults) != 0 || c.pendingApplied != 0 || c.effectNext != 0 ||
		c.stopRecord != "" || c.historyNext != 0 || c.semantic.last.Number != c.high || c.semantic.finishWindowCandidate(ctx) != nil {
		return false
	}
	c.stopVersion = 0
	if c.high == c.selection.selected.Number {
		if !c.semantic.atEnd() {
			return false
		}
		c.phase = "done" // DATA completeness, never SourceReady/publication.
	} else {
		c.low = c.high + 1
		c.high = min(c.selection.selected.Number, c.low+c.selection.input.windowBlocks-1)
		c.next, c.phase = c.low, "open"
	}
	return ctx.Err() == nil
}

func (c *providerHistoryCollectionCursor) readContentObserved(ctx context.Context, results []providerHistoryMemberResult, meter *providerWorkMeter) bool {
	if c.next < c.low || c.next > c.high || len(c.pending) != 0 || len(c.effects) != 0 || len(c.traceResults) != 0 || c.pendingApplied != 0 || c.effectNext != 0 {
		return false
	}
	count := int(min(c.selection.input.contentBlocks, c.high-c.next+1))
	if count == 0 || !meter.event(providerCopyEvent, uint64(count)*uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())) {
		return false
	}
	c.pending = make([]providerHistoryCollectionBlock, count)
	position := 0
	for i := range count {
		want, ok := c.spineHeader(c.next+uint64(i), meter)
		if !ok || len(results)-position < 2 || ctx.Err() != nil {
			return false
		}
		raw, decoded, err := providerHistoryCollectionDecodeBlockObserved(ctx, results[position].result, results[position+1].result, want, c.selection.input, meter)
		position += 2
		if err != nil {
			return false
		}
		budget := replayBudget{limits: c.selection.input.journal.Replay, observer: meter}
		if want.Timestamp >= sepoliaAmsterdamTime {
			if position >= len(results) || !candidateCapacity(0, c.native, 1, c.selection.input.native) {
				return false
			}
			trace := providerReplayBytes(results[position].result, meter)
			if len(trace) != len(results[position].result) {
				return false
			}
			position++
			profile := ""
			switch c.selection.input.gas {
			case SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile, SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile:
			default:
				return false
			}
			if len(decoded.txs) == 0 && len(decoded.receipts) == 0 {
				// Empty rows carry no dimension name to infer. The schema was
				// already selected before acquisition, not chosen after failure.
				// Exact empty native rows and the unchanged strict checker must
				// still prove zero transactions/receipts/header gas. Preserve
				// these original bytes; no backend-identity claim follows.
				rows, empty := nodeRawArrayObserved(trace, budget.limits.MaxTransactions, meter)
				if !empty || len(rows) != 0 {
					return false
				}
				profile = c.selection.input.gas
			} else {
				switch c.selection.input.gas {
				case SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile:
					profile, trace, err = nodeRootGas(ctx, trace, decoded, &budget)
				default:
					profile, err = replayGasProfileObserved(trace, budget.limits.MaxTransactions, meter)
				}
			}
			if err != nil || !providerReplayTextEqual(profile, c.selection.input.gas, meter) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[ReplayGasEvidence]().Size())) {
				return false
			}
			raw.Gas = &ReplayGasEvidence{profile, want.Hash, trace}
			c.native++
		}
		if replayCheckGas(ctx, raw, decoded, &budget) != nil || !c.retainEffectsObserved(decoded, want, meter) {
			return false
		}
		c.pending[i], ok = providerHistoryCollectionRetainBlockObserved(raw, want, meter)
		if !ok {
			return false
		}
	}
	if position != len(results) || !meter.event(providerCopyEvent, uint64(len(c.effects))*uint64(reflect.TypeFor[string]().Size())) {
		return false
	}
	c.traceResults = make([]string, len(c.effects))
	return c.applyReadyBlocksObserved(ctx, meter)
}

// Only Registry logs in complete canonical receipts select effect traces.
// No producer list, destination heuristic or hash-only statement is used.
func (c *providerHistoryCollectionCursor) retainEffectsObserved(block replayDecodedBlock, header FinalizedHeader, meter *providerWorkMeter) bool {
	registry := common.HexToAddress(SepoliaRegistryAddress)
	count := 0
	for _, receipt := range block.receipts {
		if !meter.event(providerCompareEvent, uint64(len(receipt.Logs))*20) {
			return false
		}
		for _, log := range receipt.Logs {
			if log.Address == registry {
				count++
				break
			}
		}
	}
	if !candidateCapacity(0, c.traces, uint64(len(c.effects)+count), c.selection.input.effects) ||
		!meter.event(providerCopyEvent, uint64(len(c.effects)+count)*uint64(reflect.TypeFor[providerHistoryCollectionEffect]().Size())) {
		return false
	}
	next := make([]providerHistoryCollectionEffect, len(c.effects)+count)
	copy(next, c.effects)
	index := len(c.effects)
	for i, receipt := range block.receipts {
		if !meter.event(providerCompareEvent, uint64(len(receipt.Logs))*20) {
			return false
		}
		for _, log := range receipt.Logs {
			if log.Address != registry {
				continue
			}
			hash, hashOK := nodeTransactionHashObserved(block.txs[i], meter)
			text, textOK := nodeHashTextObserved(hash, meter)
			if !hashOK || !textOK {
				return false
			}
			next[index] = providerHistoryCollectionEffect{header.Hash, text, header.Number, uint64(i)}
			index++
			break
		}
	}
	c.effects = next
	return true
}

// Release derived transient bytes as soon as a whole block is replayed. The
// original immutable T/C/S archive remains retained, and neither reader skips
// that archive on replay. Logical capacity/work is never refunded here.
func (c *providerHistoryCollectionCursor) applyReadyBlocksObserved(ctx context.Context, meter *providerWorkMeter) bool {
	if !c.valid() || ctx == nil || ctx.Err() != nil || meter == nil || meter != c.semantic.observer ||
		c.semantic.window == nil || !c.semantic.window.selected(c.semantic.window.run) {
		return false
	}
	for c.pendingApplied < len(c.pending) {
		if ctx.Err() != nil {
			return false
		}
		block := c.pending[c.pendingApplied]
		first, end := -1, -1
		for i, effect := range c.effects {
			if !meter.event(providerCompareEvent, 8) {
				return false
			}
			if effect.number == block.header.Number {
				if first < 0 {
					first = i
				}
				end = i + 1
			}
		}
		if end > c.effectNext {
			c.phase = "effects"
			return true
		}
		raw, ok := block.replayObserved(meter)
		if !ok {
			return false
		}
		// Reconstruct the remaining ORIGINAL window capacity for each ready
		// block. Earlier blocks have already advanced the semantic budget;
		// earlier trace groups merely retained raw bytes and grant no renewal.
		// No partial normalization counter is persisted or copied into execution.
		window := c.semantic.window
		normalization := replayBudget{limits: window.budget.limits, frames: window.budget.frames,
			traceLogs: window.budget.traceLogs, observer: meter}
		if first >= 0 {
			if !meter.event(providerCopyEvent, uint64(end-first)*uint64(reflect.TypeFor[ReplayExecution]().Size())) {
				return false
			}
			raw.Executions = make([]ReplayExecution, end-first)
			for i := first; i < end; i++ {
				text := c.traceResults[i]
				if text == "" || len(text) > c.selection.input.capture.BodyBytes {
					return false
				}
				frame, err := decodeCallTracerBudgetObserved(ctx, providerReplayBytes(text, meter), &normalization)
				if err != nil {
					return false
				}
				effect := c.effects[i]
				raw.Executions[i-first] = ReplayExecution{effect.block, effect.transaction, effect.index, frame}
			}
		}
		if c.semantic.applyBlockCandidate(ctx, raw, block.header) != nil ||
			window.budget.frames != normalization.frames || window.budget.traceLogs != normalization.traceLogs ||
			!meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())) {
			return false
		}
		c.pending[c.pendingApplied] = providerHistoryCollectionBlock{header: block.header, transactionCount: block.transactionCount}
		for i := max(first, 0); i < end; i++ {
			if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[string]().Size())) {
				return false
			}
			c.traceResults[i] = ""
		}
		c.pendingApplied++
	}
	if c.effectNext != len(c.effects) || len(c.pending) == 0 {
		return false
	}
	last := c.pending[len(c.pending)-1].header.Number
	c.pending, c.effects, c.traceResults = nil, nil, nil
	c.pendingApplied, c.effectNext = 0, 0
	if last == c.high {
		c.next, c.phase = c.high, "state"
	} else {
		c.next, c.phase = last+1, "content"
	}
	return ctx.Err() == nil
}
