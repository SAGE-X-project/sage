package registry011

import "context"

type providerHistoryRecoveryGroupRecord struct {
	checkpoint *providerHistoryRecoveryCheckpoint
	frame      *providerHistoryCustodyFrame
	inventory  *providerHistoryInventory
	objects    [8]providerHistoryStoredObject
}

// Separate logical and physical DATA. A complete unacknowledged S remains
// unacknowledged; neither the old confirmed checkpoint nor any ACK is changed.
type providerHistoryRecoveryGroupModel struct {
	owner             *providerHistoryRecoveryGroupModel
	recovery          *providerHistoryRecoveryOwner
	initial           *providerHistoryRecoveryInitialModel
	pass              int
	ledger            *providerHistoryGroupLedger
	memberCheckpoint  *providerHistoryRecoveryCheckpoint
	memberTicket      *providerHistoryGroupTicket
	memberConsumption *providerHistoryGroupConsumption
	scope             providerHistoryCustodyScope
	pin               providerFrontierHighwater
	inventory         string
	clock             CandidateClockDiagnostic
	confirmedModel    string
	physical          *providerHistoryRecoveryCheckpoint
	records           []providerHistoryRecoveryGroupRecord
	retention         providerHistoryStorageCharge
	stageStart        providerHistoryStorageCharge
	tailState         providerHistoryRecoveryInputState
	tailKind          providerHistoryRecoveryFileKind
	tailIndex         int
	head              string
	complete          bool
}

func (*providerHistoryRecoveryGroupModel) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (g *providerHistoryRecoveryGroupModel) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return g != nil && g.owner == g && g.recovery == r && r != nil && g.pass >= 0 && g.pass < len(r.groupModels) &&
		r.groupModels[g.pass] == g && g.complete && g.initial != nil && g.initial == r.initialModels[g.pass] && g.initial.selectedLocked(r) &&
		g.ledger != nil && g.ledger.owner == g.ledger && lowerHash(g.confirmedModel) && g.physical != nil &&
		g.physical.pin == g.pin && g.scope == r.initial.scope && g.physical.recovery == r
}

func (g *providerHistoryRecoveryGroupModel) read(ctx context.Context, c *providerHistoryRecoveryCheckpoint, kind providerHistoryRecoveryFileKind, index int) ([]byte, bool, error) {
	r := g.recovery
	probe, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: c, kind: kind, index: index})
	if err != nil || probe == nil {
		return nil, false, errProviderIntegration
	}
	if probe.state != providerHistoryRecoveryWholeInput {
		if c != r.pending {
			return nil, false, errProviderIntegration
		}
		g.tailState, g.tailKind, g.tailIndex = probe.state, kind, index
		return nil, false, nil
	}
	return probe.raw, true, nil
}

