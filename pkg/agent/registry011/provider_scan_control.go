package registry011

import "bytes"

// The candidate control owner models the finite claim and sequential two-root
// protocol. It is deliberately NOT a selected irreversible root primitive or
// qualified ACK. No public/actual factory accepts this handle as authority.
type providerScanControl struct {
	owner    *providerScanControl
	store    *providerCustodyStore
	pin      providerScanHighwater
	prepared *providerScanAttempt
	held     providerScanUpper
	failed   bool
}

func (*providerScanControl) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Called under store.mu after the preprovisioned H0/HEAD0 have been verified.
// Missing files are never replaced, created or accepted as authoritative zero.
func newProviderScanControl(s *providerCustodyStore) (*providerScanControl, error) {
	if s == nil || !s.valid() || s.config.bounds.scan == nil || s.scanControl != nil || !lowerHash(s.config.expectedScanHead) || !s.config.bounds.scan.fits(providerScanOpenUpper(s.config), providerScanUpper{}) {
		return nil, errProviderIntegration
	}
	c := &providerScanControl{store: s}
	c.owner = c
	if !c.readHighwater(c.pin) {
		return nil, errProviderIntegration
	}
	codec, codecErr := newProviderScanCodec(s.config, s.activeMeter.Load())
	if codecErr != nil {
		return nil, errProviderIntegration
	}
	head, err := codec.head(c.pin)
	if err != nil || codec.hash(providerScanHeadDomain, head) != s.config.expectedScanHead {
		return nil, errProviderIntegration
	}
	s.scanControl = c
	return c, nil
}

func (c *providerScanControl) valid() bool {
	return c != nil && c.owner == c && !c.failed && c.store != nil && c.store.valid() && c.store.scanControl == c && c.store.config.bounds.scan != nil
}

func (c *providerScanControl) readExact(disk *retentionDisk, name string, want []byte) bool {
	s := c.store
	got, err := s.read(disk, name, int64(s.config.bounds.maximumMetadataPerCommit))
	return err == nil && s.event(providerCompareEvent, uint64(len(got)+len(want))) && bytes.Equal(got, want)
}

func (c *providerScanControl) readHighwater(h providerScanHighwater) bool {
	codec, err := newProviderScanCodec(c.store.config, c.store.activeMeter.Load())
	if err != nil {
		return false
	}
	raw, a := codec.highwater(h)
	head, b := codec.head(h)
	if a != nil || b != nil {
		return false
	}
	for _, disk := range []*retentionDisk{c.store.custody, c.store.evidence} {
		if !c.readExact(disk, providerCustodyName("scan-highwater", h.ordinal), raw) || !c.readExact(disk, "SCAN-HEAD", head) {
			return false
		}
	}
	return true
}

// Every previous original is checked from provisioned H0, not adopted from an
// arbitrary later HEAD. The live pin must equal the exact rederived endpoint.
func (c *providerScanControl) readPrefix() bool {
	s := c.store
	codec, err := newProviderScanCodec(s.config, s.activeMeter.Load())
	if err != nil {
		return false
	}
	var h providerScanHighwater
	raw, err := codec.highwater(h)
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		if err != nil || !c.readExact(disk, providerCustodyName("scan-highwater", 0), raw) {
			return false
		}
	}
	for j := uint64(1); j <= c.pin.ordinal; j++ {
		intent, err := s.read(s.custody, providerCustodyName("scan-intent", j), int64(s.config.bounds.maximumMetadataPerCommit))
		if err != nil {
			return false
		}
		a, err := codec.parseAttempt(intent)
		deadline, _, deadlineOK := codec.deadline(providerClockDiagnostic(s.setupOrigin))
		if err != nil || !deadlineOK || a.clock != providerClockDiagnostic(s.setupOrigin) || a.deadline != deadline || a.purpose != "initial-inspection" || a.ordinal != j || !c.readExact(s.evidence, providerCustodyName("scan-claim", j), intent) {
			return false
		}
		after, commit, err := codec.successor(h, a)
		next, nextErr := codec.highwater(after)
		if err != nil || nextErr != nil || !c.readExact(s.custody, providerCustodyName("scan-commit", j), commit) {
			return false
		}
		for _, disk := range []*retentionDisk{s.custody, s.evidence} {
			if !c.readExact(disk, providerCustodyName("scan-highwater", j), next) {
				return false
			}
		}
		h = after
	}
	return h == c.pin && c.readHighwater(h)
}

