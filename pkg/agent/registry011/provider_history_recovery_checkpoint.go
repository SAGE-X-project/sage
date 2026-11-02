package registry011

import "strings"

// A checkpoint is sealed by the original healthy writer before its first
// physical write. Only the original final commit can select it as confirmed.
// A pending checkpoint describes expected bytes, never a recovered ACK or
// authority. Its value fields contain no catalog map or mutable model alias.
type providerHistoryRecoveryCheckpoint struct {
	owner                           *providerHistoryRecoveryCheckpoint
	recovery                        *providerHistoryRecoveryOwner
	group                           *providerHistoryGroupOwner
	previous                        *providerHistoryRecoveryCheckpoint
	claim                           *providerHistoryWriteClaim // Provenance only; replay must use the values below.
	ack                             *providerHistoryGroupAcknowledgment
	scope                           providerHistoryCustodyScope
	pin                             providerFrontierHighwater
	head                            string
	model                           string
	retention                       providerHistoryStorageCharge
	frame                           providerHistoryCustodyFrame
	inventory                       providerHistoryInventory
	failedS                         bool
	admission                       *providerHistoryRecoveryAdmissionDescriptor
	completed                       int
	memberArchive                   *providerHistoryRecoveryMemberArchive
	failureStage                    string
	failureWritten, failureReceived int
	descriptor                      *providerHistoryRecoveryInitialDescriptor // Initial checkpoint only.
}

func (*providerHistoryRecoveryCheckpoint) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

// Original writer DATA copies are charged to that writer, not to the future
// recovery allowance. Clone avoids retaining an alias to a larger raw value.
func providerHistoryRecoveryCloneObserved(raw string, meter *providerWorkMeter) string {
	if raw == "" || meter == nil || !meter.event(providerCopyEvent, uint64(len(raw))) {
		return ""
	}
	return strings.Clone(raw)
}

func (r *providerHistoryRecoveryOwner) prepareInitialCheckpointLocked(g *providerHistoryGroupOwner, scope providerHistoryCustodyScope, model string) *providerHistoryRecoveryCheckpoint {
	if r == nil || !r.unspentLocked() || r.originalGroup != g || r.initial != nil || r.confirmed != nil || r.pending != nil || r.pendingAck != nil ||
		g == nil || g.parent != r.parent || g.store != r.store || g.phase != "initializing" || !g.activeLocked(g.meter) ||
		scope.initial != g.start.main || scope.exchange != g.start.exchange || scope.origin != providerClockDiagnostic(r.parent.original) || !lowerHash(model) {
		return nil
	}
	c := &providerHistoryRecoveryCheckpoint{recovery: r, group: g, scope: scope, pin: g.start.main}
	c.owner = c
	c.head = providerHistoryRecoveryCloneObserved(r.store.frontierFiles.pinRaw, g.meter)
	c.model = providerHistoryRecoveryCloneObserved(model, g.meter)
	if c.head == "" || c.model == "" {
		return nil
	}
	c.descriptor = r.prepareInitialDescriptorLocked(c)
	if c.descriptor == nil {
		return nil
	}
	return c
}

