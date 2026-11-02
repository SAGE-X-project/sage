package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func historyCompletedMemberFixture(t *testing.T, count int) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch) {
	t.Helper()
	o, d := historyMemberDispatchFixture(t, count)
	for ordinal := range count {
		member := d.coordinator.ticket.table.members[ordinal]
		response := `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
		if member.request.method == "web3_clientVersion" {
			response = `{"jsonrpc":"2.0","id":1,"result":"owned-node"}`
		}
		conn := providerIntegrationOwnedTLS(t, member.request.body, response)
		if original, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, conn); err != nil || original == nil {
			t.Fatal("genuine completed member", ordinal, err)
		}
	}
	if !d.completedMemberShapeLocked() || o.order.state != providerHistoryOrderConfirmed {
		t.Fatal("genuine completed prefix is not ready for S")
	}
	return o, d
}

func TestProviderHistorySettlementPublishesOnlyAfterOriginalReadbackAndCAS(t *testing.T) {
	for _, count := range []int{1, 3, providerHistoryGroupMembers} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			o, d := historyCompletedMemberFixture(t, count)
			s, m := o.store, o.meter
			before, oldAck, held, work, reads, retained := o.pin, o.last, s.stored, m.actual, s.clock.read, o.retention
			window, clockStart := o.windowStart, o.clockStart
			output, err := o.retainSettlementCandidate(context.Background(), d)
			if err != nil || output == nil || output.owner != output || output.group != o || output.count != count || output.raw == "" ||
				output.ack == nil || output.ack != o.last || output.ack == oldAck || output.ack.claim.dispatch != d || output.ack.claim.consumed != oldAck ||
				!output.ack.claim.used || !output.ack.claim.renamed || output.ack.pin != o.pin || o.pin != s.frontierFiles.pin ||
				o.pin.sequence != before.sequence+1 || o.pin.ordinal != before.ordinal+1 || o.phase != "settled" || o.failed ||
				o.coordinator == o.custodian || o.coordinator.phase != "settled" || o.custodian.phase != "settled" || o.coordinator.head != o.custodian.head ||
				o.coordinator.ticket != nil || o.custodian.ticket != nil || o.coordinator.consumption != nil || o.custodian.consumption != nil ||
				o.order.state != providerHistoryOrderClosed || !o.order.valid() || s.frontierFiles.historyPending != nil || s.activeMeter.Load() != nil ||
				s.stored != held || m.failed || m.closed || m.actual[0] != work[0] || m.actual[1] != work[1] || s.clock.read != reads+2 ||
				o.windowStart != window || o.clockStart != clockStart || o.retention.physical <= retained.physical {
				t.Fatal("S escaped its original CAS/readback/clock/one-use boundary", err)
			}
			frame := output.ack.claim.frame
			checkpoint := o.parent.recovery.confirmed
			if checkpoint == nil || checkpoint != output.ack.claim.checkpoint || checkpoint.ack != output.ack || checkpoint.previous.ack != oldAck || checkpoint.failedS ||
				checkpoint.frame != *frame || checkpoint.inventory != *output.ack.claim.inventory || o.parent.recovery.pending != nil || !o.parent.recovery.checkpointSelectionLocked(o) {
				t.Fatal("successful S selected a provisional or aliased checkpoint")
			}
			if frame.previousInventory != oldAck.claim.inventory.digest || frame.previousModel != oldAck.claim.coordinator.head ||
				output.ack.claim.inventory.previous != oldAck.claim.inventory.digest || output.ack.claim.inventory.digest == oldAck.claim.inventory.digest ||
				d.ticketClock != o.admission.claim.frame.clock ||
				!o.coordinator.valid() || !o.custodian.valid() {
				t.Fatal("S changed its original T clock or introduced a containing-inventory cycle")
			}
			for i := range count {
				if output.results[i].digest != d.originals[i].result || output.results[i].response == "" || output.results[i].result == "" {
					t.Fatal("returned result is not the independently replayed ordinal", i)
				}
			}
			for i := count; i < len(output.results); i++ {
				if output.results[i] != (providerHistoryMemberResult{}) {
					t.Fatal("unused result tail changed")
				}
			}
			for _, item := range []struct{ name, raw string }{{"HEAD", s.frontierFiles.pinRaw}, {providerHistoryStoredName(output.ack.claim.results, nil), output.raw}} {
				root := s.config.custodyDirectory
				if item.name != "HEAD" {
					root = s.config.evidenceDirectory
				}
				raw, err := os.ReadFile(filepath.Join(root, item.name))
				if err != nil || string(raw) != item.raw {
					t.Fatal("selected retained original differs from returned DATA", err)
				}
			}
			after, afterReads := m.actual, s.clock.read
			if again, err := o.retainSettlementCandidate(context.Background(), d); err == nil || again != nil || m.actual != after || s.clock.read != afterReads || o.last != output.ack {
				t.Fatal("duplicate S observed or replaced the committed result")
			}
			if json.Unmarshal([]byte(`{}`), &providerHistorySettledResults{}) == nil {
				t.Fatal("serialized DATA manufactured a live completion")
			}
		})
	}
}

func TestProviderHistorySettlementDetachedEntriesDoNotObserveOrClose(t *testing.T) {
	o, d := historyCompletedMemberFixture(t, 1)
	for _, mode := range []string{"nil-context", "cancelled", "nil-dispatch", "copied-owner", "copied-dispatch", "copied-ack", "wrong-next", "active-attempt", "authority", "changed-cap", "changed-ticket-clock"} {
		t.Run(mode, func(t *testing.T) {
			owner, input, ctx := o, d, context.Background()
			ack, next, attempt, authority, upper := d.acknowledgment, o.order.next, d.attempt, o.meter.authority, o.meter.upper
			ticketClock := d.ticketClock
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			case "nil-dispatch":
				input = nil
			case "copied-owner":
				copy := *o
				owner = &copy
			case "copied-dispatch":
				copy := *d
				input = &copy
			case "copied-ack":
				copy := *d.acknowledgment
				d.acknowledgment = &copy
			case "wrong-next":
				o.order.next = 0
			case "active-attempt":
				d.attempt = &providerHistoryDispatchAttempt{}
			case "authority":
				o.meter.authority = false
			case "changed-cap":
				o.meter.upper[10]--
			case "changed-ticket-clock":
				d.ticketClock.MonotonicMS++
			}
			actual, reads, phase, pin, last, retained := o.meter.actual, o.store.clock.read, o.phase, o.pin, o.last, o.retention
			out, err := owner.retainSettlementCandidate(ctx, input)
			d.acknowledgment, o.order.next, d.attempt, o.meter.authority, o.meter.upper = ack, next, attempt, authority, upper
			d.ticketClock = ticketClock
			if err == nil || out != nil || o.meter.actual != actual || o.store.clock.read != reads || o.phase != phase || o.pin != pin || o.last != last || o.retention != retained ||
				o.failed || o.store.failed || o.meter.failed || o.meter.closed || o.store.activeMeter.Load() != nil || o.store.frontierFiles.historyPending != nil || !d.completedMemberShapeLocked() {
				t.Fatal("detached S changed the otherwise-valid original", mode, err)
			}
		})
	}
}

func TestProviderHistorySettlementIncompletePrefixRefusesBeforeObservation(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 2)
	for completed := range 2 {
		actual, reads, pin, ack, retained := o.meter.actual, o.store.clock.read, o.pin, o.last, o.retention
		if d.retained != completed || o.order.next != completed {
			t.Fatal("genuine incomplete-prefix precursor", completed)
		}
		if out, err := o.retainSettlementCandidate(context.Background(), d); err == nil || out != nil || o.meter.actual != actual || o.store.clock.read != reads || o.pin != pin || o.last != ack || o.retention != retained || o.failed || o.store.frontierFiles.historyPending != nil {
			t.Fatal("missing member was silently shortened or observed", completed)
		}
		if completed == 0 {
			member := d.coordinator.ticket.table.members[0]
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
			if original, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn); err != nil || original == nil {
				t.Fatal("genuine one-of-two completion", err)
			}
		}
	}
}

func TestProviderHistorySettlementReplaysEveryClosedOriginalBeforeWriting(t *testing.T) {
	o, d := historyCompletedMemberFixture(t, 2)
	s, m := o.store, o.meter
	pin, ack, held, retained, sent, requests := o.pin, o.last, s.stored, o.retention, o.order.sent, m.actual[0]
	data := *d
	data.owner, data.meter = &data, nil
	path := filepath.Join(s.config.evidenceDirectory, data.originalNameLocked(0, "response"))
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 1
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := o.retainSettlementCandidate(context.Background(), d); err == nil || out != nil || !o.failed || !m.failed || !m.closed ||
		o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || s.stored != held || o.retention != retained || o.order.sent != sent || m.actual[0] != requests ||
		s.frontierFiles.historyPending != nil || s.activeMeter.Load() != nil || o.order.state != providerHistoryOrderFailed {
		t.Fatal("changed original yielded S, retransmission or restored allowance", err)
	}
}

func TestProviderHistorySettlementStorageAndPostCASFailureNeverAcknowledge(t *testing.T) {
	for _, mode := range []string{"result-sync", "pre-cas-original", "post-cas-original", "post-cas-cancel"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyCompletedMemberFixture(t, 1)
			s, m := o.store, o.meter
			pin, ack, held, retained, rawBefore, requests, sent := o.pin, o.last, s.stored, o.retention, s.retainedRaw, m.actual[0], o.order.sent
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reached := false
			disk := s.custody
			if mode == "result-sync" {
				disk = s.evidence
			}
			syncFile := disk.syncFile
			disk.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				c := s.frontierFiles.historyPending
				if reached || c == nil || c.frame == nil || c.frame.phase != "settle" {
					return nil
				}
				if mode == "result-sync" {
					if filepath.Base(file.Name()) != providerHistoryStoredName(c.results, nil) {
						return nil
					}
					reached = true
					return errProviderIntegration
				}
				if mode == "pre-cas-original" {
					if filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
						return nil
					}
				} else {
					info, err := file.Stat()
					if err != nil {
						return err
					}
					if !info.IsDir() {
						return nil
					}
					head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
					if err != nil || string(head) != c.highwater {
						return nil
					}
				}
				reached = true
				if mode == "post-cas-cancel" {
					cancel()
					return nil
				}
				return os.WriteFile(filepath.Join(s.config.evidenceDirectory, providerHistoryStoredName(c.settlement, nil)), []byte("inert owned S mismatch"), 0600)
			}
			out, err := o.retainSettlementCandidate(ctx, d)
			c := s.frontierFiles.historyPending
			if err == nil || out != nil || !reached || c == nil || c.used || !o.failed || !m.failed || !m.closed ||
				o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || s.stored != held || s.retainedRaw != rawBefore || o.retention.physical <= retained.physical ||
				m.actual[0] != requests || o.order.sent != sent || o.order.state != providerHistoryOrderFailed || s.activeMeter.Load() != nil {
				t.Fatal("uncertain S published, refunded or reset the original", mode, err)
			}
			r := o.parent.recovery
			if r.confirmed == nil || r.confirmed.ack != ack || r.confirmed.pin != pin || r.pending == nil || r.pending != c.checkpoint ||
				r.pending.previous != r.confirmed || r.pending.ack == ack || r.pending.pin != c.frame.after || r.pending.failedS || !r.checkpointSelectionLocked(o) {
				t.Fatal("uncertain physical S replaced the last confirmed C checkpoint")
			}
			post := mode == "post-cas-original" || mode == "post-cas-cancel"
			if c.renamed != post {
				t.Fatal("physical CAS and logical ACK were conflated")
			}
		})
	}
}
