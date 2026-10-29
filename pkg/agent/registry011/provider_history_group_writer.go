package registry011

import "os"

// A pending history write is owned by exactly one live controller and the
// original future meter. It is not a legacy write claim or a dispatch permit.
// Committed pins stay at before even after physical rename, until independent
// original replay and the original clock both succeed.
type providerHistoryWriteClaim struct {
	owner           *providerHistoryWriteClaim
	group           *providerHistoryGroupOwner
	meter           *providerWorkMeter
	frame           *providerHistoryCustodyFrame
	inventory       *providerHistoryInventory
	fresh           [8]providerHistoryStoredObject
	count           int
	highwater       string
	request, ticket providerHistoryStoredReference
	purpose         string
	members         int
	upper           providerHistoryGroupUpper
	coordinator     *providerHistoryGroupLedger
	charge          providerHistoryStorageCharge
	renamed, used   bool
	beforeCatalog   [2]map[string]bool
	afterCatalog    [2]map[string]bool
}

type providerHistoryGroupAcknowledgment struct {
	owner *providerHistoryGroupOwner
	claim *providerHistoryWriteClaim
	pin   providerFrontierHighwater
}

func (*providerHistoryWriteClaim) UnmarshalJSON([]byte) error { return errProviderIntegration }
func (*providerHistoryGroupAcknowledgment) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

func providerHistoryLedgerCopy(l *providerHistoryGroupLedger) *providerHistoryGroupLedger {
	if l == nil || l.owner != l {
		return nil
	}
	next := *l
	next.owner = &next
	if l.ticket != nil {
		ticket := *l.ticket
		next.ticket = &ticket
	}
	if l.consumption != nil {
		consumption := *l.consumption
		next.consumption = &consumption
	}
	return &next
}

