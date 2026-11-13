package registry011

import (
	"context"
	"testing"
)

// Explicit separate UNIT profile, selected before profile/anchor/H0. This
// funds at most three members in each of three stages without changing any
// enclosing policy cap. It is not a sufficient production cost derivation.
func providerHistoryMultistageMembersTestBounds(t *testing.T, b *providerIntegrationBounds, capture ProviderCaptureLimits, stages uint64, members int) {
	t.Helper()
	if b.history == nil || stages != 3 || members < 2 || members > 3 {
		t.Fatal("separate pre-H0 three-stage multi-member UNIT selection")
	}
	initial, ok := providerHistoryGroupInitializationUpper(*b)
	rpcs, a := providerUpperProduct(stages, uint64(members))
	wire, c := providerUpperProduct(rpcs, uint64(capture.WireBytes+1))
	rpcs, d := providerUpperSum(initial[0], rpcs)
	wire, e := providerUpperSum(initial[1], wire)
	if !ok || !a || !c || !d || !e {
		t.Fatal("checked whole UNIT request and wire arithmetic")
	}
	h := b.history
	h.future.work[0], h.future.work[1] = rpcs, wire
	h.derivation.snapshot = *h
	if !b.valid() {
		t.Fatal("pre-H0 UNIT selection exceeds unchanged enclosing policy")
	}
}

func historyMemberEquivalentPositiveUnit(t *testing.T, o *providerHistoryGroupOwner, d *providerHistoryGroupDispatch) {
	t.Helper()
	before, clock, sent, next := o.meter.actual, o.store.clock.read, o.order.sent, o.order.next
	historyMemberReplayEnter(t, o, d)
	var roles [2]*providerHistoryGroupLedger
	for role := range 2 {
		full, err := o.replayCommittedConsumptionLocked(context.Background())
		if err != nil || full == nil {
			t.Fatal("full original combined oracle", role, err)
		}
		got, err := d.replayMemberEquivalentLocked(context.Background(), role)
		if err != nil || got == nil || got == full || got == d.memberReplaySeal.models[role] || got.head != full.head ||
			got.stage != full.stage || got.sequence != full.sequence || got.charged != full.charged {
			t.Fatal("combined prefix and current-member semantic equivalence", role, err)
		}
		roles[role] = got
	}
	if roles[0] == roles[1] || roles[0].ticket == roles[1].ticket || roles[0].consumption == roles[1].consumption ||
		o.store.clock.read != clock || o.order.sent != sent || o.order.next != next || o.meter.actual[0] != before[0] ||
		o.meter.actual[1] != before[1] || o.meter.actual[10] <= before[10] || o.meter.failed {
		t.Fatal("read-only UNIT probe advanced or shared a live role")
	}
	// The UNIT-only read phase has consumed no order/clock/RPC. Restore only
	// its temporary phase/attempt selector; retain every observed work unit.
	d.attempt, o.phase = nil, "dispatch-prepared"
	o.store.activeMeter.Store(nil)
}

func TestProviderHistoryMemberReplayCombinesPriorStagesAndGrowingCurrentMembers(t *testing.T) {
	o := historyMultistageOwnerWithMemberEnvelope(t, 3, nil, 3)
	m, s, r := o.meter, o.store, o.parent.recovery
	start, origin, held, upper := o.start, o.parent.original, s.stored, m.upper
	var prior [2]*providerHistoryRecoveryMemberArchive
	for stage := uint64(1); stage <= 2; stage++ {
		out := historyMultistageSettle(t, o, stage)
		prior[stage-1] = r.confirmed.memberArchive
		if err := o.advanceStageCandidate(context.Background(), out.ack); err != nil {
			t.Fatal("original full S advance", stage, err)
		}
	}
	d := historyMultistageDispatchMembers(t, o, 3, 3)
	q, pin, ack, window, clockStart := d.memberReplaySeal, o.pin, o.last, o.windowStart, o.clockStart
	if len(q.reads[0].rows) != 74 || len(q.reads[1].rows) != 74 || d.order.count != 3 {
		t.Fatal("both original transcripts must contain two complete earlier T/C/S stages")
	}
	for ordinal := range 3 {
		historyMemberEquivalentPositiveUnit(t, o, d)
		member := d.coordinator.ticket.table.members[ordinal]
		response := `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
		if member.request.method == "web3_clientVersion" {
			response = `{"jsonrpc":"2.0","id":1,"result":"owned-node"}`
		}
		conn := providerIntegrationOwnedTLS(t, member.request.body, response)
		out, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, conn)
		prefix, ok := providerHistoryMemberPrefix(ordinal + 1)
		if err != nil || out == nil || !ok || d.retained != ordinal+1 || d.order.sent != prefix || d.memberReplaySeal != q ||
			!d.memberReplaySelectedLocked() || o.pin != pin || o.last != ack || o.windowStart != window || o.clockStart != clockStart ||
			o.start != start || o.parent.original != origin || s.stored != held || m.upper != upper || o.meter != m || m.failed || m.closed {
			t.Fatal("combined genuine member progression changed the original lifetime", ordinal, err)
		}
	}
	out, err := o.retainSettlementCandidate(context.Background(), d)
	if err != nil || out == nil || out.ack.claim.frame.stage != 3 || r.confirmed.memberArchive == nil || r.confirmed.memberArchive.count != 3 ||
		d.retained != 3 || d.order.sent != 7 || d.order.consumed != 7 || d.order.state != providerHistoryOrderClosed ||
		!q.revoked || q.ancestors != nil || q.claims != ([2]providerHistoryMemberClaimInput{}) || q.fixed.groupModels != ([2]*providerHistoryGroupLedger{}) || d.memberReplaySeal != nil ||
		r.memberReplaySeal != nil || d.replayReads != ([2]*providerHistoryReplayReads{}) || prior[0].count != 1 || prior[1].count != 1 ||
		o.start != start || o.parent.original != origin || s.stored != held || m.upper != upper || o.meter != m || m.failed || m.closed {
		t.Fatal("combined original S did not archive all facts and release only the cache", err)
	}
}
