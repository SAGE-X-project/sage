package registry011

import "reflect"

// This explicit private constructor binds one original observer for the whole
// fold. It is not a factory/admission path or an authority/profile witness.
// Its configuration encoder now keeps the same original observer. Logical
// copy/encode units do not qualify physical RAM, native CPU or ABI execution.
func newProviderHistorySemanticFoldObserved(program string, config JournalConfig, selected FinalizedHeader, maximumBlocks, maximumItems uint64, meter *providerWorkMeter) (*providerHistorySemanticFold, error) {
	if meter == nil || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistorySemanticFold]().Size())) {
		return nil, ErrConfiguration
	}
	f, err := newProviderHistorySemanticFoldWithObserver(program, config, selected, maximumBlocks, maximumItems, meter)
	if err != nil {
		return nil, err
	}
	f.observer, f.sealedObserver = meter, meter
	return f, nil
}

func (f *providerHistorySemanticFold) observedCopy(bytes uint64) bool {
	return f != nil && (f.observer == nil || f.observer.event(providerCopyEvent, bytes))
}

func (f *providerHistorySemanticFold) cloneConfirmedObserved() (replayStream, bool) {
	if f.observer == nil {
		return providerHistoryCloneSemanticState(f.confirmed), true
	}
	return replaySharedIndexCloneObserved(f.confirmed, f.selection.config.MaxClaims, f.selection.config.MaxIdentities, f.observer)
}

func (s replayStream) stateBytesObserved(meter *providerWorkMeter) ([]byte, []byte, bool) {
	if meter == nil {
		record, grants := s.stateBytes()
		return record, grants, true
	}
	if s.state.version == 0 {
		return nil, nil, meter.preflightWork(candidateWorkVector{})
	}
	record, a := replayStateRecordBytesObserved(s.state.record, meter)
	grants, b := replayStateGrantsBytesObserved(s.state.grants, meter)
	return record, grants, a && b
}
