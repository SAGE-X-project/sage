package registry011

// This deterministic DATA root reinterprets both confirmed and working state,
// lifetime indexes, pending bytes and every expected item. It is never an ACK
// or a substitute for replaying this reader's own original T/C/S.
func (f *providerHistorySemanticFold) commitmentObserved(meter *providerWorkMeter) string {
	// A read-only DATA inspection can have its own observer. It cannot fork
	// or execute the fold with that observer; snapshotObserved keeps binding.
	if !f.valid() || meter == nil {
		return ""
	}
	confirmed := providerHistorySemanticStreamRootObserved(f.confirmed, f.selection.config, meter)
	if confirmed == "" {
		return ""
	}
	window := ""
	if w := f.window; w != nil {
		if !w.selected(w.run) || w.blocks < 0 || w.blocks > w.budget.limits.MaxBlocks || w.compared < 0 || w.compared > len(w.queue) || uint64(len(w.queue)) > w.budget.limits.History.MaxItems ||
			w.historyBytes < 0 || w.historyBytes > w.budget.limits.History.MaxBytes || len(w.pendingCommand) > 262144 || w.pending != (len(w.pendingCommand) != 0) {
			return ""
		}
		working := providerHistorySemanticStreamRootObserved(replayStream{state: w.run.state, claims: w.run.claims, created: w.run.created}, f.selection.config, meter)
		base, a := providerHistorySemanticDecimalObserved(w.baseVersion, meter)
		pending, b := providerHistorySemanticDecimalObserved(w.pendingVersion, meter)
		queue := providerHistoryCollectionValueDigestObserved("semantic-queue", map[string]any{"count": len(w.queue)}, meter)
		for i, item := range w.queue {
			version, ok := providerHistorySemanticDecimalObserved(item.version, meter)
			entry := providerHistorySemanticBytesRootObserved(item.entry, meter)
			command := providerHistorySemanticBytesRootObserved(item.command, meter)
			if !ok || entry == "" || command == "" || queue == "" {
				return ""
			}
			queue = providerHistoryCollectionValueDigestObserved("semantic-expected-item", map[string]any{"previous": queue, "ordinal": i, "version": version, "entry": entry, "command": command}, meter)
		}
		pendingBytes := ""
		if w.pending {
			pendingBytes = providerHistorySemanticBytesRootObserved(w.pendingCommand, meter)
			if pendingBytes == "" {
				return ""
			}
		}
		counters := []int{w.budget.bytes, w.budget.txs, w.budget.frames, w.budget.logs, w.budget.traceLogs, w.budget.proofs}
		limits := []int{w.budget.limits.MaxEvidenceBytes, w.budget.limits.MaxTransactions, w.budget.limits.MaxFrames, w.budget.limits.MaxLogs, w.budget.limits.MaxLogs, w.budget.limits.MaxProofs}
		for i, n := range counters {
			if n < 0 || n > limits[i] {
				return ""
			}
		}
		if !a || !b || working == "" || queue == "" {
			return ""
		}
		window = providerHistoryCollectionValueDigestObserved("semantic-window", map[string]any{"state": working, "base_version": base, "queue": queue, "queue_count": len(w.queue), "pending_version": pending, "pending_command": pendingBytes, "pending": w.pending, "comparing": w.comparing, "compared": w.compared, "history_bytes": w.historyBytes, "blocks": w.blocks, "budget": counters}, meter)
		if window == "" {
			return ""
		}
	}
	return providerHistoryCollectionValueDigestObserved("semantic-fold", map[string]any{"program": f.selection.program, "selected": providerHistoryCollectionHeaderObject(f.selection.selected), "last": providerHistoryCollectionHeaderObject(f.last), "confirmed": confirmed, "window": window, "blocks": f.blocks, "items": f.items, "windows": f.windows}, meter)
}

func providerHistorySemanticStreamRootObserved(s replayStream, config JournalConfig, meter *providerWorkMeter) string {
	if s.state.version == 0 && (s.state.admitted != 0 || s.state.record.ID != "" || s.state.record.Controller != "" || s.state.record.State != "" || s.state.record.Version != "" || len(s.state.record.Keys) != 0 || len(s.state.record.Services) != 0 || len(s.state.grants) != 0) {
		return ""
	}
	record, grants, ok := s.stateBytesObserved(meter)
	version, a := providerHistorySemanticDecimalObserved(s.state.version, meter)
	admitted, b := providerHistorySemanticDecimalObserved(s.state.admitted, meter)
	claims := providerHistorySemanticIndexRootObserved(s, true, config.MaxClaims, meter)
	created := providerHistorySemanticIndexRootObserved(s, false, config.MaxIdentities, meter)
	if !ok || !a || !b || claims == "" || created == "" {
		return ""
	}
	recordRoot, grantsRoot := "", ""
	if s.state.version != 0 {
		recordRoot, grantsRoot = providerReplayDigest("semantic-record", record, meter), providerReplayDigest("semantic-grants", grants, meter)
		if recordRoot == "" || grantsRoot == "" {
			return ""
		}
	}
	return providerHistoryCollectionValueDigestObserved("semantic-state", map[string]any{"version": version, "admitted": admitted, "record": recordRoot, "grants": grantsRoot, "claims": claims, "created": created, "selected": providerHistoryCollectionHeaderObject(s.selected)}, meter)
}

func providerHistorySemanticBytesRootObserved(text string, meter *providerWorkMeter) string {
	if text == "" || len(text) > 262144 {
		return ""
	}
	raw := providerReplayBytes(text, meter)
	if len(raw) != len(text) {
		return ""
	}
	return providerReplayDigest("semantic-bytes", raw, meter)
}
