package registry011

import (
	"context"
	"strconv"
)

// A healthy replay contains DATA reconstructed with the original live meter.
// It never installs the terminal recovery role or an earlier live dispatch.
type providerHistoryHealthyPrefix struct {
	ledger     *providerHistoryGroupLedger
	pin        providerFrontierHighwater
	inventory  string
	clock      CandidateClockDiagnostic
	retention  providerHistoryStorageCharge
	stageStart providerHistoryStorageCharge
}

func (*providerHistoryHealthyPrefix) UnmarshalJSON([]byte) error { return errProviderIntegration }

// This scalar selector is safe under meter.mu. The original initial bridge
// stays immutable; only an acknowledged, successful S can become stageBefore.
func (o *providerHistoryGroupOwner) stageBaseLocked() *providerHistoryRecoveryCheckpoint {
	if o == nil || o.owner != o || o.parent == nil || o.parent.recovery == nil || o.start == nil {
		return nil
	}
	r, c := o.parent.recovery, o.stageBefore
	if c == nil || c.owner != c || c.recovery != r || c.group != o || c.scope != o.scope || o.stageRetentionStart != c.retention {
		return nil
	}
	if c == r.initial {
		if c.previous != nil || c.claim != nil || c.ack != nil || c.pin != o.start.main || c.retention != (providerHistoryStorageCharge{}) {
			return nil
		}
		return c
	}
	if !o.healthyCheckpointShapeLocked(c) || c.frame.phase != "settle" || c.failedS || c.memberArchive == nil ||
		c.frame.stage == 0 || c.frame.stage >= o.parent.bounds.stages || c.memberArchive.order.state != providerHistoryOrderClosed {
		return nil
	}
	return c
}

func (o *providerHistoryGroupOwner) nextStageLocked() uint64 {
	c := o.stageBaseLocked()
	if c == nil {
		return 0
	}
	stage := c.frame.stage + 1 // Original configured stages are bounded below 2^20/3.
	if stage > o.parent.bounds.stages {
		return 0
	}
	return stage
}

func (o *providerHistoryGroupOwner) healthyCheckpointShapeLocked(c *providerHistoryRecoveryCheckpoint) bool {
	if o == nil || o.parent == nil || o.parent.recovery == nil || c == nil || c.owner != c || c.claim == nil || c.claim.owner != c.claim ||
		c.recovery != o.parent.recovery || c.group != o || c.previous == nil || c.previous.owner != c.previous ||
		c.claim.group != o || c.claim.meter != o.meter || c.claim.checkpoint != c || !c.claim.used || !c.claim.renamed ||
		c.ack == nil || c.ack.owner != o || c.ack.claim != c.claim || c.ack.pin != c.pin ||
		c.frame.before != c.previous.pin || c.frame.after != c.pin || c.scope != o.scope || c.frame.scope != o.scope ||
		c.inventory.count < 1 || c.inventory.count > len(c.inventory.entries) ||
		!c.admission.shapeLocked(o.parent.recovery, c) || !o.parent.recovery.checkpointMemberArchiveShapeLocked(c) {
		return false
	}
	return true
}

// Only today's actual logical HEAD, or the exact active claim after its
// physical rename, can be selected. A caller's string supplies no authority.
func (o *providerHistoryGroupOwner) selectedHistoryHeadLocked(head string) bool {
	if o == nil || o.store == nil || o.store.frontierFiles == nil || !o.activeLocked(o.meter) {
		return false
	}
	f := o.store.frontierFiles
	expected := f.pinRaw
	if c := f.historyPending; c != nil && c.renamed {
		if c.owner != c || c.group != o || c.meter != o.meter || c.used || c.checkpoint != o.parent.recovery.pending {
			return false
		}
		expected = c.highwater
	}
	return providerReplayTextEqual(head, expected, o.meter)
}

