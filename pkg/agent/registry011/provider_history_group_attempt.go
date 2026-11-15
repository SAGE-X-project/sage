package registry011

import (
	"context"
	"crypto/tls"
	"time"
)

// This private original is installed under store.mu. A copied preparation,
// ACK, order or attempt cannot enter the wire path. The complete program still
// requires the pre-H0 production derivation; the fixture-only witness is not
// an operating cost certificate or Source constructor.
type providerHistoryDispatchAttempt struct {
	owner             *providerHistoryDispatchAttempt
	dispatch          *providerHistoryGroupDispatch
	ack               *providerHistoryGroupAcknowledgment
	ordinal           int
	member            providerHistoryGroupMember
	tls               providerTLSBounds
	begin, end        ClockSample
	connection        *tls.Conn
	used              bool
	files             int
	original          *providerHistoryMemberOriginal
	failure           *providerHistoryCaptureData
	finalMetadataWork candidateWorkVector
	endWork           candidateWorkVector
	finalMetadataRead uint64
	recoveryAttempt   *providerHistoryRecoveryMemberAttempt
	recoveryCapture   *providerHistoryRecoveryMemberCapture
	recoveryComplete  *providerHistoryRecoveryMemberComplete
}

func (*providerHistoryDispatchAttempt) UnmarshalJSON([]byte) error { return errProviderIntegration }

// These are nonserializable SOURCE-derivation rows, pinned before H0 by the
// same original witness as the whole program. No production factory supplies
// them yet. Zero/underived rows refuse BEFORE any member observation or I/O.
// A unit fixture may select its separate synthetic rows before its own H0;
// that exercises the owner, not production cost or host qualification.
type providerHistoryDispatchBounds struct {
	member, closure              candidateWorkVector
	memberClocks, closureClocks  uint64
	tailMS                       uint64
	advance                      candidateWorkVector
	advanceClocks, advanceTailMS uint64
}

func (b providerHistoryDispatchBounds) valid() bool {
	return b.member[0] == 1 && b.member[1] > 0 && b.member[2] > 0 && b.member[9] > 0 && b.member[10] > 0 &&
		b.closure[0] == 0 && b.closure[1] == 0 && b.closure[2] > 0 && b.closure[9] > 0 && b.closure[10] > 0 &&
		b.memberClocks >= 2 && b.closureClocks >= 2 && b.tailMS > 0 && b.tailMS < 300000
}

func (b providerHistoryDispatchBounds) remaining(count int) (candidateWorkVector, uint64, bool) {
	if !b.valid() || count <= 0 || count > providerHistoryGroupMembers {
		return candidateWorkVector{}, 0, false
	}
	work := b.closure
	for i, n := range b.member {
		product, ok := providerUpperProduct(uint64(count), n)
		value, sumOK := providerUpperSum(work[i], product)
		if !ok || !sumOK {
			return candidateWorkVector{}, 0, false
		}
		work[i] = value
	}
	memberClocks, a := providerUpperProduct(uint64(count), b.memberClocks)
	clocks, c := providerUpperSum(b.closureClocks, memberClocks)
	return work, clocks, a && c
}

// Scalar checks only: this predicate is also called while meter.mu is held.
// It must not sample clocks, read files, observe codecs or acquire that mutex.
func (d *providerHistoryGroupDispatch) activeMemberShapeLocked() bool {
	if d == nil || d.owner != d || d.group == nil || d.group.dispatch != d || d.group.owner != d.group ||
		d.acknowledgment != d.group.last || d.order == nil || d.order != d.group.order || !d.order.valid() || d.meter != d.group.meter ||
		d.coordinator == nil || d.custodian == nil || d.coordinator == d.custodian || d.group.coordinator == nil || d.group.custodian == nil ||
		d.coordinator.ticket == nil || d.custodian.ticket == nil || d.coordinator.consumption == nil || d.custodian.consumption == nil ||
		d.coordinator.ticket == d.custodian.ticket || d.coordinator.consumption == d.custodian.consumption ||
		d.coordinator.head != d.group.coordinator.head || d.custodian.head != d.group.custodian.head ||
		d.retained < 0 || d.retained >= d.order.count || d.retained != d.order.next || d.group.parent == nil || !d.memberReplaySelectedLocked() {
		return false
	}
	a := d.attempt
	if a == nil || a.owner != a || a.dispatch != d || a.ack != d.acknowledgment || a.ordinal != d.retained ||
		a.ordinal < 0 || a.ordinal >= d.coordinator.ticket.table.count ||
		a.member != d.coordinator.ticket.table.members[a.ordinal] || a.tls != d.group.parent.tls || a.files < 0 || a.files > 2 {
		return false
	}
	for i := range d.retained {
		r := d.originals[i]
		if r == nil || r.owner != r || r.dispatch != d || r.attempt == nil || r.attempt.owner != r.attempt ||
			r.attempt.dispatch != d || r.attempt.ordinal != i || !r.attempt.used || r.attempt.files != 2 ||
			r.attempt.original != r || r.attempt.connection != nil {
			return false
		}
	}
	if d.group.phase == "preparing-member" {
		return !a.used && a.files == 0 && a.connection == nil && a.original == nil && d.order.state == providerHistoryOrderConfirmed
	}
	if !d.group.parent.recovery.memberAttemptShapeLocked(a) || a.original != nil && !d.group.parent.recovery.memberCaptureShapeLocked(a) {
		return false
	}
	return a.used && d.order.state == providerHistoryOrderInFlight &&
		(d.group.phase == "capturing-member" && a.files == 0 || d.group.phase == "retaining-member")
}