func (r *providerHistoryRecoveryOwner) prepareWriteCheckpointLocked(c *providerHistoryWriteClaim) bool {
	if r == nil || !r.unspentLocked() || c == nil || c.owner != c || c.group != r.originalGroup || c.meter != r.originalMeter ||
		c.group == nil || !c.group.activeLocked(c.meter) || c.used || c.renamed || c.checkpoint != nil ||
		r.initial == nil || r.confirmed == nil || r.pending != nil || r.pendingAck != nil || r.confirmed.owner != r.confirmed || r.confirmed.recovery != r ||
		r.confirmed.group != c.group || r.confirmed.pin != c.group.pin || r.confirmed.ack != c.group.last ||
		c.frame == nil || c.frame.before != r.confirmed.pin || c.frame.scope != r.initial.scope ||
		c.inventory == nil || c.count <= 0 || c.count > len(c.inventory.entries) || c.inventory.count != c.count ||
		c.frame.inventory != c.inventory.digest || c.coordinator == nil || c.frame.model != c.coordinator.head {
		return false
	}
	switch c.frame.phase {
	case "admit":
		if c.group.stageBaseLocked() != r.confirmed || !c.group.admissionBaseShapeLocked() ||
			c.frame.before != r.confirmed.pin || c.frame.stage != c.group.nextStageLocked() {
			return false
		}
	case "consume":
		if c.admission == nil || r.confirmed.frame.phase != "admit" || r.confirmed.claim != c.admission.claim {
			return false
		}
	case "settle":
		if r.confirmed.frame.phase != "consume" || c.consumed != r.confirmed.ack {
			return false
		}
	default:
		return false
	}
	// All allocations and observed copies precede physical write and native
	// end. The final ACK selection below is a scalar publication only.
	n := &providerHistoryRecoveryCheckpoint{recovery: r, group: c.group, previous: r.confirmed, claim: c,
		scope: c.frame.scope, pin: c.frame.after, frame: *c.frame, inventory: *c.inventory, model: c.frame.model, failedS: c.failure.kind != ""}
	n.owner = n
	retained, ok := c.group.retention.add(c.charge, r.parent.bounds.future)
	if !ok || !r.prepareAdmissionDescriptorLocked(c, n) || !retained.withinStage(c.group.stageRetentionStart, n.admission.upper) {
		return false
	}
	n.retention = retained
	n.completed = r.completed
	if n.completed < 0 || n.completed > n.admission.members {
		return false
	}
	if n.failedS {
		if c.dispatch == nil || c.dispatch.attempt == nil || c.dispatch.attempt.failure == nil {
			return false
		}
		failure := c.dispatch.attempt.failure
		switch failure.failureStage {
		case "dial", "write", "read", "framing":
		default:
			return false
		}
		if r.memberAttempt == nil || r.memberAttempt.attempt != c.dispatch.attempt || r.memberAttempt.ordinal != n.completed || n.completed >= n.admission.members ||
			failure.requestWritten < 0 || failure.requestWritten > len(r.memberAttempt.member.request.wire) || failure.responseRead < 0 ||
			failure.responseRead > r.memberAttempt.member.capture.WireBytes-len(r.memberAttempt.member.request.wire)+1 {
			return false
		}
		n.failureStage = providerHistoryRecoveryCloneObserved(failure.failureStage, c.meter)
		n.failureWritten, n.failureReceived = failure.requestWritten, failure.responseRead
		if n.failureStage == "" || n.failureWritten < 0 || n.failureReceived < 0 {
			return false
		}
	}
	n.ack = &providerHistoryGroupAcknowledgment{c.group, c, c.frame.after}
	n.head = providerHistoryRecoveryCloneObserved(c.highwater, c.meter)
	n.frame.intent = providerHistoryRecoveryCloneObserved(c.frame.intent, c.meter)
	n.frame.commit = providerHistoryRecoveryCloneObserved(c.frame.commit, c.meter)
	n.frame.frontier = providerHistoryRecoveryCloneObserved(c.frame.frontier, c.meter)
	n.inventory.raw = providerHistoryRecoveryCloneObserved(c.inventory.raw, c.meter)
	if n.head == "" || n.frame.intent == "" || n.frame.commit == "" || n.frame.frontier == "" || n.inventory.raw == "" {
		return false
	}
	if n.frame.phase == "settle" {
		n.memberArchive = r.prepareMemberArchiveLocked(c, n)
		if n.memberArchive == nil {
			return false
		}
	}
	// Both original selections are published together, after every fallible
	// copy and before the first physical write. They do not acknowledge S.
	c.memberArchive = n.memberArchive
	c.checkpoint, r.pending, r.pendingAck = n, n, n.ack
	return true
}