// Reconstruct initial and every complete prior T/C/S in chronological order.
// The only cutoffs are the original stageBefore and, during advancing, the
// exact currently confirmed successful S. No observed name/model is adopted.
func (o *providerHistoryGroupOwner) replayHealthyPrefixLocked(ctx context.Context, cutoff *providerHistoryRecoveryCheckpoint, head string) (*providerHistoryHealthyPrefix, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	if !o.selectedHistoryHeadLocked(head) || cutoff == nil ||
		cutoff != o.stageBefore && (o.phase != "advancing" || cutoff != o.parent.recovery.confirmed) {
		return nil, errProviderIntegration
	}
	r, s, m := o.parent.recovery, o.store, o.meter
	var count uint64
	for c := cutoff; c != r.initial; c = c.previous {
		if ctx.Err() != nil || count >= o.parent.bounds.records || !o.healthyCheckpointShapeLocked(c) || c.failedS || !m.event(providerCompareEvent, 1) {
			return nil, errProviderIntegration
		}
		count++
	}
	if count > uint64(^uint(0)>>1) || !m.event(providerCopyEvent, count+1) {
		return nil, errProviderIntegration
	}
	chain := make([]*providerHistoryRecoveryCheckpoint, int(count))
	for c, i := cutoff, len(chain)-1; i >= 0; c, i = c.previous, i-1 {
		if ctx.Err() != nil || !m.event(providerCopyEvent, 1) {
			return nil, errProviderIntegration
		}
		chain[i] = c
	}
	initial, scope, err := o.initialModelAtLocked(o.start.main, r.initial.head, head)
	if err != nil || scope != o.scope || !providerReplayTextEqual(initial.head, r.initial.model, m) {
		return nil, errProviderIntegration
	}
	p := &providerHistoryHealthyPrefix{ledger: initial, pin: r.initial.pin, inventory: o.start.objects, clock: scope.origin}
	previous := r.initial
	for _, c := range chain {
		if ctx.Err() != nil || c.previous != previous || c.frame.before != p.pin || !candidateLedgerClockAdvance(p.clock, c.frame.clock) {
			return nil, errProviderIntegration
		}
		if err := o.applyHealthyCheckpointLocked(ctx, p, c); err != nil {
			return nil, errProviderIntegration
		}
		previous = c
	}
	if ctx.Err() != nil || previous != cutoff || p.pin != cutoff.pin || !providerReplayTextEqual(p.ledger.head, cutoff.model, m) ||
		!p.ledger.validObserved(m) || !s.frontierFiles.readExact("HEAD", providerReplayBytes(head, m)) {
		return nil, errProviderIntegration
	}
	return p, nil
}

func (o *providerHistoryGroupOwner) applyHealthyCheckpointLocked(ctx context.Context, p *providerHistoryHealthyPrefix, c *providerHistoryRecoveryCheckpoint) error {
	if ctx == nil || ctx.Err() != nil {
		return errProviderIntegration
	}
	s, m := o.store, o.meter
	highwater, err := s.read(s.custody, s.name("highwater", c.pin.ordinal), int64(s.config.bounds.maximumMetadataPerCommit))
	if err != nil || !providerReplayEqual(highwater, providerReplayBytes(c.head, m), m) {
		return errProviderIntegration
	}
	h, err := parseProviderFrontierHighwaterObserved(highwater, s.config, m)
	if err != nil || h != c.pin {
		return errProviderIntegration
	}
	var controls [4][]byte
	for i, item := range [4]struct {
		disk *retentionDisk
		kind string
	}{{s.custody, "intent"}, {s.custody, "commit"}, {s.custody, "frontier"}, {s.evidence, "history-pending"}} {
		controls[i], err = s.read(item.disk, s.name(item.kind, c.pin.sequence), providerHistoryCustodyFrameBytes)
		if err != nil || ctx.Err() != nil {
			return errProviderIntegration
		}
	}
	f, err := parseProviderHistoryCustodyFrameObserved(s.config, o.scope, p.pin, h, controls[0], controls[1], controls[2], m)
	if err != nil || !providerReplayEqual(controls[0], controls[3], m) ||
		!providerReplayEqual(controls[0], providerReplayBytes(c.frame.intent, m), m) ||
		!providerReplayEqual(controls[1], providerReplayBytes(c.frame.commit, m), m) ||
		!providerReplayEqual(controls[2], providerReplayBytes(c.frame.frontier, m), m) ||
		f.stage != c.frame.stage || f.phase != c.frame.phase || f.clock != c.frame.clock || f.previousInventory != p.inventory ||
		!providerReplayTextEqual(f.previousModel, p.ledger.head, m) || f.inventory != c.inventory.digest || f.model != c.model {
		return errProviderIntegration
	}
	var objects [8]providerHistoryStoredObject
	for i := range c.inventory.count {
		ref := c.inventory.entries[i]
		raw, readErr := s.read(s.evidence, providerHistoryStoredName(ref, m), int64(ref.bytes))
		if readErr != nil || ctx.Err() != nil || uint64(len(raw)) != ref.bytes {
			return errProviderIntegration
		}
		objects[i], err = newProviderHistoryStoredObjectObserved(ref.kind, providerReplayString(raw, m), s.config.bounds.maximumObjectBytes, m)
		if err != nil || !providerReplayTextEqual(objects[i].digest, ref.digest, m) {
			return errProviderIntegration
		}
	}
	rawInventory, err := s.read(s.evidence, providerHistoryInventoryName(c.inventory.digest, m), int64(s.config.bounds.maximumMetadataPerCommit))
	if err != nil {
		return errProviderIntegration
	}
	inventory, err := parseProviderHistoryInventoryObserved(rawInventory, p.inventory, f.inventory, objects[:c.inventory.count], min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m)
	if err != nil || !providerReplayEqual(rawInventory, providerReplayBytes(c.inventory.raw, m), m) {
		return errProviderIntegration
	}
	stageStart := p.stageStart
	if f.phase == "admit" {
		stageStart = p.retention
	}
	if err := o.applyHealthyRecordLocked(ctx, p, c, f, objects); err != nil || ctx.Err() != nil || !providerReplayTextEqual(p.ledger.head, f.model, m) {
		return errProviderIntegration
	}
	charge, ok := providerHistoryStorageChargeFor(inventory, objects[:c.inventory.count], f, uint64(len(highwater)), s.config.bounds.fileOverheadBytes)
	retained, a := p.retention.add(charge, o.parent.bounds.future)
	charged, b := p.retention.add(charge, p.ledger.charged)
	if ctx.Err() != nil || !ok || !a || !b || retained != charged || !retained.withinStage(stageStart, c.admission.upper) || retained != c.retention {
		return errProviderIntegration
	}
	p.pin, p.inventory, p.clock, p.retention = h, inventory.digest, f.clock, retained
	p.stageStart = stageStart
	return nil
}