func providerScanInitialFiles(s *providerCustodyStore) (uint64, error) {
	if s == nil || s.config.bounds.scan == nil || !lowerHash(s.config.expectedScanHead) {
		return 0, errProviderIntegration
	}
	codec, err := newProviderScanCodec(s.config, s.activeMeter.Load())
	if err != nil {
		return 0, errProviderIntegration
	}
	zero := providerScanHighwater{}
	raw, a := codec.highwater(zero)
	head, b := codec.head(zero)
	if a != nil || b != nil || codec.hash(providerScanHeadDomain, head) != s.config.expectedScanHead {
		return 0, errProviderIntegration
	}
	c := &providerScanControl{store: s}
	if !c.readHighwater(zero) {
		return 0, errProviderIntegration
	}
	return uint64(2*len(raw) + 2*len(head)), nil
}

func (c *providerScanControl) checkMainPin() bool {
	s := c.store
	f := s.frontierFiles
	if f == nil || !f.valid() || !f.initialized || f.pending != nil || f.pin.commit != s.tip || f.pin.setup != s.setupTip || f.pin.sequence != s.sequence || f.pin.profile != s.config.profile {
		return false
	}
	return f.readExact("HEAD", []byte(f.pinRaw)) && f.readExact(providerCustodyName("highwater", f.pin.ordinal), []byte(f.pinRaw))
}

// Conservative finite control overhead from the closed maximum record size,
// including all prior-control reads, seven new payload copies and readbacks.
// Logical weights are not a physical allocator/CPU qualification.
func providerScanPreparationUpper(config providerCustodyConfig, ordinal uint64) (providerScanUpper, error) {
	if config.bounds.scan == nil || !config.bounds.valid() || ordinal == 0 || ordinal > config.bounds.scan.maximumAttempts {
		return providerScanUpper{}, errProviderIntegration
	}
	n := config.bounds.maximumMetadataPerCommit
	count, a := providerUpperProduct(ordinal, 128)
	count, b := providerUpperSum(count, 256)
	square, d := providerUpperProduct(n, n)
	units, e := providerUpperSum(square, n, 8)
	units, f := providerUpperProduct(units, count)
	decoded, g := providerUpperProduct(n, count)
	written, h := providerUpperProduct(n, 7)
	allocation, i := providerUpperProduct(config.bounds.fileOverheadBytes, 7)
	retained, j := providerUpperSum(written, allocation)
	if !a || !b || !d || !e || !f || !g || !h || !i || !j {
		return providerScanUpper{}, errProviderIntegration
	}
	return providerScanUpper{candidateWorkVector{0, 0, decoded, 0, 0, 0, 0, 0, 0, written, units}, retained, 4}, nil
}

