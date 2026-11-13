package registry011

import (
	"context"
	"errors"
	"reflect"
)

// Detached verification DATA only. No owner, model, catalog, store, clock,
// slice, map, interface or executable callback escapes with this value. All
// strings have independent bounded copies prepared inside the final reader
// BEFORE C1. Returning DATA never enables Source readiness or message use.
type providerHistoryFinalData struct {
	program, costs, profile, manifest, model, commit, collection string
	selected                                                     FinalizedHeader
	clockID, epoch                                               string
	beginWall, beginMonotonic, beginUncertainty                  int64
	endWall, endMonotonic, endUncertainty                        int64
	groups, sequence, files, bytes, entries, names               uint64
	retention                                                    providerHistoryStorageCharge
	work                                                         candidateWorkVector
}

func (*providerHistoryFinalData) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Additional final-read copy work, excluding its two existing clock events.
// The eventual pre-H0 compiler must bind clockIDBytes to the SAME immutable
// selected backend and add this row to its derived read work. No new clock-ID
// limit, profile/default or native allocation/CPU qualification is inferred.
func providerHistoryFinalDataCopyUpper(clockIDBytes uint64) (candidateWorkVector, bool) {
	if clockIDBytes == 0 {
		return candidateWorkVector{}, false
	}
	// Ten fixed hash strings and the nonempty selected clock ID; each Clone
	// observes n+1. One original struct-copy observation adds sizeof(DATA)+1.
	units, ok := providerUpperSum(clockIDBytes, 10*66, 11, uint64(reflect.TypeFor[providerHistoryFinalData]().Size()), 1)
	if !ok {
		return candidateWorkVector{}, false
	}
	return candidateWorkVector{10: units}, true
}

// In addition to the detached copy, the final owner retains an independent
// fixed value for a complete comparison at publication. Both copies belong
// to the same final read row before C1. This does not fund native backing,
// owner allocation/lifetime or the unfinished whole-profile compiler.
func providerHistoryFinalDataPreparationUpper(clockIDBytes uint64) (candidateWorkVector, bool) {
	copy, ok := providerHistoryFinalDataCopyUpper(clockIDBytes)
	if !ok {
		return candidateWorkVector{}, false
	}
	units, ok := providerUpperSum(copy[10], uint64(reflect.TypeFor[providerHistoryFinalData]().Size()), 1)
	if !ok {
		return candidateWorkVector{}, false
	}
	return candidateWorkVector{10: units}, true
}

// Binding comparison only, never admission, selection or native authority.
// Caller holds the original store mutex. A self-owned copy of a result is
// still a different result; every DATA field must match the sealed value.
func (f *providerHistoryFinalOwner) selectedDataMatchesLocked(out *providerHistoryFinalResult) bool {
	return f != nil && f.owner == f && out != nil && out.owner == out &&
		f.result == out && out.final == f && out.data == f.preparedData
}

// A pure DATA copy, never a verifier/selection constructor. Only the final
// owner constructs its inputs after both independent original models agree.
// Source primitives and allocation precede C1; failure publishes no partial
// value. Transient copies/backing/lifetime and cleanup still need whole-profile
// funding before the unfinished production factory may enable execution.
func providerHistoryFinalDataCopyObserved(in providerHistoryFinalData, meter *providerWorkMeter) (providerHistoryFinalData, bool) {
	var empty providerHistoryFinalData
	if in.clockID == "" || in.groups == 0 || in.groups > (1<<20)/3 || !validHeader(in.selected) {
		return empty, false
	}
	for _, value := range [10]string{in.program, in.costs, in.profile, in.manifest, in.model, in.commit, in.collection, in.selected.Hash, in.selected.ParentHash, in.epoch} {
		if !lowerHash(value) {
			return empty, false
		}
	}
	upper, ok := providerHistoryFinalDataCopyUpper(uint64(len(in.clockID)))
	if !ok || !meter.preflightWork(upper) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryFinalData]().Size())) {
		return empty, false
	}
	out := in
	for _, field := range [11]*string{&out.program, &out.costs, &out.profile, &out.manifest, &out.model, &out.commit, &out.collection, &out.selected.Hash, &out.selected.ParentHash, &out.epoch, &out.clockID} {
		copy := providerHistoryRecoveryCloneObserved(*field, meter)
		if copy == "" {
			return empty, false
		}
		*field = copy
	}
	return out, true
}

