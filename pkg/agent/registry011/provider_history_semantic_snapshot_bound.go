package registry011

import "reflect"

// Planned graph cardinalities for one snapshot, never an authenticated fold
// or a replacement observer. Pending bounds nonempty pendingCommand bytes,
// which snapshotObserved copies independently of the pending boolean flag.
type providerHistorySemanticSnapshotLimits struct {
	Confirmed, Working providerHistorySemanticCloneLimits
	Queue              uint64
	Window, Pending    bool
}

// Compose the actual confirmed/working clone and queue/pending copy events.
// The queue bound comes directly from each successful copy's262144-byte cap,
// not from an unchecked reported total or historyBytes. This is scoped logical
// work[10], not commitment/root multiplicity, native/RAM or factory authority.
func providerHistorySemanticSnapshotUpper(l providerHistorySemanticSnapshotLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if l.Queue > 4096 || (!l.Window && (l.Queue != 0 || l.Pending || l.Working != (providerHistorySemanticCloneLimits{}))) {
		return empty, false
	}
	confirmed, ok := providerHistorySemanticCloneUpper(l.Confirmed)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	units := a.sum(uint64(reflect.TypeFor[providerHistorySemanticFold]().Size()), 1, confirmed[10])
	if l.Window {
		working, ok := providerHistorySemanticCloneUpper(l.Working)
		if !ok {
			return empty, false
		}
		// One combined window/run copy event, and one queue array event
		// even for a nonnil-empty queue. A nil queue needs no extra event.
		window := a.sum(uint64(reflect.TypeFor[providerHistoryExpectedWindow]().Size()), uint64(reflect.TypeFor[replayRun]().Size()), 1)
		item := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
		queue := a.sum(1, a.product(l.Queue, a.sum(a.product(2, item), 1048581)))
		units = a.sum(units, window, working[10], queue)
		if l.Pending {
			units = a.sum(units, 524290)
		}
	}
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[10] = units
	return work, true
}

// Same owned snapshot route and original observer. The queue's immutable text
// sum is bounded by its SAME selected History.MaxBytes, as in the applied fold
// commitment. Pending text is separate. These are logical copy events only;
// shared string backing and simultaneous native allocation need another proof.
func providerHistoryAppliedSnapshotUpper(l providerHistorySemanticSnapshotLimits, historyBytes uint64, kinds providerHistoryAppliedStateKinds) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if !l.Window && (historyBytes != 0 || kinds.WorkingEmpty) || l.Window && (historyBytes == 0 || historyBytes > 8<<20) {
		return empty, false
	}
	// Preserve rejection of EVERY original child domain even when an empty
	// branch would not use its declared nonempty cardinalities.
	if _, ok := providerHistorySemanticSnapshotUpper(l); !ok {
		return empty, false
	}
	selected := l
	if kinds.ConfirmedEmpty {
		selected.Confirmed.Keys, selected.Confirmed.Services, selected.Confirmed.Grants = 0, 0, 0
	}
	if kinds.WorkingEmpty {
		selected.Working.Keys, selected.Working.Services, selected.Working.Grants = 0, 0, 0
	}
	work, ok := providerHistorySemanticSnapshotUpper(selected)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	if l.Window {
		queueBytes := min(historyBytes, a.product(l.Queue, 2*262144))
		// Each field has one text-to-bytes and one owned-string copy, so the
		// byte-dependent part is2*totalText. Array/item and +1 events stay in
		// the original row; no charge or counter is refunded on the observer.
		work[10] = a.sum(a.difference(work[10], a.product(l.Queue, 4*262144)), a.product(2, queueBytes))
	}
	if a.failed {
		return empty, false
	}
	return work, true
}