// No recovered metadata can enter this transaction. This candidate entry
// retains T only; it does not issue C, authorize capture, or provide Source.
// The production whole-program derivation factory is still required before
// the profile can be opened by a real Agent.
func (o *providerHistoryGroupOwner) retainAdmissionCandidate(purpose string, requests []*providerIntegrationRequest, upper providerHistoryGroupUpper) (*providerHistoryGroupAcknowledgment, error) {
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.owner != o || o.parent == nil || o.parent.group != o || o.failed || o.phase != "prepared" ||
		o.coordinator == nil || o.custodian == nil || o.coordinator == o.custodian ||
		o.coordinator.phase != "prepared" || o.custodian.phase != "prepared" || o.coordinator.head != o.custodian.head ||
		s.activeMeter.Load() != nil || s.frontierFiles == nil || s.frontierFiles.pending != nil || s.frontierFiles.historyPending != nil ||
		len(requests) == 0 || len(requests) > providerHistoryGroupMembers || !upper.valid(len(requests)) {
		return nil, errProviderIntegration
	}
	// Validate the selected observer before taking its mutex or installing a
	// window. Invalid private state must refuse without a native operation.
	m := o.meter
	if m == nil || m.owner != m || m.historyGroup != o || m.integration != nil || m.scan != nil || m.scanBody != nil ||
		m.firstAdmission != nil || m.firstOperation != nil || m.guardian != nil || m.continuing != nil || m.parent != nil || !m.authority ||
		m.upper != o.parent.bounds.future.work || m.maximumClocks != o.parent.bounds.future.clocks {
		return nil, errProviderIntegration
	}
	work, workOK := upper.work()
	if !workOK {
		return nil, errProviderIntegration
	}
	m.mu.Lock()
	if m.failed || m.closed {
		m.mu.Unlock()
		return nil, errProviderIntegration
	}
	o.window, o.windowStart, o.windowUpper = true, m.actual, work
	o.clockStart, o.clockUpper = m.clocks, upper.clocks
	m.mu.Unlock()
	o.phase = "preparing-admit"
	s.activeMeter.Store(o.meter)
	succeeded := false
	defer func() {
		if !succeeded {
			o.failLocked()
		}
	}()
	if upper.clocks < 2 || !o.fits(o.meter) || !o.identityLocked() {
		return nil, errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) {
		return nil, errProviderIntegration
	}
	table, err := newProviderHistoryGroupTableObserved(o.parent.bounds.program, purpose, 1, requests, *s.config.bounds.methods, s.config.capture, o.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	inputs := make([]providerHistoryOriginalInput, table.count)
	for i := range table.count {
		inputs[i] = providerHistoryOriginalInput{"request", table.members[i].request.wire}
	}
	pack, err := newProviderHistoryOriginalPackObserved(1, table.digest, "requests", 0, table.count, inputs, min(s.config.bounds.maximumObjectBytes, providerHistoryOriginalPackBytes), o.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	ticket, err := newProviderHistoryGroupTicketObserved(o.parent.bounds.program, s.config.profile, o.parent.bounds.costs, o.coordinator.head, o.start.objects, table, upper, o.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	requestObject, err := newProviderHistoryStoredObjectObserved("requests", pack.raw, s.config.bounds.maximumObjectBytes, o.meter)
	ticketObject, ticketErr := newProviderHistoryStoredObjectObserved("ticket", ticket.raw, s.config.bounds.maximumObjectBytes, o.meter)
	if err != nil || ticketErr != nil {
		return nil, errProviderIntegration
	}
	fresh := []providerHistoryStoredObject{requestObject, ticketObject}
	inventory, err := newProviderHistoryInventoryObserved(o.start.objects, fresh, min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), o.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	coordinator := providerHistoryLedgerCopy(o.coordinator)
	if coordinator == nil || coordinator.admitObserved(ticket, o.meter) != nil {
		return nil, errProviderIntegration
	}
	frame, err := newProviderHistoryCustodyFrameObserved(s.config, o.scope, o.pin, 1, "admit", o.start.objects, inventory.digest, o.coordinator.head, coordinator.head, ticket.digest, providerClockDiagnostic(begin), o.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	raw, err := frame.after.bytesObserved(s.config, o.meter)
	charge, ok := providerHistoryStorageChargeFor(inventory, fresh, frame, uint64(len(raw)), s.config.bounds.fileOverheadBytes)
	retained, retainedOK := o.retention.add(charge, o.parent.bounds.future)
	groupCharge, groupOK := (providerHistoryStorageCharge{}).add(charge, coordinator.charged)
	if err != nil || !ok || !retainedOK || !groupOK || groupCharge != charge ||
		uint64(len(s.objects))+retained.objects > uint64(s.config.bounds.maximumObjects) || frame.after.sequence > uint64(s.config.bounds.maximumCommits) {
		return nil, errProviderIntegration
	}
	claim := &providerHistoryWriteClaim{group: o, meter: o.meter, frame: frame, inventory: inventory, count: len(fresh),
		purpose: purpose, members: table.count, upper: upper, coordinator: coordinator, charge: charge,
		request: inventory.entries[0], ticket: inventory.entries[1], highwater: providerReplayString(raw, o.meter)}
	claim.owner = claim
	copy(claim.fresh[:], fresh)
	if claim.highwater == "" || !o.admissionCatalogFitsLocked(claim) || !o.prepareAdmissionCatalogLocked(claim) || !o.exactAdmissionCatalogLocked(claim, false) {
		return nil, errProviderIntegration
	}
	// This is one no-refund suballocation of the original pre-H0 hold.
	// s.stored is deliberately unchanged; its equality binds that whole hold.
	o.retention = retained
	s.frontierFiles.historyPending, o.phase = claim, "writing-admit"
	if err := o.writeAdmissionLocked(claim); err != nil {
		return nil, errProviderIntegration
	}
	custodian, err := o.readAdmissionLocked(claim, true)
	if err != nil || custodian == coordinator || custodian.head != coordinator.head || !o.claimActiveLocked(claim) {
		return nil, errProviderIntegration
	}
	if !o.identityLocked() {
		return nil, errProviderIntegration
	}
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		uint64(end.MonotonicMS-begin.MonotonicMS) > 300000 || o.meter.failed || o.meter.closed {
		return nil, errProviderIntegration
	}
	// No earlier assignment publishes an after pin or a provisional model.
	s.frontierFiles.pin, s.frontierFiles.pinRaw = frame.after, claim.highwater
	o.pin, o.coordinator, o.custodian, o.inventory = frame.after, coordinator, custodian, inventory.digest
	s.sequence, s.tip = frame.after.sequence, frame.after.commit
	s.retainedRaw += charge.raw // The complete checked sum was held above.
	claim.used, s.frontierFiles.historyPending, o.phase = true, nil, "admitted"
	ack := &providerHistoryGroupAcknowledgment{o, claim, frame.after}
	o.last = ack
	s.activeMeter.Store(nil)
	succeeded = true
	return ack, nil
}

func (o *providerHistoryGroupOwner) claimActiveLocked(c *providerHistoryWriteClaim) bool {
	return o != nil && c != nil && c.owner == c && c.group == o && c.meter == o.meter && !c.used &&
		c.frame != nil && c.frame.before == o.pin && c.frame.stage == 1 && c.frame.phase == "admit" &&
		c.inventory != nil && c.count == 2 && c.inventory.count == c.count &&
		c.coordinator != nil && c.coordinator.owner == c.coordinator && c.coordinator.ticket != nil &&
		c.request == c.inventory.entries[0] && c.ticket == c.inventory.entries[1] && c.request.kind == "requests" && c.ticket.kind == "ticket" &&
		o.store.frontierFiles.historyPending == c && o.activeLocked(o.meter)
}

func (o *providerHistoryGroupOwner) admissionCatalogFitsLocked(c *providerHistoryWriteClaim) bool {
	b := o.parent.bounds
	if b.catalogEntries == 0 || b.catalogEntries > 1<<20 {
		return false
	}
	var count, names uint64
	for _, disk := range []*retentionDisk{o.store.custody, o.store.evidence} {
		// providerDirectoryNamesObserved performs one four-read identity check.
		var identity candidateWorkVector
		identity[10] = 4
		if !o.meter.preflightWork(identity) {
			return false
		}
		for range 4 {
			if !o.meter.event(providerReadEvent, 0) {
				return false
			}
		}
		items, err := providerDirectoryNamesObserved(disk, int(b.catalogEntries), o.meter)
		if err != nil {
			return false
		}
		for name := range items {
			if !candidateCapacity(0, count, 1, b.catalogEntries) || !candidateCapacity(0, names, uint64(len(name)), b.catalogNameBytes) {
				return false
			}
			count++
			names += uint64(len(name))
		}
	}
	// Retaining/replacing HEAD counts as a new cumulative file allocation too.
	// 255 is the admitted filename upper, not a reported actual filename size.
	additional, ok := providerUpperProduct(c.charge.files, 255)
	retained, rawOK := providerUpperSum(o.store.retainedRaw, c.charge.raw)
	return ok && rawOK && retained <= o.parent.held.retained && candidateCapacity(0, count, c.charge.files, b.catalogEntries) &&
		candidateCapacity(0, names, additional, b.catalogNameBytes)
}

func (o *providerHistoryGroupOwner) writeAdmissionLocked(c *providerHistoryWriteClaim) error {
	if !o.claimActiveLocked(c) || c.renamed {
		return errProviderIntegration
	}
	s, m, f := o.store, o.meter, o.store.frontierFiles
	before := providerReplayBytes(f.pinRaw, m)
	if !f.readExact("HEAD", before) || !f.readExact(s.name("highwater", o.pin.ordinal), before) {
		return errProviderIntegration
	}
	for i := range c.count {
		object := c.fresh[i]
		if o.writeHistoryFileLocked(s.evidence, providerHistoryStoredName(c.inventory.entries[i], m), providerReplayBytes(object.raw, m)) != nil {
			return errProviderIntegration
		}
	}
	for _, file := range []struct {
		disk *retentionDisk
		name string
		raw  string
	}{
		{s.evidence, providerHistoryInventoryName(c.inventory.digest, m), c.inventory.raw},
		{s.custody, s.name("intent", c.frame.after.sequence), c.frame.intent},
		{s.evidence, s.name("history-pending", c.frame.after.sequence), c.frame.intent},
		{s.custody, s.name("commit", c.frame.after.sequence), c.frame.commit},
		{s.custody, s.name("frontier", c.frame.after.sequence), c.frame.frontier},
		{s.custody, s.name("highwater", c.frame.after.ordinal), c.highwater},
		{s.custody, s.name("HEAD.pending", c.frame.after.ordinal), c.highwater},
	} {
		if file.name == "" || o.writeHistoryFileLocked(file.disk, file.name, providerReplayBytes(file.raw, m)) != nil {
			return errProviderIntegration
		}
	}
	if _, err := o.readAdmissionLocked(c, false); err != nil || !o.exactAdmissionCatalogLocked(c, true) || !f.readExact("HEAD", before) || !o.identityLocked() ||
		!m.event(providerRenameEvent, 0) || s.custody.root.Rename(s.name("HEAD.pending", c.frame.after.ordinal), "HEAD") != nil {
		return errProviderIntegration
	}
	c.renamed = true
	if !m.event(providerSyncEvent, 0) || s.custody.sync() != nil || !o.identityLocked() || !o.exactAdmissionCatalogLocked(c, true) {
		return errProviderIntegration
	}
	return nil
}

func (o *providerHistoryGroupOwner) readAdmissionLocked(c *providerHistoryWriteClaim, renamed bool) (*providerHistoryGroupLedger, error) {
	if !o.claimActiveLocked(c) || c.renamed != renamed {
		return nil, errProviderIntegration
	}
	s, m, f := o.store, o.meter, o.store.frontierFiles
	head := f.pinRaw
	if renamed {
		head = c.highwater
	}
	if !f.readExact("HEAD", providerReplayBytes(head, m)) || !f.readExact(s.name("highwater", o.pin.ordinal), providerReplayBytes(f.pinRaw, m)) ||
		!f.readExact(s.name("highwater", c.frame.after.ordinal), providerReplayBytes(c.highwater, m)) {
		return nil, errProviderIntegration
	}
	var originals [8]providerHistoryStoredObject
	for i := range c.count {
		ref := c.inventory.entries[i]
		raw, err := s.read(s.evidence, providerHistoryStoredName(ref, m), int64(ref.bytes))
		if err != nil || uint64(len(raw)) != ref.bytes {
			return nil, errProviderIntegration
		}
		originals[i], err = newProviderHistoryStoredObjectObserved(ref.kind, providerReplayString(raw, m), s.config.bounds.maximumObjectBytes, m)
		if err != nil || !providerReplayTextEqual(originals[i].digest, ref.digest, m) {
			return nil, errProviderIntegration
		}
	}
	rawInventory, err := s.read(s.evidence, providerHistoryInventoryName(c.inventory.digest, m), int64(s.config.bounds.maximumMetadataPerCommit))
	if err != nil {
		return nil, errProviderIntegration
	}
	if _, err := parseProviderHistoryInventoryObserved(rawInventory, o.start.objects, c.inventory.digest, originals[:c.count], min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m); err != nil {
		return nil, errProviderIntegration
	}
	_, table, err := parseProviderHistoryRequestPackObserved(providerReplayBytes(originals[0].raw, m), c.request.digest, 1, o.parent.bounds.program, c.purpose, c.coordinator.ticket.table.digest, c.members, *s.config.bounds.methods, s.config.capture, s.config.bounds.maximumObjectBytes, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	ticket, err := parseProviderHistoryGroupTicketObserved(providerReplayBytes(originals[1].raw, m), o.parent.bounds.program, s.config.profile, o.parent.bounds.costs, o.custodian.head, o.start.objects, table, c.upper, m)
	if err != nil || !providerReplayTextEqual(ticket.digest, c.ticket.digest, m) {
		return nil, errProviderIntegration
	}
	model := providerHistoryLedgerCopy(o.custodian)
	if model == nil || model.admitObserved(ticket, m) != nil || !providerReplayTextEqual(model.head, c.coordinator.head, m) {
		return nil, errProviderIntegration
	}
	intent, a := s.read(s.custody, s.name("intent", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	pending, b := s.read(s.evidence, s.name("history-pending", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	commit, d := s.read(s.custody, s.name("commit", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	frontier, e := s.read(s.custody, s.name("frontier", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	frame, err := parseProviderHistoryCustodyFrameObserved(s.config, o.scope, o.pin, c.frame.after, intent, commit, frontier, m)
	if a != nil || b != nil || d != nil || e != nil || err != nil || !providerReplayEqual(intent, pending, m) ||
		frame.previousInventory != o.start.objects || frame.inventory != c.inventory.digest || frame.previousModel != o.custodian.head ||
		frame.model != model.head || frame.operation != ticket.digest || frame.phase != "admit" || frame.stage != 1 {
		return nil, errProviderIntegration
	}
	return model, nil
}

// Keep the actual identity checks separate so each four-read invocation is
// metered at its real call site, including failed writes. disk.write() folds
// two checks inside one primitive and cannot provide that history transcript.
func (o *providerHistoryGroupOwner) historyDiskIdentityLocked(d *retentionDisk) bool {
	if !o.activeLocked(o.meter) || d == nil || d.root == nil || d.lock == nil || d != o.store.custody && d != o.store.evidence {
		return false
	}
	var identity candidateWorkVector
	identity[10] = 4
	if !o.meter.preflightWork(identity) {
		return false
	}
	for range 4 {
		if !o.meter.event(providerReadEvent, 0) {
			return false
		}
	}
	return d.checkIdentity() == nil
}

func (o *providerHistoryGroupOwner) writeHistoryFileLocked(d *retentionDisk, name string, raw []byte) error {
	if name == "" || len(raw) == 0 || !o.historyDiskIdentityLocked(d) || !o.meter.event(providerReadEvent, 0) {
		return errProviderIntegration
	}
	f, err := d.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errProviderIntegration
	}
	closed := false
	defer func() {
		if !closed {
			_ = f.Close() // Cleanup never restores a failed meter or emits an ACK.
		}
	}()
	if !o.meter.event(providerWriteEvent, uint64(len(raw))) {
		return errProviderIntegration
	}
	n, err := f.Write(raw)
	if err != nil || n != len(raw) || !o.meter.event(providerSyncEvent, 0) || d.syncFile(f) != nil || !o.meter.event(providerReadEvent, 0) {
		return errProviderIntegration
	}
	err = f.Close()
	closed = true
	if err != nil || !o.meter.event(providerSyncEvent, 0) || d.sync() != nil || !o.historyDiskIdentityLocked(d) {
		return errProviderIntegration
	}
	return nil
}
