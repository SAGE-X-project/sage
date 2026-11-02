package registry011

// The archive keeps original healthy member facts for this exact S checkpoint.
// It retains no table, parsed response, catalog map or executable ledger. The
// array is copied by value; its pointed-to facts were separately sealed before
// each original dial/capture/completion and are never reused for another stage.
type providerHistoryRecoveryMemberArchive struct {
	owner       *providerHistoryRecoveryMemberArchive
	recovery    *providerHistoryRecoveryOwner
	checkpoint  *providerHistoryRecoveryCheckpoint
	consumption *providerHistoryRecoveryCheckpoint
	dispatch    *providerHistoryGroupDispatch
	order       *providerHistoryGroupOrder
	stage       uint64
	count       int
	members     [providerHistoryGroupMembers]*providerHistoryRecoveryMemberComplete
}

func (*providerHistoryRecoveryMemberArchive) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

// Fixed descriptor slots only, not a whole live-set/allocator certificate.
// This slot is larger than the fixed pointer array and scalar fields on the
// supported 32/64-bit targets. Existing immutable member facts/backing strings
// still need the separate complete-program retained-memory derivation.
const providerHistoryRecoveryArchiveSlotBytes uint64 = 4096

func providerHistoryRecoveryArchiveSlots(stages uint64) (uint64, bool) {
	if stages == 0 || stages > (1<<20)/3 {
		return 0, false
	}
	return providerUpperProduct(stages, providerHistoryRecoveryArchiveSlotBytes)
}

// This runs in the healthy writer before the first physical S write. A failed
// or unacknowledged S may retain these DATA facts but grants no next stage.
func (r *providerHistoryRecoveryOwner) prepareMemberArchiveLocked(c *providerHistoryWriteClaim, n *providerHistoryRecoveryCheckpoint) *providerHistoryRecoveryMemberArchive {
	if r == nil || !r.unspentLocked() || c == nil || c.owner != c || n == nil || n.owner != n || c.frame == nil || c.frame.phase != "settle" ||
		c.group != r.originalGroup || c.meter != r.originalMeter || c.group == nil || !c.group.activeLocked(c.meter) ||
		c.checkpoint != nil || r.pending != nil || r.pendingAck != nil || n.group != c.group ||
		c.memberArchive != nil || n.memberArchive != nil || n.recovery != r || n.claim != c || n.previous != r.consumption ||
		n.previous == nil || n.previous.frame.phase != "consume" || n.admission == nil || n.admission != n.previous.admission ||
		c.dispatch == nil || c.dispatch != r.originalDispatch || c.dispatch.order != r.order || c.consumed != n.previous.ack ||
		n.completed != r.completed || n.completed < 0 || n.completed > n.admission.members || !r.memberSelectionLocked(r.originalGroup) ||
		!n.failedS && n.completed != n.admission.members || n.failedS && n.completed >= n.admission.members {
		return nil
	}
	slots, ok := providerHistoryRecoveryArchiveSlots(r.parent.bounds.stages)
	if !ok || slots > r.parent.bounds.resident || !r.originalMeter.event(providerCopyEvent, providerHistoryRecoveryArchiveSlotBytes) {
		return nil
	}
	a := &providerHistoryRecoveryMemberArchive{recovery: r, checkpoint: n, consumption: n.previous, dispatch: c.dispatch,
		order: r.order, stage: n.frame.stage, count: n.completed, members: r.members}
	a.owner = a
	return a
}

