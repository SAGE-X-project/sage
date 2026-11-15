package registry011

import (
	"reflect"
	"strconv"
)

// Every retained byte is freshly hashed under the original observer. The root
// is DATA integrity only: no supplied digest replaces original reconstruction.
func providerHistoryCollectionBytesRootObserved(text string, maximum int, meter *providerWorkMeter) string {
	if maximum < 1 || maximum > 64<<20 || len(text) == 0 || len(text) > maximum || meter == nil {
		return ""
	}
	raw := providerReplayBytes(text, meter)
	if len(raw) != len(text) {
		return ""
	}
	return providerReplayDigest("collection-pending-bytes", raw, meter)
}

func (c *providerHistoryCollectionCursor) pendingRootObserved(meter *providerWorkMeter) string {
	if !c.valid() || meter == nil || c.pendingApplied < 0 || c.effectNext < 0 || c.effectNext > len(c.effects) || len(c.traceResults) != len(c.effects) {
		return ""
	}
	if len(c.pending) == 0 {
		if c.pendingApplied != 0 || len(c.effects) != 0 || c.effectNext != 0 {
			return ""
		}
		return providerHistoryCollectionValueDigestObserved("pending-empty", map[string]any{}, meter)
	}
	if c.phase != "effects" || c.mode != "content" || uint64(len(c.pending)) > c.selection.input.contentBlocks || c.pendingApplied >= len(c.pending) ||
		len(c.effects) == 0 || c.effectNext >= len(c.effects) || c.semantic.window == nil {
		return ""
	}
	root := ""
	for i, block := range c.pending {
		header, ok := c.spineHeader(c.next+uint64(i), meter)
		if !ok || header != block.header || block.header.Number > c.high || block.transactionCount < 0 || block.transactionCount > c.selection.input.journal.Replay.MaxTransactions {
			return ""
		}
		applied := i < c.pendingApplied
		headerRoot, txRoot, receiptRoot, gasRoot := "", "", "", ""
		if applied {
			if block.headerRLP != "" || len(block.transactions) != 0 || len(block.receipts) != 0 || block.gasProfile != "" || block.gasTrace != "" {
				return ""
			}
		} else {
			if len(block.transactions) != block.transactionCount || len(block.receipts) != block.transactionCount {
				return ""
			}
			headerRoot = providerHistoryCollectionBytesRootObserved(block.headerRLP, 65536, meter)
			txRoot = providerHistoryCollectionListRootObserved(block.transactions, c.selection.input.journal.Replay.MaxEvidenceBytes, meter)
			receiptRoot = providerHistoryCollectionListRootObserved(block.receipts, c.selection.input.journal.Replay.MaxEvidenceBytes, meter)
			if headerRoot == "" || txRoot == "" || receiptRoot == "" {
				return ""
			}
			if block.header.Timestamp >= sepoliaAmsterdamTime {
				if !providerReplayTextEqual(block.gasProfile, c.selection.input.gas, meter) {
					return ""
				}
				gasRoot = providerHistoryCollectionBytesRootObserved(block.gasTrace, c.selection.input.journal.Replay.MaxEvidenceBytes, meter)
				if gasRoot == "" {
					return ""
				}
			} else if block.gasProfile != "" || block.gasTrace != "" {
				return ""
			}
		}
		root = providerHistoryCollectionValueDigestObserved("pending-block", map[string]any{"previous": root, "ordinal": i,
			"header": providerHistoryCollectionHeaderObject(block.header), "applied": applied, "transaction_count": block.transactionCount,
			"header_rlp": headerRoot, "transactions": txRoot, "receipts": receiptRoot, "gas_profile": block.gasProfile, "gas_trace": gasRoot}, meter)
		if root == "" {
			return ""
		}
	}
	if c.pendingApplied > 0 {
		if c.semantic.last != c.pending[c.pendingApplied-1].header {
			return ""
		}
	} else if c.semantic.blocks == 0 {
		if c.next != c.selection.input.journal.Deployment.Number {
			return ""
		}
	} else if c.semantic.last.Number == ^uint64(0) || c.semantic.last.Number+1 != c.next {
		return ""
	}
	for i, effect := range c.effects {
		if effect.number < c.next || effect.number-c.next >= uint64(len(c.pending)) {
			return ""
		}
		block := c.pending[effect.number-c.next]
		if effect.index >= uint64(block.transactionCount) {
			return ""
		}
		applied := effect.number-c.next < uint64(c.pendingApplied)
		text, digest := c.traceResults[i], ""
		if applied || i >= c.effectNext {
			if text != "" || applied && i >= c.effectNext {
				return ""
			}
		} else {
			digest = providerHistoryCollectionBytesRootObserved(text, c.selection.input.capture.BodyBytes, meter)
			if digest == "" {
				return ""
			}
		}
		root = providerHistoryCollectionValueDigestObserved("pending-trace", map[string]any{"previous": root, "ordinal": i,
			"applied": applied, "received": i < c.effectNext, "bytes": digest}, meter)
		if root == "" {
			return ""
		}
	}
	return providerHistoryCollectionValueDigestObserved("pending-batch", map[string]any{"members": root, "blocks": len(c.pending), "applied": c.pendingApplied}, meter)
}

func providerHistoryCollectionListRootObserved(values []string, maximum int, meter *providerWorkMeter) string {
	root := providerHistoryCollectionValueDigestObserved("pending-list", map[string]any{"count": len(values)}, meter)
	for i, value := range values {
		digest := providerHistoryCollectionBytesRootObserved(value, maximum, meter)
		if digest == "" || root == "" {
			return ""
		}
		root = providerHistoryCollectionValueDigestObserved("pending-list-item", map[string]any{"previous": root, "ordinal": strconv.Itoa(i), "bytes": digest}, meter)
	}
	return root
}

func (c *providerHistoryCollectionCursor) copyPendingObserved(next *providerHistoryCollectionCursor, meter *providerWorkMeter) bool {
	if next == nil || meter == nil || !meter.event(providerCopyEvent, uint64(len(c.pending))*uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())) ||
		!meter.event(providerCopyEvent, uint64(len(c.traceResults))*uint64(reflect.TypeFor[string]().Size())) {
		return false
	}
	next.pending = make([]providerHistoryCollectionBlock, len(c.pending))
	for i, block := range c.pending {
		if !nodeJSONSliceObserved(len(block.transactions), 2, meter) || !nodeJSONSliceObserved(len(block.receipts), 2, meter) {
			return false
		}
		next.pending[i] = block
		next.pending[i].transactions = make([]string, len(block.transactions))
		next.pending[i].receipts = make([]string, len(block.receipts))
		copy(next.pending[i].transactions, block.transactions)
		copy(next.pending[i].receipts, block.receipts)
	}
	next.traceResults = make([]string, len(c.traceResults))
	copy(next.traceResults, c.traceResults)
	return true
}