func (o *providerHistoryGroupOwner) applyHealthyRecordLocked(ctx context.Context, p *providerHistoryHealthyPrefix, c *providerHistoryRecoveryCheckpoint, f *providerHistoryCustodyFrame, objects [8]providerHistoryStoredObject) error {
	if ctx == nil || ctx.Err() != nil {
		return errProviderIntegration
	}
	s, m, d := o.store, o.meter, c.admission
	switch f.phase {
	case "admit":
		if c.inventory.count != 2 || objects[0].kind != "requests" || objects[1].kind != "ticket" ||
			objects[0].digest != d.requests.digest || objects[1].digest != d.ticket.digest {
			return errProviderIntegration
		}
		_, table, err := parseProviderHistoryRequestPackObserved(providerReplayBytes(objects[0].raw, m), d.requests.digest, f.stage,
			o.parent.bounds.program, d.purpose, d.table, d.members, *s.config.bounds.methods, s.config.capture, s.config.bounds.maximumObjectBytes, m)
		if err != nil {
			return errProviderIntegration
		}
		ticket, err := parseProviderHistoryGroupTicketObserved(providerReplayBytes(objects[1].raw, m), o.parent.bounds.program, s.config.profile,
			o.parent.bounds.costs, p.ledger.head, p.inventory, table, d.upper, m)
		if err != nil || ticket.digest != f.operation || p.ledger.admitOriginalsObserved(ticket, s.config.bounds.methods, m) != nil {
			return errProviderIntegration
		}
	case "consume":
		if c.inventory.count != 1 || objects[0].kind != "consumption" || p.ledger.ticket == nil || !providerHistoryGroupClockFits(d.clock, f.clock) {
			return errProviderIntegration
		}
		consumption, err := parseProviderHistoryGroupConsumptionObserved(providerReplayBytes(objects[0].raw, m), p.ledger.ticket, p.ledger.head, p.inventory, m)
		if err != nil || consumption.digest != f.operation || p.ledger.consumeObserved(consumption, m) != nil {
			return errProviderIntegration
		}
	case "settle":
		return o.applyHealthySettlementLocked(ctx, p, c, f, objects)
	default:
		return errProviderIntegration
	}
	if ctx.Err() != nil {
		return errProviderIntegration
	}
	return nil
}

// Archived names are formatted and compared using the REAL healthy observer.
// No call below charges r.meter or consults an old producer ticket/cache.
func (o *providerHistoryGroupOwner) archivedMemberNameLocked(c *providerHistoryRecoveryCheckpoint, ordinal int, kind string) string {
	if o == nil || !o.activeLocked(o.meter) || !o.healthyCheckpointShapeLocked(c) || c.frame.phase != "settle" ||
		kind != "response" && kind != "tls" {
		return ""
	}
	sealed := o.parent.recovery.settledMemberLocked(c, ordinal)
	if sealed == nil {
		return ""
	}
	n := sealed.capture.intent
	expected := n.responseName
	if kind == "tls" {
		expected = n.tlsName
	}
	if n.consumption != c.previous || !lowerHash(c.previous.pin.commit) || n.stage != c.frame.stage {
		return ""
	}
	name := providerReplayJoin([]string{"history-original-", c.previous.pin.commit[2:], "-", strconv.FormatUint(n.stage, 10), "-", strconv.Itoa(ordinal), "-", kind}, o.meter)
	if name == "" || !providerReplayTextEqual(name, expected, o.meter) {
		return ""
	}
	return name
}