// Enter only the next original ordinal. All detached/refused inputs fail before
// the first native sample, file observation, context timer or network effect.
// The invocation remains a candidate inside the unpublished transport/S bundle;
// production opening is blocked until the complete source derivation exists.
func (o *providerHistoryGroupOwner) prepareMemberCandidateLocked(ctx context.Context, d *providerHistoryGroupDispatch, ordinal int) (*providerHistoryDispatchAttempt, context.Context, context.CancelFunc, error) {
	if ctx == nil || ctx.Err() != nil || o == nil || o.owner != o || o.store == nil || o.parent == nil || o.parent.group != o || o.failed ||
		o.phase != "dispatch-prepared" || d == nil || d.owner != d || d != o.dispatch || d.group != o || d.acknowledgment != o.last ||
		d.order != o.order || !o.committedConsumptionShapeLocked() || o.order.state != providerHistoryOrderConfirmed ||
		d.retained != o.order.next || ordinal != d.retained || ordinal < 0 || ordinal >= o.order.count || d.attempt != nil ||
		d.meter != o.meter || o.meter == nil || o.meter.owner != o.meter || o.meter.failed || o.meter.closed ||
		o.store.activeMeter.Load() != nil || o.store.frontierFiles == nil || o.store.frontierFiles.pending != nil || o.store.frontierFiles.historyPending != nil ||
		o.parent.clock == nil || o.admission.claim.frame.clock != d.ticketClock || !o.parent.matchesLocked() ||
		o.parent.recovery == nil || !o.parent.recovery.memberSelectionLocked(o) ||
		!o.parent.dispatchBounds.valid() || !d.memberReplaySelectedLocked() ||
		d.coordinator == nil || d.coordinator.ticket == nil || d.custodian == nil || d.custodian.ticket == nil {
		return nil, nil, nil, errProviderIntegration
	}
	m := o.meter
	if m.historyGroup != o || m.historyRecovery != nil || m.integration != nil || m.scan != nil || m.scanBody != nil || m.firstAdmission != nil || m.firstOperation != nil ||
		m.guardian != nil || m.continuing != nil || m.parent != nil || !m.authority ||
		m.upper != o.parent.bounds.future.work || m.maximumClocks != o.parent.bounds.future.clocks {
		return nil, nil, nil, errProviderIntegration
	}
	if !d.wholeResultPackFitsLocked() || !d.failureOriginalFitsLocked() {
		return nil, nil, nil, errProviderIntegration
	}
	// Overflow is a detached-input refusal, not an entered invocation. Keep
	// this scalar arithmetic before phase/attempt installation or observation.
	remaining, clocks, remainingOK := o.parent.dispatchBounds.remaining(o.order.count - ordinal)
	if !remainingOK {
		return nil, nil, nil, errProviderIntegration
	}
	a := &providerHistoryDispatchAttempt{dispatch: d, ack: o.last, ordinal: ordinal,
		member: d.coordinator.ticket.table.members[ordinal], tls: o.parent.tls}
	a.owner, d.attempt, o.phase = a, a, "preparing-member"
	s := o.store
	s.activeMeter.Store(m)
	if !o.fits(m) || !o.identityLocked() || !d.memberCatalogLocked() {
		return nil, nil, nil, errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) || !providerHistoryGroupClockFits(d.ticketClock, providerClockDiagnostic(begin)) {
		return nil, nil, nil, errProviderIntegration
	}
	a.begin = begin
	left, err := d.replayMemberEquivalentLocked(ctx, 0)
	if err != nil {
		return nil, nil, nil, errProviderIntegration
	}
	right, err := d.replayMemberEquivalentLocked(ctx, 1)
	if err != nil || left == right || left.ticket == right.ticket || left.consumption == right.consumption ||
		!d.coordinator.validObserved(m) || !d.custodian.validObserved(m) ||
		!providerReplayTextEqual(left.head, right.head, m) || !providerHistoryGroupRequestEqualObserved(&left.ticket.table.members[ordinal].request, &a.member.request, m) ||
		left.ticket.table.members[ordinal].capture != a.member.capture || !o.identityLocked() || !d.memberCatalogLocked() {
		return nil, nil, nil, errProviderIntegration
	}
	if d.retained > 0 {
		roots := d.rootsObservedLocked()
		first, err := d.readOriginalPrefixCandidateLocked(d.retained, roots)
		if err != nil || first == nil {
			return nil, nil, nil, errProviderIntegration
		}
		second, err := d.readOriginalPrefixCandidateLocked(d.retained, roots)
		if err != nil || second == nil || first == second || first.pack == second.pack || !providerReplayTextEqual(first.pack.digest, second.pack.digest, m) {
			return nil, nil, nil, errProviderIntegration
		}
	}
	upper, ok := providerHistoryCaptureUpper(uint64(len(a.member.request.wire)), a.member.capture, a.tls)
	upper[0] = 1
	for i, n := range upper {
		if n > o.parent.dispatchBounds.member[i] {
			return nil, nil, nil, errProviderIntegration
		}
	}
	intent := o.parent.recovery.prepareMemberAttemptLocked(a)
	if !ok || intent == nil || !o.parent.recovery.memberAttemptDataMatchesLocked(intent, a) || !m.preflightWork(remaining) || m.clocks > m.maximumClocks || clocks > m.maximumClocks-m.clocks ||
		m.clocks < o.clockStart || m.clocks-o.clockStart > o.clockUpper || clocks > o.clockUpper-(m.clocks-o.clockStart) {
		return nil, nil, nil, errProviderIntegration
	}
	caller, _ := ctx.Deadline()
	c := o.parent.clock
	c.mu.Lock()
	claim := o.parent.clockClaim
	clockOK := c.owner == c && !c.failed && !c.historyRevoked && c.history == o.parent && o.parent.clockSelectionMatches(c) &&
		claim != nil && claim.owner == claim && claim.parent == o.parent && claim.meter == m && claim.used &&
		claim.expectedRead < ^uint64(0) && c.read == claim.expectedRead+1 && c.last == begin
	origin := c.origin
	c.mu.Unlock()
	// The same pre-H0 witness must derive the entire positive closure tail.
	// Caller diagnostics cannot choose or shorten it to obtain a network slot.
	deadline, fits := providerHistoryTransportDeadline(origin, o.parent.original, ClockSample(d.ticketClock), begin, o.parent.bounds.durationMS, a.member.capture.Timeout, time.Duration(o.parent.dispatchBounds.tailMS)*time.Millisecond, caller)
	if !clockOK || !fits || !o.order.begin(ordinal) {
		return nil, nil, nil, errProviderIntegration
	}
	a.used, o.phase = true, "capturing-member" // Irreversible before even dial/TLS.
	o.parent.recovery.memberAttempt, o.parent.recovery.sent = intent, intent.sent
	var request candidateWorkVector
	request[0] = 1
	if !m.add(request) {
		return nil, nil, nil, errProviderIntegration
	}
	bounded, cancel := context.WithDeadline(ctx, deadline)
	return a, bounded, cancel, nil
}