// Scalar provenance only: neither the current stage's order/dispatch/member
// array nor a producer model can select an earlier stage's spools. This is safe
// under the original meter mutex and performs no observations or native work.
func (a *providerHistoryRecoveryMemberArchive) shapeLocked(r *providerHistoryRecoveryOwner, c *providerHistoryRecoveryCheckpoint) bool {
	if a == nil || a.owner != a || r == nil || a.recovery != r || c == nil || c.owner != c || c.memberArchive != a || a.checkpoint != c ||
		c.recovery != r || c.group != r.originalGroup || c.claim == nil || c.claim.owner != c.claim || c.claim.checkpoint != c || c.claim.memberArchive != a ||
		c.frame.phase != "settle" || a.consumption == nil || a.consumption != c.previous || a.consumption.owner != a.consumption ||
		a.consumption.recovery != r || a.consumption.group != c.group || a.consumption.frame.phase != "consume" ||
		c.admission == nil || c.admission != a.consumption.admission || a.stage != c.frame.stage || a.stage != a.consumption.frame.stage ||
		a.count != c.completed || a.count < 0 || a.count > providerHistoryGroupMembers || a.count > c.admission.members ||
		!c.failedS && a.count != c.admission.members || c.failedS && a.count >= c.admission.members ||
		a.dispatch == nil || a.dispatch != c.claim.dispatch || a.dispatch.owner != a.dispatch || a.dispatch.group != c.group || a.dispatch.meter != r.originalMeter ||
		a.dispatch.acknowledgment != a.consumption.ack || a.consumption.ack != c.claim.consumed ||
		a.order == nil || a.order != c.claim.order || a.order.owner != a.order || a.dispatch.order != a.order ||
		a.order.count != c.admission.members {
		return false
	}
	for i, sealed := range a.members {
		if i >= a.count {
			if sealed != nil {
				return false
			}
			continue
		}
		if sealed == nil || sealed.owner != sealed || sealed.capture == nil || sealed.capture.owner != sealed.capture ||
			sealed.capture.intent == nil || sealed.capture.intent.owner != sealed.capture.intent || !lowerHash(sealed.result) {
			return false
		}
		capture, intent := sealed.capture, sealed.capture.intent
		original, attempt := capture.original, intent.attempt
		if intent.recovery != r || intent.dispatch != a.dispatch || intent.consumption != a.consumption || intent.order != a.order ||
			intent.stage != a.stage || intent.ordinal != i || intent.count != a.order.count || intent.consumed != a.order.consumed ||
			original == nil || original.owner != original || original != a.dispatch.originals[i] || original.dispatch != a.dispatch ||
			attempt == nil || attempt.owner != attempt || attempt.dispatch != a.dispatch || attempt.ack != a.consumption.ack ||
			attempt.ordinal != i || !attempt.used || attempt.files != 2 || attempt.connection != nil || attempt.original != original || original.attempt != attempt ||
			attempt.recoveryAttempt != intent || attempt.recoveryCapture != capture || attempt.recoveryComplete != sealed ||
			intent.member.request.inputDigest != attempt.member.request.inputDigest || intent.member.request.paramsDigest != attempt.member.request.paramsDigest ||
			intent.member.class != attempt.member.class || intent.member.capture != attempt.member.capture ||
			original.response != capture.response || original.tls != capture.tls || original.result != sealed.result ||
			original.responseBytes != capture.responseBytes || original.tlsBytes != capture.tlsBytes {
			return false
		}
	}
	return true
}

func (r *providerHistoryRecoveryOwner) checkpointMemberArchiveShapeLocked(c *providerHistoryRecoveryCheckpoint) bool {
	if c == nil {
		return false
	}
	if c.frame.phase != "settle" {
		return c.memberArchive == nil && (c.claim == nil || c.claim.memberArchive == nil)
	}
	return c.memberArchive != nil && c.memberArchive.shapeLocked(r, c)
}

func (r *providerHistoryRecoveryOwner) settledMemberLocked(c *providerHistoryRecoveryCheckpoint, ordinal int) *providerHistoryRecoveryMemberComplete {
	if ordinal < 0 || !r.checkpointMemberArchiveShapeLocked(c) || c.memberArchive == nil || ordinal >= c.memberArchive.count {
		return nil
	}
	return c.memberArchive.members[ordinal]
}
