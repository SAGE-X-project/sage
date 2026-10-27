package registry011

// Read the same frozen E0 again under the original finishing meter. The old
// inspection maps are discarded, not imported as evidence. This returns only
// typed DATA for the later first-I transaction; no E1, ACK or permit is made.
// The concrete transaction holds store.mu and has already started this owner.
func (f *providerFirstAdmissionOwner) replayEnrollmentLocked() (*providerScanReplayReader, error) {
	return f.replayEnrollmentPassLocked(false)
}

// The custodian owns a distinct irreversible pass. It never restores the
// inspection body, reuses its reader, or shares coordinator ledger/index maps.
func (f *providerFirstAdmissionOwner) replayCustodianLocked() (*providerScanReplayReader, error) {
	return f.replayEnrollmentPassLocked(true)
}

func (f *providerFirstAdmissionOwner) replayEnrollmentPassLocked(custodian bool) (*providerScanReplayReader, error) {
	if f == nil || f.store == nil || f.store.firstAdmissionOwner != f || f.state != providerFirstAdmissionStarted || !custodian && f.replayStarted || custodian && (!f.replayStarted || f.custodianStarted || f.plan == nil || f.plan.owner != f) {
		return nil, errProviderIntegration
	}
	if custodian {
		f.custodianStarted = true
	} else {
		f.replayStarted = true
	}
	s, m, b := f.store, f.meter, f.body
	success := false
	defer func() {
		if !success {
			f.abortLocked()
		}
	}()
	if !s.valid() || m == nil || m.firstAdmission != f || !f.fits(candidateWorkVector{}) || b == nil || b.active || !custodian && b.reader == nil || f.readerOutput == 0 || s.frontierFiles == nil || s.frontierFiles.pin != f.mainPin || s.frontierFiles.pinRaw != f.mainRaw || s.scanControl != f.control || f.control.prepared == nil || *f.control.prepared != f.attempt || f.control.pin != f.scanPin || s.setupOrigin != f.origin || f.mainPin.sequence != 1 || f.records != f.mainPin.ordinal+1 {
		return nil, errProviderIntegration
	}
	// One sample before construction, one per setup/main record and one after
	// the final original-head check. Both F and cumulative B must have room.
	samples, ok := providerUpperSum(f.mainPin.ordinal, 3)
	if !ok || m.clocks < f.startClocks || !candidateCapacity(0, m.clocks-f.startClocks, samples, f.limits.clocks) || !candidateCapacity(0, m.clocks, samples, m.maximumClocks) {
		return nil, errProviderIntegration
	}
	if _, err := f.sampleLocked(); err != nil || !f.control.readHighwater(f.scanPin) || !f.control.checkMainPin() {
		return nil, errProviderIntegration
	}
	// The captured owner already holds every needed scalar. Release only its
	// closed inspection caches. External diagnostic output is still charged
	// at the full captured readerOutput upper in the new reader's RAM model.
	b.reader, b.receipts, b.required, b.observed, b.targetRaw = nil, nil, nil, nil, ""
	raw := providerReplayBytes(f.targetRaw, m)
	r, err := newProviderScanReplayReaderWithOutput(f.control, m, raw, f.readerOutput)
	if err != nil || r.selected != f.mainPin || r.target.admission != "" || r.target.budget != "" || r.target.checkpoint != "" || r.target.enrollment == "" {
		return nil, errProviderIntegration
	}
	for r.nextSetup <= r.selected.ordinal-r.selected.sequence {
		if _, err := f.sampleLocked(); err != nil || r.setup() != nil {
			return nil, errProviderIntegration
		}
	}
	for r.view.sequence < r.selected.sequence {
		if _, err := f.sampleLocked(); err != nil || r.main() != nil {
			return nil, errProviderIntegration
		}
	}
	if !r.lineage() || r.replay.body.anchor == nil || r.replay.body.anchor.digest != r.target.enrollment || r.replay.body.anchor.budget != "" || r.replay.body.intent != nil || r.replay.body.ledger != nil || !f.control.readHighwater(f.scanPin) || !f.control.checkMainPin() {
		return nil, errProviderIntegration
	}
	if _, err := f.sampleLocked(); err != nil || m.failed || b.active || s.frontierFiles.pin != f.mainPin || s.frontierFiles.pinRaw != f.mainRaw {
		return nil, errProviderIntegration
	}
	success = true
	if custodian {
		f.custodian = r
	}
	return r, nil
}