// Production has one fixed dial, no redirect, retry, callback or endpoint
// injection. The original attempt is consumed before this call is reached.
// Returning a retained private member ACK is not S or a protected RPC result.
func (o *providerHistoryGroupOwner) retainMemberCandidate(ctx context.Context, d *providerHistoryGroupDispatch, ordinal int) (*providerHistoryMemberOriginal, error) {
	return o.retainMemberCandidateWithConnection(ctx, d, ordinal, nil)
}

// Only package-owned normal TLS fixtures supply a connection. No exported
// constructor accepts it. Such fixtures prove the byte/retention boundary,
// not ordering before a handshake which the fixture has already performed.
func (o *providerHistoryGroupOwner) retainMemberCandidateWithConnection(ctx context.Context, d *providerHistoryGroupDispatch, ordinal int, owned *tls.Conn) (*providerHistoryMemberOriginal, error) {
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	entered, complete := false, false
	defer func() {
		activeEntry := o.owner == o && o.meter != nil && s.activeMeter.Load() == o.meter && d != nil && d == o.dispatch && d.group == o && d.attempt != nil
		if entered && !complete || activeEntry && (o.phase == "preparing-member" || o.phase == "capturing-member" || o.phase == "retaining-member") {
			o.failLocked()
		}
	}()
	a, bounded, cancel, err := o.prepareMemberCandidateLocked(ctx, d, ordinal)
	if err != nil {
		return nil, errProviderIntegration
	}
	entered = true
	defer cancel()
	conn := owned
	if conn == nil {
		conn, err = providerIntegrationDial(bounded, a.member.capture)
	}
	if err != nil || conn == nil {
		if conn != nil {
			_ = conn.Close()
		}
		// A real fixed dial failed before any HTTP byte. This never claims a
		// verified peer or successful reply. Clock/caller/budget uncertainty still
		// refuses failed S, leaving the original invocation quarantined.
		a.failure = &providerHistoryCaptureData{knownFailure: true, failureStage: "dial"}
		if o.retainFailedCaptureLocked(ctx, d, a) == nil {
			complete = true
		}
		return nil, errProviderIntegration
	}
	a.connection = conn
	defer func() {
		if a.connection != nil {
			_ = a.connection.Close()
			a.connection = nil
		}
	}()
	data, err := providerHistoryCaptureBytes(bounded, conn, a.member.capture, &a.member.request, o.meter, a.tls)
	if err != nil || data == nil || data.sample == nil || !data.sample.Complete || data.knownFailure || data.uncertain {
		a.failure = data // Preserve the actual bounded prefix; no retry/result.
		_ = conn.Close()
		a.connection = nil
		if data != nil && data.knownFailure && !data.uncertain && o.retainFailedCaptureLocked(ctx, d, a) == nil {
			complete = true
		}
		return nil, errProviderIntegration
	}
	o.phase = "retaining-member"
	original, err := a.retainOriginalCandidateLocked(data)
	if err != nil {
		return nil, errProviderIntegration
	}
	completion := o.parent.recovery.prepareMemberCompleteLocked(a, original)
	if completion == nil {
		return nil, errProviderIntegration
	}
	if !o.identityLocked() || !d.memberCatalogLocked() || !d.activeMemberShapeLocked() || !o.parent.recovery.memberAttemptDataMatchesLocked(a.recoveryAttempt, a) {
		return nil, errProviderIntegration
	}
	// Actual observer/counter markers, not caller diagnostics or authority.
	// After all final metadata, the native end sample may add ONLY its one
	// logical clock unit. No later file/codec observation may reach publication.
	a.finalMetadataWork = o.meter.actual
	c := o.parent.clock
	c.mu.Lock()
	a.finalMetadataRead = c.read
	c.mu.Unlock()
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(a.begin), providerClockDiagnostic(end)) ||
		!providerHistoryGroupClockFits(d.ticketClock, providerClockDiagnostic(end)) {
		return nil, errProviderIntegration
	}
	a.endWork = o.meter.actual
	expected, unitsOK := providerUpperSum(a.finalMetadataWork[10], 1)
	c.mu.Lock()
	read := c.read
	c.mu.Unlock()
	for i, value := range a.finalMetadataWork {
		if i != 10 && a.endWork[i] != value {
			return nil, errProviderIntegration
		}
	}
	if !unitsOK || a.endWork[10] != expected || a.finalMetadataRead == ^uint64(0) || read != a.finalMetadataRead+1 ||
		bounded.Err() != nil || !o.order.complete(ordinal) {
		return nil, errProviderIntegration
	}
	a.end = end
	// Logical C pin/last/models and the original hold/window remain unchanged.
	// Only the independently retained member prefix advances here.
	d.originals[ordinal], d.retained, d.attempt, o.phase = original, ordinal+1, nil, "dispatch-prepared"
	a.recoveryComplete = completion
	o.parent.recovery.members[ordinal], o.parent.recovery.completed = completion, ordinal+1
	o.parent.recovery.memberAttempt, o.parent.recovery.memberCapture, o.parent.recovery.memberOriginal = nil, nil, nil
	s.activeMeter.Store(nil)
	complete = true
	return original, nil
}
