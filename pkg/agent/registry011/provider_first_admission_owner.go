package registry011

const (
	providerFirstAdmissionPrepared = iota + 1
	providerFirstAdmissionStarted
	providerFirstAdmissionFailed
)

// A terminal inspection can hand its exact live B meter to one private
// finishing phase. Returned chunk DATA cannot construct or restore this owner.
// This handoff alone supplies neither the mandatory I/E1 binding nor an ACK.
type providerFirstAdmissionOwner struct {
	store              *providerCustodyStore
	body               *providerScanBody
	meter              *providerWorkMeter
	control            *providerScanControl
	attempt            providerScanAttempt
	scanPin            providerScanHighwater
	mainPin            providerFrontierHighwater
	mainRaw, targetRaw string
	lastChunk          string
	records            uint64
	origin, inspected  ClockSample
	limits             providerFirstAdmissionBounds
	startWork, upper   candidateWorkVector
	startClocks        uint64
	maximumClocks      uint64
	state              int
}

func (*providerFirstAdmissionOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Called only under store.mu after the last full catalog and owned sample.
// B already copied bounded target DATA before final IO. This checks scalars.
// The caller installs the returned owner immediately before successful return;
// no fallible IO follows that handoff. Inspection defers still close the body.
func newProviderFirstAdmissionOwner(b *providerScanBody, result *providerScanDataResult, inspected ClockSample) (*providerFirstAdmissionOwner, error) {
	if !b.valid() || b.beforeFirst == nil || b.receipts == nil || !b.receipts.valid() || b.encoding || b.finalCatalog || !b.lineageComplete || b.reader == nil || !b.reader.valid() || b.control.prepared == nil || result == nil || result.target != b.control.prepared.target || len(result.chunks) == 0 || uint64(len(result.chunks)) != result.records || len(b.receipts.chunks) != len(result.chunks) || len(b.observed) != len(b.required) || !candidateLedgerClockAdvance(result.end, providerClockDiagnostic(inspected)) || candidateLedgerExpired(providerClockDiagnostic(inspected), b.control.prepared.deadline) {
		return nil, errProviderIntegration
	}
	s, m := b.control.store, b.meter
	if s.firstAdmissionOwner != nil || s.frontierFiles == nil || s.bootstrapOwner == nil || !s.bootstrapOwner.done || s.bootstrapOwner.failed || m.closed || m.failed || s.config.bounds.firstAdmission == nil || b.targetRaw == "" || uint64(len(b.targetRaw)) > s.config.bounds.maximumMetadataPerCommit || result.records != b.reader.selected.ordinal+1 {
		return nil, errProviderIntegration
	}
	f := &providerFirstAdmissionOwner{store: s, body: b, meter: m, control: b.control, attempt: *b.control.prepared, scanPin: b.control.pin, mainPin: s.frontierFiles.pin, mainRaw: s.frontierFiles.pinRaw, targetRaw: b.targetRaw, lastChunk: b.receipts.chunks[len(b.receipts.chunks)-1], records: result.records, origin: s.setupOrigin, inspected: inspected, limits: *s.config.bounds.firstAdmission, startWork: m.actual, upper: m.upper, startClocks: m.clocks, maximumClocks: m.maximumClocks, state: providerFirstAdmissionPrepared}
	return f, nil
}

// The future concrete first-E1 transaction holds store.mu for its entire
// finishing phase. It must install an abort defer before any primitive after
// this start. No callback, reopened DATA or alternate meter enters this path.
func (f *providerFirstAdmissionOwner) startLocked() error {
	if f == nil || f.store == nil || f.store.firstAdmissionOwner != f || f.state != providerFirstAdmissionPrepared {
		return errProviderIntegration
	}
	s, m, b := f.store, f.meter, f.body
	if !s.valid() || m == nil || m.owner != m || m.closed || m.failed || m.firstAdmission != nil || m.actual != f.startWork || m.clocks != f.startClocks || m.upper != f.upper || m.maximumClocks != f.maximumClocks || m.scan != f.control || m.scanBody != b || s.activeMeter.Load() != nil || b == nil || b.active || b.failed || b.meter != m || s.scanBody != b || s.scanControl != f.control || !f.control.valid() || f.control.prepared == nil || *f.control.prepared != f.attempt || f.control.pin != f.scanPin || s.frontierFiles == nil || !s.frontierFiles.valid() || s.frontierFiles.pin != f.mainPin || s.frontierFiles.pinRaw != f.mainRaw || s.setupOrigin != f.origin || s.config.bounds.firstAdmission == nil || *s.config.bounds.firstAdmission != f.limits {
		f.abortLocked()
		return errProviderIntegration
	}
	f.state = providerFirstAdmissionStarted // Before any sample/read/codec/write.
	m.firstAdmission = f
	s.activeMeter.Store(m)
	return nil
}

// Called under the meter mutex while the concrete transaction holds store.mu.
// F is only a sublimit of the original B, whose global checks still follow.
func (f *providerFirstAdmissionOwner) fits(work candidateWorkVector) bool {
	if f == nil || f.state != providerFirstAdmissionStarted || f.store == nil || f.store.firstAdmissionOwner != f || f.meter == nil || f.meter.firstAdmission != f || f.store.activeMeter.Load() != f.meter || f.store.closed || f.store.failed || f.body == nil || f.body.active || f.body.failed {
		return false
	}
	m := f.meter
	if m.upper != f.upper || m.maximumClocks != f.maximumClocks || m.scan != f.control || m.scanBody != f.body || f.body.meter != m || f.store.scanBody != f.body || f.store.scanControl != f.control || f.store.setupOrigin != f.origin || f.store.config.bounds.firstAdmission == nil || *f.store.config.bounds.firstAdmission != f.limits || m.clocks < f.startClocks || m.clocks-f.startClocks > f.limits.clocks {
		return false
	}
	for k, n := range work {
		if m.actual[k] < f.startWork[k] || !candidateCapacity(0, m.actual[k]-f.startWork[k], n, f.limits.work[k]) {
			return false
		}
	}
	return true
}

// The concrete finishing transaction must call this before its first IO and
// at each write/readback boundary. Waiting after inspection consumes both the
// original deadline and F duration; no new clock epoch or deadline is created.
func (f *providerFirstAdmissionOwner) sampleLocked() (ClockSample, error) {
	if f == nil || f.store == nil || f.store.firstAdmissionOwner != f || f.state != providerFirstAdmissionStarted {
		return ClockSample{}, errProviderIntegration
	}
	if !f.meter.preflight(providerClockEvent, 0) {
		f.abortLocked()
		return ClockSample{}, errProviderIntegration
	}
	sample, err := f.store.sample()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(f.inspected), providerClockDiagnostic(sample)) || candidateLedgerExpired(providerClockDiagnostic(sample), f.attempt.deadline) || uint64(sample.MonotonicMS-f.inspected.MonotonicMS) > f.limits.durationMS || uint64(sample.MonotonicMS-f.origin.MonotonicMS) > f.store.config.bounds.scan.totalDurationMS {
		f.abortLocked()
		return ClockSample{}, errProviderIntegration
	}
	return sample, nil
}

// This is only failure closure. Success remains unavailable until a concrete
// first-I/binding/seeded-E1 transaction and exact original readbacks exist.
func (f *providerFirstAdmissionOwner) abortLocked() {
	if f == nil || f.store == nil || f.store.firstAdmissionOwner != f {
		return
	}
	f.state, f.store.failed = providerFirstAdmissionFailed, true
	if f.control != nil {
		f.control.failed = true
	}
	if m := f.meter; m != nil {
		m.mu.Lock()
		m.failed, m.closed = true, true
		m.mu.Unlock()
		if f.store.activeMeter.Load() == m {
			f.store.activeMeter.Store(nil)
		}
	}
}