// Scalar-only: this can run under the execution meter mutex without I/O or
// recursively observing that same meter. Raw-byte comparisons follow below.
func (r *providerHistoryRecoveryOwner) pendingCheckpointShapeLocked(c *providerHistoryWriteClaim) bool {
	if r == nil || r.owner != r || c == nil || c.owner != c || c.group != r.originalGroup || c.meter == nil || c.meter != r.originalMeter ||
		c.meter.owner != c.meter || c.meter.historyGroup != c.group || c.meter.historyRecovery != nil ||
		c.frame == nil || c.inventory == nil || c.coordinator == nil || r.pending == nil || r.pendingAck == nil {
		return false
	}
	n := r.pending
	return c.checkpoint == n && n.owner == n && n.recovery == r && n.group == c.group && n.claim == c && n.previous == r.confirmed &&
		n.admission.shapeLocked(r, n) && r.checkpointMemberArchiveShapeLocked(n) &&
		n.ack == r.pendingAck && n.ack.owner == c.group && n.ack.claim == c && n.ack.pin == n.pin && n.pin == c.frame.after &&
		n.scope == c.frame.scope && n.frame.scope == c.frame.scope && n.frame.before == c.frame.before && n.frame.after == c.frame.after &&
		n.frame.stage == c.frame.stage && n.frame.phase == c.frame.phase && n.frame.clock == c.frame.clock && n.frame.operation == c.frame.operation &&
		n.frame.previousInventory == c.frame.previousInventory && n.frame.inventory == c.frame.inventory &&
		n.frame.previousModel == c.frame.previousModel && n.frame.model == c.frame.model && n.model == c.coordinator.head &&
		n.inventory.previous == c.inventory.previous && n.inventory.digest == c.inventory.digest && n.inventory.count == c.inventory.count && n.inventory.entries == c.inventory.entries &&
		n.retention == c.group.retention && n.failedS == (c.failure.kind != "")
}

// Called before the final native sample, outside meter.mu. Each actual text
// comparison is observed; no unmetered raw comparison is hidden in hasOwner.
func (r *providerHistoryRecoveryOwner) pendingCheckpointDataMatchesLocked(c *providerHistoryWriteClaim) bool {
	if !r.pendingCheckpointShapeLocked(c) {
		return false
	}
	n := r.pending
	for _, pair := range [][2]string{{n.head, c.highwater}, {n.frame.intent, c.frame.intent}, {n.frame.commit, c.frame.commit}, {n.frame.frontier, c.frame.frontier}, {n.inventory.raw, c.inventory.raw}} {
		if !providerReplayTextEqual(pair[0], pair[1], c.meter) {
			return false
		}
	}
	return true
}

// This gate selects only originals that this same process sealed while
// healthy. It performs no I/O, promotes no pending frame and reads no cached
// claim catalog/model. Actual retained-byte reconstruction remains mandatory.
func (r *providerHistoryRecoveryOwner) checkpointSelectionLocked(g *providerHistoryGroupOwner) bool {
	if r == nil || !r.initialDescriptorShapeLocked(g) || r.parent == nil || g == nil || g.start == nil || r.store == nil || r.store.frontierFiles == nil || r.initial == nil || r.confirmed == nil || r.initial.owner != r.initial ||
		r.initial.recovery != r || r.initial.group != g || r.initial.previous != nil || r.initial.claim != nil || r.initial.ack != nil || r.initial.admission != nil || r.initial.memberArchive != nil ||
		r.initial.pin != g.start.main || r.initial.scope.initial != g.start.main ||
		r.initial.retention != (providerHistoryStorageCharge{}) ||
		r.confirmed.owner != r.confirmed || r.confirmed.recovery != r || r.confirmed.group != g || r.confirmed.pin != g.pin || r.confirmed.ack != g.last {
		return false
	}
	steps := uint64(0)
	for c := r.confirmed; c != r.initial; c = c.previous {
		if c == nil || c.owner != c || c.recovery != r || c.group != g || c.claim == nil || c.claim.owner != c.claim || c.claim.checkpoint != c || c.ack == nil ||
			c.ack.owner != g || c.ack.claim != c.claim || c.ack.pin != c.pin || c.previous == nil || c.previous.owner != c.previous ||
			c.previous.recovery != r || c.previous.group != g || c.frame.scope != r.initial.scope || c.frame.before != c.previous.pin || c.frame.after != c.pin ||
			!c.admission.shapeLocked(r, c) || !r.checkpointMemberArchiveShapeLocked(c) ||
			steps >= r.parent.bounds.records {
			return false
		}
		steps++
	}
	if r.pending != nil {
		p := r.pending
		return p.owner == p && p.recovery == r && p.group == g && p.previous == r.confirmed &&
			p.admission.shapeLocked(r, p) && r.checkpointMemberArchiveShapeLocked(p) &&
			p.claim != nil && p.claim.owner == p.claim && p.claim.checkpoint == p && p.claim == r.store.frontierFiles.historyPending && p.frame.before == r.confirmed.pin &&
			p.ack != nil && p.ack == r.pendingAck && p.ack != g.last && p.ack.owner == g && p.ack.claim == p.claim && p.ack.pin == p.pin
	}
	return r.pendingAck == nil && r.store.frontierFiles.historyPending == nil
}
