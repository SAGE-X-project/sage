package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// The genuine owner has already performed first P, T, C and both full dispatch
// replays. This enters only its read-only preparation DATA phase for comparison
// with the unchanged full replay oracle. It does not claim a native clock,
// begin an order, reserve RPC work or open a connection.
func historyMemberReplayEnter(t *testing.T, o *providerHistoryGroupOwner, d *providerHistoryGroupDispatch) {
	t.Helper()
	if o.phase != "dispatch-prepared" || d.retained >= d.order.count || d.attempt != nil || !d.memberReplaySelectedLocked() {
		t.Fatal("genuine original dispatch prerequisite")
	}
	a := &providerHistoryDispatchAttempt{dispatch: d, ack: o.last, ordinal: d.retained,
		member: d.coordinator.ticket.table.members[d.retained], tls: o.parent.tls}
	a.owner = a
	d.attempt, o.phase = a, "preparing-member"
	o.store.activeMeter.Store(o.meter)
	if !o.activeLocked(o.meter) {
		t.Fatal("original read-only member phase")
	}
}

func TestProviderHistoryMemberReplayMatchesFullOriginalOracleForEachRole(t *testing.T) {
	for _, count := range []int{1, 3, providerHistoryGroupMembers} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, count)
			q := d.memberReplaySeal
			historyMemberReplayEnter(t, o, d)
			originalWork, clock := o.meter.actual, o.store.clock.read
			var outputs [2]*providerHistoryGroupLedger
			for role := range 2 {
				oracle, err := o.replayCommittedConsumptionLocked(context.Background())
				if err != nil || oracle == nil {
					t.Fatal("unchanged full semantic original oracle", role, err)
				}
				got, err := d.replayMemberEquivalentLocked(context.Background(), role)
				if err != nil || got == nil || got == oracle || got == q.models[role] ||
					got.head != oracle.head || got.previous != oracle.previous || got.charged != oracle.charged ||
					got.ticket == q.models[role].ticket || got.consumption == q.models[role].consumption ||
					!got.valid() || o.failed || o.meter.failed {
					t.Fatal("fresh originals did not yield independently owned equivalent DATA", role, err)
				}
				outputs[role] = got
			}
			if outputs[0] == outputs[1] || outputs[0].ticket == outputs[1].ticket || outputs[0].consumption == outputs[1].consumption ||
				o.store.clock.read != clock || o.meter.actual[0] != originalWork[0] || o.meter.actual[1] != originalWork[1] ||
				o.meter.actual[10] <= originalWork[10] || o.order.sent != 0 || o.order.next != 0 || d.retained != 0 {
				t.Fatal("read equivalence shared a mutable role or claimed clock/network/order work")
			}
		})
	}
}

func TestProviderHistoryMemberReplayChecksPriorStagesAndReleasesOnlyTheCache(t *testing.T) {
	o := historyMultistageOwner(t, 3)
	r := o.parent.recovery
	for stage := uint64(1); stage <= 2; stage++ {
		d := historyMultistageDispatch(t, o, stage)
		historyRetainNormalPrefix(t, o, d, 1)
		out, err := o.retainSettlementCandidate(context.Background(), d)
		if err != nil || out == nil || d.memberReplaySeal != nil || d.replayReads != ([2]*providerHistoryReplayReads{}) || r.memberReplaySeal != nil ||
			d.coordinator == nil || d.custodian == nil || d.order.state != providerHistoryOrderClosed || d.order.sent != 1 || d.order.consumed != 1 || d.retained != 1 {
			t.Fatal("S must revoke only the replay cache and preserve archived consumed originals", stage, err)
		}
		if err := o.advanceStageCandidate(context.Background(), out.ack); err != nil || r.memberReplaySeal != nil {
			t.Fatal("full original S replay and next-stage transition", stage, err)
		}
	}
	d := historyMultistageDispatch(t, o, 3)
	if len(d.replayReads[0].rows) != 74 || len(d.replayReads[1].rows) != 74 {
		t.Fatal("two complete prior one-member T/C/S stages must remain in both input transcripts")
	}
	historyMemberReplayEnter(t, o, d)
	for role := range 2 {
		oracle, err := o.replayCommittedConsumptionLocked(context.Background())
		if err != nil {
			t.Fatal("prior archived original replay", err)
		}
		got, err := d.replayMemberEquivalentLocked(context.Background(), role)
		if err != nil || got == nil || got.head != oracle.head || got.stage != 2 || got.sequence != 8 {
			t.Fatal("prior-stage semantics omitted", role, err)
		}
	}
}

