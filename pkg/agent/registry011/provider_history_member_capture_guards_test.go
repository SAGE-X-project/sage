package registry011

import (
	"context"
	"reflect"
	"testing"
)

func TestProviderHistoryCaptureNormalOwnedPeerHonorsSelectedRefusalLimits(t *testing.T) {
	for _, mode := range []string{"certificate-limit", "closed-connection", "header-limit"} {
		t.Run(mode, func(t *testing.T) {
			request := historyGroupSimple(t)
			_, limits := historyGroupBounds()
			certificates := providerTLSBounds{2, 4, 8 << 10}
			if mode == "certificate-limit" {
				certificates.certificateBytes = 1
			}
			if mode == "header-limit" {
				limits.HeaderBytes = 1
			}
			if !limits.valid() || !certificates.valid() {
				t.Fatal("finite selected primitive limits required")
			}
			// The peer always sends the ordinary valid response. Only this
			// separate BYTE test's limits or its own connection lifetime differ.
			conn := providerIntegrationOwnedTLS(t, request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
			if mode == "closed-connection" {
				if err := conn.Close(); err != nil {
					t.Fatal("close the test-owned connection", err)
				}
			}
			m := historyCaptureUnitMeter(limits)
			upper, ok := providerHistoryCaptureUpper(uint64(len(request.wire)), limits, certificates)
			if !ok {
				t.Fatal("positive closed primitive envelope required")
			}
			ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
			defer cancel()
			data, err := providerHistoryCaptureBytes(ctx, conn, limits, request, m, certificates)
			if err == nil || m.closed || m.actual[0] != 0 || m.actual[9] != 0 || m.actual[10] == 0 || m.clocks != 0 ||
				m.historyGroup != nil || m.firstOperation != nil || m.authority {
				t.Fatal("refused byte primitive acquired an attempt, storage, clock or execution role", mode, err)
			}
			if mode == "header-limit" {
				if data == nil || data.sample == nil || !data.knownFailure || data.uncertain || data.failureStage != "framing" ||
					data.requestWritten != len(request.wire) || data.responseRead <= 0 || data.responseRead != len(data.sample.ResponseWire) ||
					data.sample.Complete || data.sample.CaptureError == "" || data.sample.Result != nil || data.sample.ResponseBody != nil ||
					m.failed || m.actual[1] != uint64(data.requestWritten+data.responseRead) || m.actual[2] != uint64(data.responseRead) {
					t.Fatal("normal response exceeding the selected header limit lost its charged original failure prefix", err)
				}
			} else if data != nil || !m.failed || m.actual[1] != 0 {
				t.Fatal("certificate or closed-connection refusal reached the request write", mode, err)
			}
			for coordinate, actual := range m.actual {
				if actual > upper[coordinate] {
					t.Fatal("refused primitive exceeded its unchanged envelope", mode, coordinate, actual, upper[coordinate])
				}
			}
		})
	}
}

func TestProviderHistoryMemberPreselectedInsufficientRowsRefuseBeforeWire(t *testing.T) {
	for _, mode := range []string{"capture-wire", "remaining-work", "closure-tail"} {
		t.Run(mode, func(t *testing.T) {
			_, capture := providerFrontierBoundsFixture()
			rows := providerHistoryDispatchBounds{
				member:  candidateWorkVector{0: 1, 1: uint64(capture.WireBytes + 1), 2: 4 << 20, 9: 128 << 10, 10: 1 << 48},
				closure: candidateWorkVector{2: 32 << 20, 9: 1 << 20, 10: 1 << 50}, memberClocks: 2, closureClocks: 2, tailMS: 100,
			}
			switch mode {
			case "capture-wire":
				rows.member[1] = 1
			case "remaining-work":
				rows.member[10] = 1 << 61
			case "closure-tail":
				rows.tailMS = 299999
			}
			// Every different numeric row is selected before this fixture's H0.
			// No live owner, cap, role, original clock or counter is replaced.
			o, d := historyMemberDispatchFixtureWithRows(t, 1, &rows)
			s, m := o.store, o.meter
			s.mu.Lock()
			selected := o.parent.matchesLocked() && d.memberReplaySelectedLocked()
			s.mu.Unlock()
			if _, _, ok := rows.remaining(1); !ok || !selected || o.parent.dispatchBounds != rows || !rows.valid() {
				t.Fatal("otherwise-valid original selected finite rows required")
			}
			member := d.coordinator.ticket.table.members[0]
			if mode == "closure-tail" && uint64(member.capture.Timeout.Milliseconds()) >= rows.tailMS {
				t.Fatal("selected positive closure tail must exceed the whole member slot")
			}
			// Supply only a normal owned connection; the expected refusal must
			// happen before using it. This test never invokes the official dial.
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
			before, reads, clocks := m.actual, s.clock.read, m.clocks
			held, retention, pin, ack := s.stored, o.retention, o.pin, o.last
			originals := providerHistoryProvisionedOriginalBytes(t, s.config)
			got, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn)
			// A refused native entry read also burns its read counters. Require
			// the recorded successful begin, not counters alone, before judging
			// the later selected-row refusal. Quarantine may revoke this clock.
			s.clock.mu.Lock()
			clockLast, clockFailed, nativeRefusal := s.clock.last, s.clock.failed, s.clock.nativeRefusal
			clockID, epoch := s.clock.id, s.clock.epoch
			s.clock.mu.Unlock()
			if err == nil || got != nil || !o.failed || !s.failed || !m.failed || !m.closed || m.actual[0] != before[0] ||
				m.actual[1] != before[1] || m.actual[9] != before[9] || m.actual[10] <= before[10] ||
				s.clock.read != reads+1 || m.clocks != clocks+1 || s.stored != held || o.retention != retention || o.pin != pin || o.last != ack ||
				d.attempt == nil || d.attempt.owner != d.attempt || d.attempt.begin == (ClockSample{}) ||
				d.attempt.begin != clockLast || clockFailed || nativeRefusal != 0 || d.attempt.begin.ClockID != clockID || d.attempt.begin.Epoch != epoch ||
				!candidateLedgerClockAdvance(providerClockDiagnostic(ClockSample(d.ticketClock)), providerClockDiagnostic(d.attempt.begin)) || d.attempt.used || d.attempt.files != 0 || d.attempt.connection != nil || d.attempt.original != nil ||
				d.retained != 0 || d.originals[0] != nil || o.order.sent != 0 || o.order.next != 0 || o.order.state != providerHistoryOrderFailed ||
				s.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("insufficient selected rows dispatched, acknowledged, refunded or replaced original state", mode, err)
			}
			intent := d.attempt.recoveryAttempt
			if mode == "capture-wire" {
				if intent != nil {
					t.Fatal("insufficient capture row reached later original recovery preparation")
				}
			} else if intent == nil || intent.owner != intent || intent.attempt != d.attempt || intent.dispatch != d || intent.member != member ||
				intent.consumption != ack.claim.checkpoint || intent.ordinal != 0 || intent.count != 1 {
				t.Fatal("later selected-row refusal did not finish both original replay prerequisites", mode)
			}
			after, afterReads := m.actual, s.clock.read
			if got, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, nil); err == nil || got != nil || m.actual != after || s.clock.read != afterReads {
				t.Fatal("quarantined original member was retried", mode)
			}
		})
	}
}
