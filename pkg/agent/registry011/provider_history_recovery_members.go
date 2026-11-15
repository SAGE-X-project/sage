package registry011

// These are sealed original-process facts, not permits or recovered ACKs.
// Request values are immutable Go strings; no mutable capture slice, model or
// catalog map is retained. The actual reader must still reconstruct T/C and
// every selected spool from retained bytes with its separate observer.
type providerHistoryRecoveryMemberAttempt struct {
	owner                 *providerHistoryRecoveryMemberAttempt
	recovery              *providerHistoryRecoveryOwner
	dispatch              *providerHistoryGroupDispatch
	attempt               *providerHistoryDispatchAttempt
	consumption           *providerHistoryRecoveryCheckpoint
	order                 *providerHistoryGroupOrder
	member                providerHistoryGroupMember
	ordinal, count        int
	stage                 uint64
	consumed, sent        uint32
	responseName, tlsName string
}

type providerHistoryRecoveryMemberCapture struct {
	owner                   *providerHistoryRecoveryMemberCapture
	intent                  *providerHistoryRecoveryMemberAttempt
	original                *providerHistoryMemberOriginal
	response, tls           string
	responseBytes, tlsBytes uint64
}

type providerHistoryRecoveryMemberComplete struct {
	owner   *providerHistoryRecoveryMemberComplete
	capture *providerHistoryRecoveryMemberCapture
	result  string
}

func (*providerHistoryRecoveryMemberAttempt) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}
func (*providerHistoryRecoveryMemberCapture) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}
func (*providerHistoryRecoveryMemberComplete) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

// Original healthy preparation, before order.begin/dial or any spool. Filename
// copies belong to the SAME execution meter; nothing spends the recovery row.
func (r *providerHistoryRecoveryOwner) prepareMemberAttemptLocked(a *providerHistoryDispatchAttempt) *providerHistoryRecoveryMemberAttempt {
	if r == nil || !r.unspentLocked() || a == nil || a.owner != a || a.dispatch == nil || a.used || a.files != 0 || a.original != nil || a.recoveryAttempt != nil ||
		r.originalDispatch != a.dispatch || r.originalDispatch.group != r.originalGroup || r.memberAttempt != nil || r.memberCapture != nil || r.memberOriginal != nil ||
		r.consumption == nil || r.consumption != r.confirmed || r.consumption.ack != a.ack || r.consumption.frame.phase != "consume" ||
		r.order == nil || r.order != a.dispatch.order || r.order.state != providerHistoryOrderConfirmed || !r.order.valid() ||
		r.consumed != r.order.consumed || r.sent != r.order.sent || r.completed != r.order.next || r.completed != a.ordinal ||
		a.ordinal < 0 || a.ordinal >= r.order.count || a.dispatch.retained != a.ordinal || r.members[a.ordinal] != nil ||
		r.originalGroup.phase != "preparing-member" || !a.dispatch.activeMemberShapeLocked() {
		return nil
	}
	sent, ok := providerHistoryMemberPrefix(a.ordinal + 1)
	if !ok {
		return nil
	}
	n := &providerHistoryRecoveryMemberAttempt{recovery: r, dispatch: a.dispatch, attempt: a, consumption: r.consumption,
		order: r.order, member: a.member, ordinal: a.ordinal, count: r.order.count, stage: a.dispatch.coordinator.ticket.table.stage,
		consumed: r.consumed, sent: sent}
	n.owner = n
	n.responseName = a.dispatch.originalNameLocked(a.ordinal, "response")
	n.tlsName = a.dispatch.originalNameLocked(a.ordinal, "tls")
	if n.responseName == "" || n.tlsName == "" {
		return nil
	}
	a.recoveryAttempt = n
	return n
}