// The successful caller receives ONLY detached DATA, and only after BOTH
// original roots/locks are closed. Invalid/copied/unfinished results cannot
// close a foreign live execution. Close failure, cancellation or reentry
// withholds the entire value. There is no post-C1 replay, codec, native sample
// or DATA-to-authority conversion. Failed-execution cleanup remains the
// enclosing fresh store owner's responsibility, not a revived final lane.
func (out *providerHistoryFinalResult) closeAndTakeDataCandidate(ctx context.Context) (providerHistoryFinalData, error) {
	var empty providerHistoryFinalData
	if out == nil || out.owner != out || out.final == nil || out.final.parent == nil || out.final.parent.store == nil {
		return empty, errProviderIntegration
	}
	f, p := out.final, out.final.parent
	s := p.store
	s.mu.Lock()
	r, g := f.reader, p.group
	ready := p.owner == p && s.owner == s && s.history == p && s.clock != nil && s.clock == p.clock && s.custody != nil && s.evidence != nil && !s.closed && !s.failed && !p.failed &&
		p.final == f && p.recovery != nil && f.selectedDataMatchesLocked(out) && f.used && f.closed && !f.failed && r != nil && r.owner == r && r.parent == p && r.store == s && r.closed && !r.failed && !r.active &&
		r.meter != nil && r.meter.historyFinal == f && r.meter.closed && !r.meter.failed && !r.meter.authority && r.meter.clocks == 2 && r.meter.historyClockClaims == 2 &&
		g != nil && g.owner == g && g.parent == p && g.store == s && g.phase == "verified" && !g.failed && g.meter != nil && g.meter.historyGroup == g && g.meter.closed && !g.meter.failed &&
		out.checkpoint != nil && out.checkpoint == r.confirmed && out.checkpoint == p.recovery.confirmed &&
		out.catalog != nil && out.catalog == r.catalogResult && out.cursor != nil && out.cursor.owner == out.cursor && out.cursor.phase == "done" &&
		out.data.program == p.bounds.program && out.data.costs == p.bounds.costs && out.data.profile == p.config.profile &&
		out.data.manifest == p.collectionDigest && out.data.model == out.checkpoint.model && out.data.commit == out.checkpoint.pin.commit &&
		out.data.groups == out.cursor.completed && out.data.selected == out.cursor.selection.selected &&
		out.data.work == out.work && out.work == r.meter.actual && out.begin == f.begin && out.end != (ClockSample{}) &&
		out.data.clockID == out.begin.ClockID && out.data.epoch == out.begin.Epoch && out.data.clockID == out.end.ClockID && out.data.epoch == out.end.Epoch &&
		out.data.beginWall == out.begin.WallMS && out.data.beginMonotonic == out.begin.MonotonicMS && out.data.beginUncertainty == out.begin.UncertaintyMS &&
		out.data.endWall == out.end.WallMS && out.data.endMonotonic == out.end.MonotonicMS && out.data.endUncertainty == out.end.UncertaintyMS &&
		s.activeMeter.Load() == nil
	data := out.data // Scalar/value publication only; every string is cloned.
	s.mu.Unlock()
	if !ready {
		return empty, errProviderIntegration
	}
	closeErr := s.close() // Closing revokes old roles; detached DATA survives.
	var contextErr error
	if ctx == nil {
		contextErr = errProviderIntegration
	} else {
		contextErr = ctx.Err()
	}
	if err := errors.Join(closeErr, contextErr); err != nil {
		return empty, err
	}
	return data, nil
}