func TestProviderHistoryMemberReplayRefreshesEveryOriginalCategoryForBothRoles(t *testing.T) {
	for _, kind := range []string{"initial-frontier", "prior-request", "prior-results", "current-request", "current-consumption", "current-frontier", "HEAD"} {
		for role := range 2 {
			t.Run(kind+"-"+strconv.Itoa(role), func(t *testing.T) {
				o := historyMultistageOwner(t, 2)
				old := historyMultistageSettle(t, o, 1)
				priorRequest := old.ack.claim.checkpoint.previous.previous.inventory.entries[0]
				priorResults := old.ack.claim.results
				if err := o.advanceStageCandidate(context.Background(), old.ack); err != nil {
					t.Fatal(err)
				}
				d := historyMultistageDispatch(t, o, 2)
				root, name := o.store.config.custodyDirectory, "HEAD"
				switch kind {
				case "initial-frontier":
					name = o.store.name("frontier", o.start.main.sequence)
				case "prior-request":
					root, name = o.store.config.evidenceDirectory, providerHistoryStoredName(priorRequest, nil)
				case "prior-results":
					root, name = o.store.config.evidenceDirectory, providerHistoryStoredName(priorResults, nil)
				case "current-request":
					root, name = o.store.config.evidenceDirectory, providerHistoryStoredName(o.admission.claim.request, nil)
				case "current-consumption":
					root, name = o.store.config.evidenceDirectory, providerHistoryStoredName(o.last.claim.consumption, nil)
				case "current-frontier":
					name = o.store.name("frontier", o.pin.sequence)
				}
				// Only an owned temporary fixture file is changed. Same length
				// prevents length checks from substituting for byte verification.
				path := filepath.Join(root, name)
				raw, err := os.ReadFile(path)
				if err != nil || len(raw) == 0 {
					t.Fatal("real original prerequisite", err)
				}
				raw[len(raw)-1] ^= 1
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				historyMemberReplayEnter(t, o, d)
				before, clock := o.meter.actual, o.store.clock.read
				got, err := d.replayMemberEquivalentLocked(context.Background(), role)
				if err == nil || got != nil || o.meter.actual[0] != before[0] || o.meter.actual[1] != before[1] ||
					o.meter.actual[10] <= before[10] || o.store.clock.read != clock || o.order.sent != 0 {
					t.Fatal("changed original was accepted or read work refunded", kind, role, err)
				}
			})
		}
	}
}

func TestProviderHistoryMemberReplayRejectsDetachedOrChangedBindings(t *testing.T) {
	for _, mode := range []string{"missing", "copied", "repaired-copy", "registration", "dispatch-copy", "role-swap", "record-copy", "window", "clock-window", "fixed-methods", "start", "checkpoint", "cutoff"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			historyMemberReplayEnter(t, o, d)
			if positive, err := d.replayMemberEquivalentLocked(context.Background(), 0); err != nil || positive == nil {
				t.Fatal("genuine positive preparing-member control before one mutation", err)
			}
			q := d.memberReplaySeal
			switch mode {
			case "missing":
				d.memberReplaySeal = nil
			case "copied", "repaired-copy":
				copy := *q
				if mode == "repaired-copy" {
					copy.owner = &copy
				}
				d.memberReplaySeal = &copy
			case "registration":
				o.parent.recovery.memberReplaySeal = nil
			case "dispatch-copy":
				copy := *d
				copy.owner = &copy
				d, o.dispatch = &copy, &copy
			case "role-swap":
				d.replayReads[0], d.replayReads[1] = d.replayReads[1], d.replayReads[0]
			case "record-copy":
				copy := *d.replayReads[0]
				copy.owner = &copy
				d.replayReads[0] = &copy
			case "window":
				o.windowStart[10]++
			case "clock-window":
				o.clockStart++
			case "fixed-methods":
				o.parent.methods.scalar.body++
			case "start":
				o.start.exchange = admissionTestDigest("f")
			case "checkpoint":
				o.parent.recovery.consumption.model = admissionTestDigest("f")
			case "cutoff":
				o.stageBefore.model = admissionTestDigest("f")
			}
			before, clock := o.meter.actual, o.store.clock.read
			if out, err := d.replayMemberEquivalentLocked(context.Background(), 0); err == nil || out != nil ||
				o.store.clock.read != clock || o.order.sent != 0 {
				t.Fatal("detached binding observed work or authorized a request", mode, err)
			}
			for i, n := range before {
				if o.meter.actual[i] < n {
					t.Fatal("refusal refunded prior work", mode, i)
				}
			}
			switch mode {
			case "missing", "copied", "repaired-copy", "registration", "dispatch-copy", "role-swap", "record-copy", "window", "clock-window":
				if o.meter.actual != before {
					t.Fatal("pointer/numeric detachment must precede observation", mode)
				}
			}
		})
	}
}