// Scalar-only role checks, including the original separate selected pointer.
// These are safe when the original execution observer mutex is already held.
func (r *providerHistoryRecoveryOwner) memberAttemptShapeLocked(a *providerHistoryDispatchAttempt) bool {
	if r == nil || a == nil || r.memberAttempt == nil || r.order == nil || r.originalDispatch == nil || r.originalGroup == nil {
		return false
	}
	n := r.memberAttempt
	return n.owner == n && n.recovery == r && n.attempt == a && a.owner == a && a.recoveryAttempt == n &&
		n.dispatch == r.originalDispatch && a.dispatch == n.dispatch && n.dispatch.attempt == a &&
		n.order == r.order && r.originalGroup.order == r.order && n.consumption == r.consumption &&
		n.consumption != nil && n.consumption.ack == a.ack && n.dispatch.acknowledgment == a.ack &&
		n.ordinal == a.ordinal && n.ordinal == r.completed && n.count == r.order.count &&
		n.consumed == r.consumed && r.consumed == r.order.consumed && n.sent == r.sent && r.sent == r.order.sent &&
		n.member.request.inputDigest == a.member.request.inputDigest && n.member.request.paramsDigest == a.member.request.paramsDigest &&
		n.member.class == a.member.class && n.member.capture == a.member.capture && a.used
}

// Raw request/name comparisons are observed outside meter.mu. Callers use
// this before order.begin and again before original member/failed-S end.
func (r *providerHistoryRecoveryOwner) memberAttemptDataMatchesLocked(n *providerHistoryRecoveryMemberAttempt, a *providerHistoryDispatchAttempt) bool {
	if r == nil || n == nil || n.owner != n || n.recovery != r || a == nil || a.owner != a || a.recoveryAttempt != n || n.attempt != a ||
		n.dispatch != r.originalDispatch || a.dispatch != n.dispatch || r.originalMeter == nil || r.originalMeter != n.dispatch.meter {
		return false
	}
	m := r.originalMeter
	for _, pair := range [][2]string{{n.member.request.method, a.member.request.method}, {n.member.request.params, a.member.request.params},
		{n.member.request.head, a.member.request.head}, {n.member.request.body, a.member.request.body}, {n.member.request.wire, a.member.request.wire},
		{n.responseName, n.dispatch.originalNameLocked(a.ordinal, "response")}, {n.tlsName, n.dispatch.originalNameLocked(a.ordinal, "tls")}} {
		if pair[1] == "" || !providerReplayTextEqual(pair[0], pair[1], m) {
			return false
		}
	}
	return true
}

func (r *providerHistoryRecoveryOwner) prepareMemberCaptureLocked(a *providerHistoryDispatchAttempt, original *providerHistoryMemberOriginal) *providerHistoryRecoveryMemberCapture {
	if !r.memberAttemptShapeLocked(a) || r.memberCapture != nil || r.memberOriginal != nil || a.recoveryCapture != nil || a.original != nil || a.files != 0 ||
		original == nil || original.owner != original || original.attempt != a || original.dispatch != r.originalDispatch ||
		!lowerHash(original.response) || !lowerHash(original.tls) || original.responseBytes == 0 || original.tlsBytes == 0 {
		return nil
	}
	n := &providerHistoryRecoveryMemberCapture{intent: r.memberAttempt, original: original, response: original.response, tls: original.tls,
		responseBytes: original.responseBytes, tlsBytes: original.tlsBytes}
	n.owner = n
	return n
}

func (r *providerHistoryRecoveryOwner) memberCaptureShapeLocked(a *providerHistoryDispatchAttempt) bool {
	if !r.memberAttemptShapeLocked(a) || r.memberCapture == nil || r.memberOriginal == nil || a.recoveryCapture != r.memberCapture {
		return false
	}
	n, original := r.memberCapture, r.memberOriginal
	return n.owner == n && n.intent == r.memberAttempt && n.original == original && original.owner == original &&
		a.original == original && original.attempt == a && original.dispatch == r.originalDispatch &&
		n.response == original.response && n.tls == original.tls && n.responseBytes == original.responseBytes && n.tlsBytes == original.tlsBytes
}

func (r *providerHistoryRecoveryOwner) prepareMemberCompleteLocked(a *providerHistoryDispatchAttempt, original *providerHistoryMemberOriginal) *providerHistoryRecoveryMemberComplete {
	if !r.memberCaptureShapeLocked(a) || original != r.memberOriginal || a.recoveryComplete != nil || a.files != 2 || !lowerHash(original.result) {
		return nil
	}
	n := &providerHistoryRecoveryMemberComplete{capture: r.memberCapture}
	n.owner = n
	n.result = providerHistoryRecoveryCloneObserved(original.result, r.originalMeter)
	if n.result == "" {
		return nil
	}
	return n
}