// Under the store mutex, irreversibly hold P+B BEFORE the first old-control
// read/write. This is candidate in-process admission only; host rollback and
// process loss still need the separately selected actual retaining boundary.
// The result is metadata; it does not grant an operation or a dispatch permit.
func (c *providerScanControl) claim(a providerScanAttempt) error {
	if c == nil || c.store == nil {
		return errProviderIntegration
	}
	s := c.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !c.valid() || c.prepared != nil || s.activeMeter.Load() != nil {
		return errProviderIntegration
	}
	preparation, err := providerScanPreparationUpper(s.config, a.ordinal)
	body, bodyErr := providerScanBodyUpper(s.config)
	// Only bounded scalar/vector checks precede the irreversible claim. No
	// policy parse, profile encoding, successor hash or target read occurs here.
	if err != nil || bodyErr != nil || a.preparation != preparation || a.body != body || a.ordinal != c.pin.ordinal+1 || a.setup != s.setupTip || a.custody != s.tip || a.clock != providerClockDiagnostic(s.setupOrigin) || a.purpose != "initial-inspection" {
		return errProviderIntegration
	}
	held, ok := providerScanUpperSum(c.held, preparation, body)
	addition, addOK := providerScanUpperSum(preparation, body)
	total, totalOK := providerScanUpperSum(providerScanOpenUpper(s.config), held)
	if !ok || !addOK || !totalOK || c.held != c.pin.held || !s.config.bounds.scan.fits(providerScanOpenUpper(s.config), held) || !providerFirstAdmissionCapacity(s, total) || !candidateCapacity(0, s.stored, addition.retained, s.config.bounds.maximumStoredBytes) {
		return errProviderIntegration
	}
	c.held, s.stored, c.failed = held, s.stored+addition.retained, true
	m := &providerWorkMeter{scan: c, upper: preparation.work, maximumClocks: preparation.clocks}
	m.owner = m
	s.activeMeter.Store(m)
	defer s.activeMeter.Store(nil)
	succeeded := false
	defer func() {
		if !succeeded {
			s.failed = true
		}
	}()
	codec, err := newProviderScanCodec(s.config, m)
	if err != nil {
		return errProviderIntegration
	}
	deadline, duration, deadlineOK := codec.deadline(a.clock)
	if !deadlineOK || a.deadline != deadline || s.config.bounds.scan.totalDurationMS > duration {
		return errProviderIntegration
	}
	after, commit, err := codec.successor(c.pin, a)
	intent, intentErr := codec.attempt(a)
	if err != nil || intentErr != nil || after.held != held {
		return errProviderIntegration
	}
	start, err := s.sample()
	if err != nil || !candidateLedgerClockAdvance(a.clock, providerClockDiagnostic(start)) || candidateLedgerExpired(providerClockDiagnostic(start), a.deadline) || !c.readPrefix() || !c.checkMainPin() {
		return errProviderIntegration
	}
	raw, rawErr := codec.highwater(after)
	head, headErr := codec.head(after)
	if rawErr != nil || headErr != nil || s.write(s.custody, providerCustodyName("scan-intent", a.ordinal), intent) != nil || s.write(s.evidence, providerCustodyName("scan-claim", a.ordinal), intent) != nil || s.write(s.custody, providerCustodyName("scan-commit", a.ordinal), commit) != nil {
		return errProviderIntegration
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		if s.write(disk, providerCustodyName("scan-highwater", a.ordinal), raw) != nil {
			return errProviderIntegration
		}
	}
	if !c.readHighwater(c.pin) {
		return errProviderIntegration
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		pending := providerCustodyName("SCAN-HEAD.pending", a.ordinal)
		if s.write(disk, pending, head) != nil || !s.event(providerRenameEvent, 0) || disk.checkIdentity() != nil || disk.root.Rename(pending, "SCAN-HEAD") != nil || !s.event(providerSyncEvent, 0) || disk.sync() != nil || disk.checkIdentity() != nil {
			return errProviderIntegration
		}
	}
	if !c.readExact(s.custody, providerCustodyName("scan-intent", a.ordinal), intent) || !c.readExact(s.evidence, providerCustodyName("scan-claim", a.ordinal), intent) || !c.readExact(s.custody, providerCustodyName("scan-commit", a.ordinal), commit) || !c.readHighwater(after) || !c.checkMainPin() {
		return errProviderIntegration
	}
	end, err := s.sample()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(start), providerClockDiagnostic(end)) || candidateLedgerExpired(providerClockDiagnostic(end), a.deadline) || uint64(end.MonotonicMS-start.MonotonicMS) > s.config.bounds.setupMaximumDurationMS || m.failed {
		return errProviderIntegration
	}
	// Successful write diagnostics are distinct from the already-held upper.
	actualRaw, actualOK := providerUpperSum(s.retainedRaw, uint64(2*len(intent)+len(commit)+2*len(raw)+2*len(head)))
	if !actualOK || actualRaw > s.stored {
		return errProviderIntegration
	}
	s.retainedRaw = actualRaw
	c.pin, c.prepared, c.failed = after, &a, false
	succeeded = true
	return nil
}

func providerScanDeadline(config providerCustodyConfig, clock CandidateClockDiagnostic) (uint64, uint64, bool) {
	codec, err := newProviderScanCodec(config, nil)
	if err != nil {
		return 0, 0, false
	}
	return codec.deadline(clock)
}