func TestProviderHistoryMemberReplayRefusesChangedBaselineModelAndWindowQuota(t *testing.T) {
	for _, mode := range []string{"row", "recorded-model", "model", "coherent-model", "coherent-cached-roots", "window-quota", "cancelled", "invalid-role", "wrong-phase"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			historyMemberReplayEnter(t, o, d)
			ctx, role := context.Background(), 0
			q := d.memberReplaySeal
			switch mode {
			case "row":
				q.reads[0].rows[len(q.reads[0].rows)-1].digest = admissionTestDigest("f")
			case "recorded-model":
				q.reads[0].model = admissionTestDigest("f")
			case "model", "coherent-model":
				q.models[0].charged.work[10]++
				if mode == "coherent-model" {
					q.models[0].head = providerReplayDigest(providerHistoryGroupRootDomain, q.models[0].root(), nil)
					if !q.models[0].valid() {
						t.Fatal("self-consistent changed model prerequisite")
					}
				}
			case "coherent-cached-roots":
				// Typed UNIT mutation only: even mutually consistent private
				// cached roots cannot replace the actual retained C model.
				for _, ledger := range []*providerHistoryGroupLedger{d.coordinator, d.custodian, o.coordinator, o.custodian} {
					ledger.charged.work[10]++
					ledger.head = providerReplayDigest(providerHistoryGroupRootDomain, ledger.root(), nil)
					if !ledger.valid() {
						t.Fatal("consistent cached DATA precursor")
					}
				}
				for role, reads := range q.reads {
					q.modelRoots[role], reads.model = q.models[role].head, q.models[role].head
					reads.root = reads.dataRootObserved(nil)
					q.readRoots[role] = reads.root
				}
			case "window-quota":
				// Monotone UNIT prefix INSIDE the original T window. This
				// checks entered exhaustion, not a specific late codec edge.
				end, ok := providerUpperSum(o.windowStart[10], o.windowUpper[10])
				if !ok || end < 64 || end-64 < o.meter.actual[10] || end-64 > o.meter.upper[10] {
					t.Fatal("valid original-window exhaustion precursor")
				}
				o.meter.actual[10] = end - 64
				if !o.fitsWork(o.meter, candidateWorkVector{}) || !o.meter.preflightWork(candidateWorkVector{}) {
					t.Fatal("quota precursor was already outside the selected window")
				}
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			case "invalid-role":
				role = 2
			case "wrong-phase":
				o.phase = "dispatch-prepared"
			}
			before, clock := o.meter.actual, o.store.clock.read
			out, err := d.replayMemberEquivalentLocked(ctx, role)
			if err == nil || out != nil || o.order.sent != 0 || o.order.next != 0 || o.store.clock.read != clock {
				t.Fatal("bad input yielded semantic or transport authority", mode, err)
			}
			for i, n := range before {
				if o.meter.actual[i] < n {
					t.Fatal("entered refusal refunded a coordinate", mode, i)
				}
			}
			if mode == "window-quota" && (!o.meter.failed || o.meter.actual[10] <= before[10]) {
				t.Fatal("valid original window did not retain entered work and sticky exhaustion")
			}
		})
	}
}

func TestProviderHistoryMemberReplayEnteredFailureQuarantinesWithoutDispatch(t *testing.T) {
	for _, subject := range []string{"old-request", "completed-member"} {
		t.Run(subject, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 3)
			ordinal := 0
			name := providerHistoryStoredName(o.admission.claim.request, nil)
			if subject == "completed-member" {
				historyRetainNormalPrefix(t, o, d, 1)
				ordinal = 1
				name = o.parent.recovery.members[0].capture.intent.responseName
			}
			path := filepath.Join(o.store.config.evidenceDirectory, name)
			raw, err := os.ReadFile(path)
			if err != nil || len(raw) == 0 {
				t.Fatal("owned original prerequisite", err)
			}
			raw[len(raw)-1] ^= 1
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			member := d.coordinator.ticket.table.members[ordinal]
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"owned-node"}`)
			defer conn.Close()
			before, sent := o.meter.actual, o.order.sent
			out, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, conn)
			if err == nil || out != nil || !o.failed || !o.store.failed || !o.meter.failed || !o.meter.closed ||
				d.memberReplaySeal != nil || o.parent.recovery.memberReplaySeal != nil || d.replayReads != ([2]*providerHistoryReplayReads{}) ||
				o.meter.actual[0] != before[0] || o.meter.actual[1] != before[1] || o.order.sent != sent || d.retained != ordinal ||
				o.store.activeMeter.Load() != nil {
				t.Fatal("entered verification failure dispatched, refunded or left a reusable cache", subject, err)
			}
		})
	}
}

func TestProviderHistoryMemberReplayPreservesGrowingMembersAndFailureRecovery(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 3)
	q, retention, actual := d.memberReplaySeal, o.retention, o.meter.actual
	historyRetainNormalPrefix(t, o, d, 1)
	if !d.memberReplaySelectedLocked() || d.memberReplaySeal != q || d.retained != 1 || o.retention == retention || o.meter.actual == actual ||
		q.consumption.retention != o.parent.recovery.consumption.retention {
		t.Fatal("fixed C binding froze legitimate member retention or cumulative work")
	}
	for ordinal := 1; ordinal < 3; ordinal++ {
		member := d.coordinator.ticket.table.members[ordinal]
		response := `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
		if member.request.method == "web3_clientVersion" {
			response = `{"jsonrpc":"2.0","id":1,"result":"owned-node"}`
		}
		conn := providerIntegrationOwnedTLS(t, member.request.body, response)
		if got, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, conn); err != nil || got == nil {
			t.Fatal("next original member", ordinal, err)
		}
	}
	if !d.memberReplaySelectedLocked() || d.retained != 3 {
		t.Fatal("later original members lost the same-C binding")
	}
	o.failLocked()
	if d.memberReplaySeal != nil || d.replayReads != ([2]*providerHistoryReplayReads{}) || o.parent.recovery.memberReplaySeal != nil ||
		d.retained != 3 || o.order.sent != 7 || o.order.consumed != 7 || !o.parent.recovery.memberSelectionLocked(o) {
		t.Fatal("failure retained an eligible cache or erased original recovery facts")
	}
	before := o.meter.actual
	if out, err := d.replayMemberEquivalentLocked(context.Background(), 0); err == nil || out != nil || o.meter.actual != before {
		t.Fatal("revoked cache became usable again")
	}
	var decoded providerHistoryMemberReplaySeal
	if json.Unmarshal([]byte(`{}`), &decoded) == nil {
		t.Fatal("serialized data must not reconstruct a live replay binding")
	}
}

