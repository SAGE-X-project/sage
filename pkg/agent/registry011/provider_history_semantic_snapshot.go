package registry011

import "reflect"

// A candidate cursor owns the whole semantic working graph. It never borrows
// another reader's fold or replaces the original observer on a later copy.
// Explicit logical copies do not qualify native allocation/backing RAM.
func (f *providerHistorySemanticFold) snapshotObserved(meter *providerWorkMeter) (*providerHistorySemanticFold, bool) {
	if !f.valid() || meter == nil || f.observer != meter || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistorySemanticFold]().Size())) {
		return nil, false
	}
	next := *f
	next.owner = &next
	confirmed, ok := providerHistorySemanticStreamCopyObserved(f.confirmed, f.selection.config, meter)
	if !ok {
		return nil, false
	}
	next.confirmed = confirmed
	if f.window == nil {
		return &next, next.valid()
	}
	w := f.window
	if !w.selected(w.run) || len(w.queue) > 4096 || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryExpectedWindow]().Size()+reflect.TypeFor[replayRun]().Size())) {
		return nil, false
	}
	working, ok := providerHistorySemanticStreamCopyObserved(replayStream{state: w.run.state, claims: w.run.claims, created: w.run.created}, f.selection.config, meter)
	if !ok {
		return nil, false
	}
	window, run := *w, *w.run
	window.owner, window.fold, window.run = &window, &next, &run
	run.state, run.claims, run.created, run.budget = working.state, working.claims, working.created, &window.budget
	if w.queue != nil {
		itemSize := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
		if !meter.event(providerCopyEvent, uint64(len(w.queue))*itemSize) {
			return nil, false
		}
		window.queue = make([]providerHistoryExpectedItem, len(w.queue))
		for i, item := range w.queue {
			entry, a := replayExpectedTextObserved(providerReplayBytes(item.entry, meter), meter)
			command, b := replayExpectedTextObserved(providerReplayBytes(item.command, meter), meter)
			if !a || !b || !meter.event(providerCopyEvent, itemSize) {
				return nil, false
			}
			window.queue[i] = providerHistoryExpectedItem{item.version, entry, command}
		}
	}
	if w.pendingCommand != "" {
		window.pendingCommand, ok = replayExpectedTextObserved(providerReplayBytes(w.pendingCommand, meter), meter)
		if !ok {
			return nil, false
		}
	}
	next.window = &window
	return &next, next.valid() && window.selected(&run) && meter.preflightWork(candidateWorkVector{})
}

func providerHistorySemanticStreamCopyObserved(s replayStream, config JournalConfig, meter *providerWorkMeter) (replayStream, bool) {
	out, ok := replaySharedIndexCloneObserved(s, config.MaxClaims, config.MaxIdentities, meter)
	if !ok {
		return replayStream{}, false
	}
	// A fork preserves the exact zero-state shape; beginning a replay window
	// separately retains the legacy empty-array normalization contract.
	if s.state.record.Keys == nil {
		out.state.record.Keys = nil
	}
	if s.state.record.Services == nil {
		out.state.record.Services = nil
	}
	if s.state.grants == nil {
		out.state.grants = nil
	}
	if s.claims == nil {
		out.claims = nil
	}
	if s.created == nil {
		out.created = nil
	}
	return out, true
}