// Terminal selection does not consult a healthy model or make failed order
// bookkeeping valid again. It checks exact original consumed/sent/completed
// facts and private object provenance before any actual recovery I/O.
func (r *providerHistoryRecoveryOwner) memberSelectionLocked(g *providerHistoryGroupOwner) bool {
	if r != nil && r.final != nil {
		original := r.recordedOwnerLocked()
		return original != nil && original.memberSelectionLocked(g)
	}
	if r == nil || g == nil || g != r.originalGroup || r.completed < 0 || r.completed > providerHistoryGroupMembers {
		return false
	}
	for ordinal := r.completed; ordinal < len(r.members); ordinal++ {
		if r.members[ordinal] != nil {
			return false
		}
	}
	if r.order == nil {
		return g.order == nil && r.consumption == nil && r.originalDispatch == nil && r.consumed == 0 && r.sent == 0 && r.completed == 0 &&
			r.memberAttempt == nil && r.memberCapture == nil && r.memberOriginal == nil
	}
	order := r.order
	all, ok := providerHistoryMemberPrefix(order.count)
	if !ok || order.owner != order || g.order != order || r.consumed != all || order.consumed != r.consumed || order.sent != r.sent || order.next != r.completed ||
		r.completed > order.count {
		return false
	}
	if r.originalDispatch == nil {
		return g.dispatch == nil && r.completed == 0 && r.sent == 0 && r.memberAttempt == nil && r.memberCapture == nil && r.memberOriginal == nil
	}
	d := r.originalDispatch
	if g.dispatch != d || d.owner != d || d.group != g || d.order != order || d.meter != r.originalMeter || d.retained != r.completed ||
		r.consumption == nil || r.consumption.owner != r.consumption || r.consumption.recovery != r || r.consumption.group != g || r.consumption.frame.phase != "consume" ||
		d.acknowledgment == nil || d.acknowledgment != r.consumption.ack || d.acknowledgment.claim == nil ||
		d.acknowledgment.claim != r.consumption.claim || d.acknowledgment.claim.checkpoint != r.consumption {
		return false
	}
	for ordinal := range r.completed {
		n := r.members[ordinal]
		if n == nil || n.owner != n || n.capture == nil || n.capture.owner != n.capture || n.capture.intent == nil || n.capture.intent.owner != n.capture.intent ||
			n.capture.intent.recovery != r || n.capture.intent.dispatch != d || n.capture.intent.consumption != r.consumption || n.capture.intent.ordinal != ordinal ||
			n.capture.intent.order != order || n.capture.intent.count != order.count ||
			n.capture.original == nil || n.capture.original != d.originals[ordinal] || n.capture.original.owner != n.capture.original ||
			n.capture.original.attempt != n.capture.intent.attempt || n.capture.intent.attempt == nil || n.capture.intent.attempt.owner != n.capture.intent.attempt ||
			n.capture.intent.attempt.recoveryAttempt != n.capture.intent || n.capture.intent.attempt.recoveryCapture != n.capture || n.capture.intent.attempt.recoveryComplete != n ||
			!n.capture.intent.attempt.used || n.capture.intent.attempt.files != 2 || n.capture.intent.attempt.connection != nil || n.capture.intent.attempt.original != n.capture.original ||
			n.capture.original.response != n.capture.response || n.capture.original.tls != n.capture.tls || n.capture.original.result != n.result ||
			n.capture.original.responseBytes != n.capture.responseBytes || n.capture.original.tlsBytes != n.capture.tlsBytes {
			return false
		}
	}
	if r.memberAttempt == nil {
		prefix, _ := providerHistoryMemberPrefix(r.completed)
		return r.sent == prefix && r.memberCapture == nil && r.memberOriginal == nil &&
			(d.attempt == nil || !d.attempt.used && d.attempt.files == 0 && d.attempt.original == nil)
	}
	if !r.memberAttemptShapeLocked(d.attempt) {
		return false
	}
	if r.memberCapture == nil {
		return r.memberOriginal == nil && d.attempt.recoveryCapture == nil && d.attempt.original == nil && d.attempt.files == 0
	}
	return d.attempt.files >= 0 && d.attempt.files <= 2 && r.memberCaptureShapeLocked(d.attempt)
}