func TestProviderHistoryMemberReplayCopyOwnsBehaviorOutsideSemanticRoot(t *testing.T) {
	// This copy check needs an allocated semantic window, not a fabricated
	// post-ancestry ledger. Select the smallest complete synthetic interval
	// before constructing its cursor and enter the window through ordinary
	// original-shaped T/C/S DATA. No Source or live dispatch is constructed.
	in := replayTestInput(t)
	in.Blocks, in.History = in.Blocks[:1], nil
	in.Anchor = journalTestHeader(t, in.Blocks[0])
	c, _, methods := historyCollectionContentInputFixture(t, 1, in, SepoliaAmsterdamRootExecutionGasProfile)
	if !c.initial() {
		t.Fatal("semantic copy fixture must start at the genuine initial DATA cut")
	}
	l := historyCollectionLedger(t, c)
	settlement, results := historyCollectionPendingSettlement(t, l, nil, methods)
	m := l.collection.semantic.observer
	if err := l.settleOriginalsObserved(context.Background(), settlement, results, methods, m); err != nil ||
		l.stage != 1 || l.phase != "settled" || l.collection.completed != 1 || l.collection.phase != "content" ||
		l.collection.semantic.window == nil || !l.validObserved(m) || c.completed != 0 || !c.initial() {
		t.Fatal("normal initial T/C/S DATA did not open an independent semantic window", err)
	}
	f := l.collection.semantic
	w := f.window
	contractMethods, outcomeEvents := len(w.run.contract.Methods), len(w.run.outcomes.Events)
	for key := range w.run.contract.Methods {
		delete(w.run.contract.Methods, key)
	}
	for key := range w.run.outcomes.Events {
		delete(w.run.outcomes.Events, key)
	}
	w.run.steps = make([]HistoryContentStep, 0, 17)
	if !l.validObserved(f.observer) || contractMethods == 0 || outcomeEvents == 0 {
		t.Fatal("non-root ABI/backing DATA prerequisite")
	}
	a := providerHistoryMemberReplayCopyObserved(l, f.observer)
	b := providerHistoryMemberReplayCopyObserved(l, f.observer)
	if a == nil || b == nil || a == b || a.collection == b.collection || a.collection.semantic.window == b.collection.semantic.window ||
		a.head != l.head || b.head != l.head || len(a.collection.semantic.window.run.contract.Methods) != contractMethods ||
		len(b.collection.semantic.window.run.outcomes.Events) != outcomeEvents || cap(a.collection.semantic.window.run.steps) != 0 ||
		len(w.run.contract.Methods) != 0 || cap(w.run.steps) != 17 {
		t.Fatal("owned copy trusted or rewrote non-root behavioral fields")
	}
	for key := range a.collection.semantic.window.run.contract.Methods {
		delete(a.collection.semantic.window.run.contract.Methods, key)
	}
	if len(b.collection.semantic.window.run.contract.Methods) != contractMethods {
		t.Fatal("independent copies shared mutable ABI maps")
	}
	if providerHistoryMemberReplayCopyObserved(nil, f.observer) != nil || providerHistoryMemberReplayCopyObserved(l, nil) != nil {
		t.Fatal("missing original DATA/meter admitted")
	}
}
