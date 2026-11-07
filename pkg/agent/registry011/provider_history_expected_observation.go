package registry011

import "reflect"

func (w *providerHistoryExpectedWindow) prepareSubjectObserved(command replayCommand) bool {
	meter := w.budget.observer
	owned, ok := replayExpectedTextObserved(command.raw, meter)
	if !ok || !meter.event(providerCopyEvent, uint64(8+reflect.TypeFor[string]().Size()+reflect.TypeFor[bool]().Size())) {
		return false
	}
	w.pendingVersion, w.pendingCommand, w.pending = command.expected+1, owned, true
	return true
}

func (w *providerHistoryExpectedWindow) retainExpectedObserved(r *replayRun, step HistoryContentStep) bool {
	if !w.selected(r) || !w.pending || step.History.Version != w.pendingVersion || r.state.version != w.pendingVersion || len(w.queue) >= 4096 ||
		len(step.History.Entry) == 0 || len(step.History.Entry) > 262144 || len(step.History.Command) == 0 || len(step.History.Command) > 262144 {
		return false
	}
	meter := w.budget.observer
	if meter == nil || !providerReplayEqual(step.History.Command, providerReplayBytes(w.pendingCommand, meter), meter) ||
		!w.chargeHistory(len(step.Record)+len(step.Grants)+len(step.History.Entry)+len(step.History.Command)) {
		return false
	}
	entry, a := replayExpectedTextObserved(step.History.Entry, meter)
	command, b := replayExpectedTextObserved(step.History.Command, meter)
	itemSize := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
	if !a || !b || !meter.event(providerCopyEvent, uint64(len(w.queue)+1)*itemSize) {
		return false
	}
	out := make([]providerHistoryExpectedItem, len(w.queue)+1)
	for i, old := range w.queue {
		if !meter.event(providerCopyEvent, itemSize) {
			return false
		}
		out[i] = old
	}
	if !meter.event(providerCopyEvent, itemSize) || !meter.event(providerCopyEvent, uint64(8+reflect.TypeFor[string]().Size()+reflect.TypeFor[bool]().Size()+reflect.TypeFor[[]providerHistoryExpectedItem]().Size())) {
		return false
	}
	out[len(w.queue)] = providerHistoryExpectedItem{step.History.Version, entry, command}
	w.queue = out
	w.fold.items++
	w.pending, w.pendingCommand = false, ""
	return true
}

func (w *providerHistoryExpectedWindow) historyEqualObserved(version uint64, entry, command []byte) bool {
	if w == nil || w.budget.observer == nil || w.compared < 0 || w.compared >= len(w.queue) {
		return false
	}
	meter := w.budget.observer
	if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())) {
		return false
	}
	want := w.queue[w.compared]
	return version == want.version && providerReplayEqual(entry, providerReplayBytes(want.entry, meter), meter) &&
		providerReplayEqual(command, providerReplayBytes(want.command, meter), meter)
}