func (r *providerHistoryRecoveryOwner) replayGroupModelCandidate(ctx context.Context, pass int) (*providerHistoryRecoveryGroupModel, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || pass < 0 || pass >= len(r.groupModels) || r.groupModels[pass] != nil ||
		r.initialModels[pass] == nil || !r.initialModels[pass].selectedLocked(r) ||
		pass == 1 && (r.groupModels[0] == nil || !r.groupModels[0].selectedLocked(r)) {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	initial := r.initialModels[pass]
	f := initial.frontier.current
	raw, err := f.bytesObserved(r.meter)
	if err != nil || !providerReplayTextEqual(providerReplayDigest("sage-provider-custody-frontier-v1", raw, r.meter), r.initial.pin.frontier, r.meter) {
		return nil, errProviderIntegration
	}
	var selected *FinalizedHeader
	if r.parent.bounds.collection != nil {
		if !initial.finalizedSelected {
			return nil, errProviderIntegration
		}
		selected = &initial.firstFinalized
	}
	digest := providerHistoryInitialRootObserved(r.parent.bounds, r.config.profile, r.initial.pin, initial.frontier.body.firstExchange, f.budget, f.objects, selected, r.meter)
	var cursor *providerHistoryCollectionCursor
	if selected != nil {
		cursor, err = newProviderHistoryCollectionCursorObserved(r.parent.bounds.program, r.parent.bounds.collection, *selected, r.meter)
		if err != nil {
			return nil, errProviderIntegration
		}
	}
	ledger, err := newProviderHistoryCollectionLedgerObserved(r.parent.bounds.program, r.config.profile, r.parent.bounds.costs, digest, r.parent.bounds.stages, r.parent.bounds.future, cursor, r.meter)
	if digest == "" || err != nil || !providerReplayTextEqual(ledger.head, r.initial.model, r.meter) {
		return nil, errProviderIntegration
	}
	g := &providerHistoryRecoveryGroupModel{recovery: r, initial: initial, pass: pass, ledger: ledger, scope: r.initial.scope,
		pin: r.initial.pin, inventory: f.objects, clock: f.clock, physical: r.initial}
	g.owner = g
	head, whole, err := g.read(ctx, r.initial, providerHistoryRecoveryHead, 0)
	if err != nil || !whole {
		return nil, errProviderIntegration
	}
	current := providerReplayPhaseBytes(r.confirmed.head, r.meter)
	if !providerReplayEqual(head, current, r.meter) {
		if r.pending == nil || !providerReplayEqual(head, providerReplayPhaseBytes(r.pending.head, r.meter), r.meter) {
			return nil, errProviderIntegration
		}
	}
	g.head = providerReplayString(head, r.meter)
	if g.head == "" {
		return nil, errProviderIntegration
	}
	// Reserve a minimum number of remaining actual inputs before allocating
	// the checkpoint chain or records. A pending checkpoint may stop at its
	// first missing file; confirmed checkpoints must be read in full.
	minimumFiles := uint64(1) // Final HEAD, in addition to the HEAD already read.
	if r.pending != nil {
		minimumFiles++ // At least the pending highwater probe.
	}
	if r.inputFiles > r.bounds.maximumFiles || minimumFiles > r.bounds.maximumFiles-r.inputFiles {
		return nil, errProviderIntegration
	}
	remainingFiles := r.bounds.maximumFiles - r.inputFiles
	var count uint64
	for c := r.confirmed; c != r.initial; c = c.previous {
		if c == nil || count >= r.parent.bounds.records || c.inventory.count < 1 || c.inventory.count > len(c.inventory.entries) ||
			!r.meter.event(providerCompareEvent, 1) {
			return nil, errProviderIntegration
		}
		files := uint64(6 + c.inventory.count) // Highwater, four controls, inventory, objects.
		if c.frame.phase == "settle" {
			if c.completed < 0 || c.completed > providerHistoryGroupMembers {
				return nil, errProviderIntegration
			}
			spools, ok := providerUpperProduct(uint64(c.completed), 2)
			if !ok {
				return nil, errProviderIntegration
			}
			files, ok = providerUpperSum(files, spools)
			if !ok {
				return nil, errProviderIntegration
			}
		}
		var ok bool
		minimumFiles, ok = providerUpperSum(minimumFiles, files)
		if !ok || minimumFiles > remainingFiles {
			return nil, errProviderIntegration
		}
		count++
	}
	slots, ok := providerUpperSum(count, 1)
	if !ok || slots > uint64(^uint(0)>>1) || !r.meter.event(providerCopyEvent, slots) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	chain := make([]*providerHistoryRecoveryCheckpoint, int(count))
	for c, i := r.confirmed, len(chain)-1; i >= 0; c, i = c.previous, i-1 {
		if !r.meter.event(providerCopyEvent, 1) {
			return nil, errProviderIntegration
		}
		chain[i] = c
	}
	g.records = make([]providerHistoryRecoveryGroupRecord, 0, int(slots))
	for _, c := range chain {
		whole, err := g.applyCheckpoint(ctx, c)
		if err != nil || !whole {
			return nil, errProviderIntegration
		}
	}
	if g.pin != r.confirmed.pin || !providerReplayTextEqual(g.ledger.head, r.confirmed.model, r.meter) {
		return nil, errProviderIntegration
	}
	g.confirmedModel = providerHistoryRecoveryCloneObserved(g.ledger.head, r.meter)
	if r.pending != nil {
		if _, err := g.applyCheckpoint(ctx, r.pending); err != nil {
			return nil, errProviderIntegration
		}
	}
	last, whole, err := g.read(ctx, r.initial, providerHistoryRecoveryHead, 0)
	if err != nil || !whole || !providerReplayEqual(last, providerReplayPhaseBytes(g.head, r.meter), r.meter) ||
		g.confirmedModel == "" || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	g.complete, r.groupModels[pass], accepted = true, g, true
	return g, nil
}

func (g *providerHistoryRecoveryGroupModel) applyCheckpoint(ctx context.Context, c *providerHistoryRecoveryCheckpoint) (bool, error) {
	r := g.recovery
	if c == nil || c.previous != g.physical || c.frame.before != g.pin || !c.admission.shapeLocked(r, c) ||
		!candidateLedgerClockAdvance(g.clock, c.frame.clock) || len(g.records) >= cap(g.records) || !r.inputReadyLocked(ctx) {
		return false, errProviderIntegration
	}
	highwater, whole, err := g.read(ctx, c, providerHistoryRecoveryHighwater, 0)
	if err != nil || !whole {
		return false, err
	}
	h, err := parseProviderFrontierHighwaterObserved(highwater, r.config, r.meter)
	if err != nil || h != c.pin || !providerReplayEqual(highwater, providerReplayPhaseBytes(c.head, r.meter), r.meter) {
		return false, errProviderIntegration
	}
	var controls [4][]byte
	for i, kind := range [4]providerHistoryRecoveryFileKind{providerHistoryRecoveryIntent, providerHistoryRecoveryCommit, providerHistoryRecoveryFrontier, providerHistoryRecoveryPending} {
		controls[i], whole, err = g.read(ctx, c, kind, 0)
		if err != nil || !whole {
			return false, err
		}
	}
	f, err := parseProviderHistoryCustodyFrameObserved(r.config, g.scope, g.pin, h, controls[0], controls[1], controls[2], r.meter)
	if err != nil || !providerReplayEqual(controls[0], controls[3], r.meter) || f.stage != c.frame.stage || f.phase != c.frame.phase || f.clock != c.frame.clock ||
		f.previousInventory != g.inventory || !providerReplayTextEqual(f.previousModel, g.ledger.head, r.meter) ||
		!providerReplayTextEqual(f.model, c.model, r.meter) || f.inventory != c.inventory.digest {
		return false, errProviderIntegration
	}
	record := providerHistoryRecoveryGroupRecord{checkpoint: c, frame: f}
	for i := range c.inventory.count {
		ref := c.inventory.entries[i]
		raw, whole, err := g.read(ctx, c, providerHistoryRecoveryObject, i)
		if err != nil || !whole {
			return false, err
		}
		object, err := newProviderHistoryStoredObjectObserved(ref.kind, providerReplayString(raw, r.meter), r.config.bounds.maximumObjectBytes, r.meter)
		if err != nil || uint64(len(raw)) != ref.bytes || !providerReplayTextEqual(object.digest, ref.digest, r.meter) {
			return false, errProviderIntegration
		}
		record.objects[i] = object
	}
	inventory, whole, err := g.read(ctx, c, providerHistoryRecoveryInventory, 0)
	if err != nil || !whole {
		return false, err
	}
	record.inventory, err = parseProviderHistoryInventoryObserved(inventory, g.inventory, f.inventory, record.objects[:c.inventory.count], min(r.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), r.meter)
	if err != nil {
		return false, errProviderIntegration
	}
	// Keep spool accounting local until the entire checkpoint is verified.
	// Partial pending files describe a tail, not a completed semantic prefix.
	prefixRetention := g.retention
	stageStart := g.stageStart
	if f.phase == "admit" {
		stageStart = g.retention
	}
	if f.phase == "settle" {
		whole, err = g.settlement(ctx, c, &record, &prefixRetention)
	} else {
		err = g.admitOrConsume(c, &record)
	}
	if err != nil || !whole {
		return false, err
	}
	if !providerReplayTextEqual(g.ledger.head, f.model, r.meter) {
		return false, errProviderIntegration
	}
	charge, ok := providerHistoryStorageChargeFor(record.inventory, record.objects[:c.inventory.count], f, uint64(len(highwater)), r.config.bounds.fileOverheadBytes)
	retained, a := prefixRetention.add(charge, r.parent.bounds.future)
	groupRetained, b := prefixRetention.add(charge, g.ledger.charged)
	if !ok || !a || !b || retained != groupRetained || !retained.withinStage(stageStart, c.admission.upper) || retained != c.retention || !r.meter.event(providerCopyEvent, 1) || !r.inputReadyLocked(ctx) {
		return false, errProviderIntegration
	}
	if f.phase == "consume" && (g.ledger.ticket == nil || g.ledger.consumption == nil || !r.meter.event(providerCopyEvent, 3) || !r.inputReadyLocked(ctx)) {
		return false, errProviderIntegration
	}
	// Keep this pass's actual T/C context when a later S clears ledger.ticket.
	// These pointers belong to the fresh replay, never the failed producer.
	if f.phase == "consume" {
		g.memberCheckpoint, g.memberTicket, g.memberConsumption = c, g.ledger.ticket, g.ledger.consumption
	}
	g.pin, g.physical, g.inventory, g.clock, g.retention = h, c, f.inventory, f.clock, retained
	g.stageStart = stageStart
	g.records = append(g.records, record)
	return true, nil
}

func (g *providerHistoryRecoveryGroupModel) admitOrConsume(c *providerHistoryRecoveryCheckpoint, record *providerHistoryRecoveryGroupRecord) error {
	r, d, f := g.recovery, c.admission, record.frame
	switch f.phase {
	case "admit":
		if c.inventory.count != 2 || record.objects[0].kind != "requests" || record.objects[1].kind != "ticket" ||
			!providerReplayTextEqual(record.objects[0].digest, d.requests.digest, r.meter) || !providerReplayTextEqual(record.objects[1].digest, d.ticket.digest, r.meter) {
			return errProviderIntegration
		}
		_, table, err := parseProviderHistoryRequestPackObserved(providerReplayBytes(record.objects[0].raw, r.meter), d.requests.digest, f.stage,
			r.parent.bounds.program, d.purpose, d.table, d.members, *r.config.bounds.methods, r.config.capture, r.config.bounds.maximumObjectBytes, r.meter)
		if err != nil {
			return errProviderIntegration
		}
		ticket, err := parseProviderHistoryGroupTicketObserved(providerReplayBytes(record.objects[1].raw, r.meter), r.parent.bounds.program, r.config.profile,
			r.parent.bounds.costs, g.ledger.head, g.inventory, table, d.upper, r.meter)
		if err != nil || !providerReplayTextEqual(ticket.digest, f.operation, r.meter) || g.ledger.admitOriginalsObserved(ticket, r.config.bounds.methods, r.meter) != nil {
			return errProviderIntegration
		}
	case "consume":
		if c.inventory.count != 1 || record.objects[0].kind != "consumption" || g.ledger.ticket == nil ||
			!providerHistoryGroupClockFits(d.clock, f.clock) {
			return errProviderIntegration
		}
		consumption, err := parseProviderHistoryGroupConsumptionObserved(providerReplayBytes(record.objects[0].raw, r.meter), g.ledger.ticket, g.ledger.head, g.inventory, r.meter)
		if err != nil || !providerReplayTextEqual(consumption.digest, f.operation, r.meter) || g.ledger.consumeObserved(consumption, r.meter) != nil {
			return errProviderIntegration
		}
	default:
		return errProviderIntegration
	}
	return nil
}
